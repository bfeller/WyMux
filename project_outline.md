**Project Overview:**
Build a custom middleware service in Go that acts as a central orchestrator for a voice assistant pipeline. The service intercepts live audio streams via the Wyoming protocol, handles concurrent Speech-to-Text (STT) and Speaker Identification, logs inference data asynchronously, and conditionally routes the final text payload between Home Assistant and a custom LLM endpoint.

**Deployment Target: Home Assistant Local Add-on (HAOS)**
The service must be packaged as a Home Assistant Local Add-on. Provide the complete file structure, including:
1. `Dockerfile`: To build the proxy environment, install dependencies, and expose the Wyoming TCP port.
2. `config.yaml`: Add-on configuration defining service name, version, ports, architecture, and mapping a persistent storage volume (e.g., `/share/voice_training_data/`) for data logging.
3. `run.sh`: Initialization script to launch the proxy service.
4. The core application source code.

**Core Modules to Implement:**

1. Wyoming Server Interface
- Act as a Wyoming protocol server to accept incoming TCP connections from smart speakers.
- Read multiplexed JSONL metadata and continuous raw PCM audio frames into a rolling RAM buffer.

2. Stream Forker & Buffer Manager
- Fork A (Real-time): Act as a Wyoming client piping live incoming audio frames to an external Whisper STT service. Await the final text transcript.
- Fork B (Buffered): Aggregate a sufficient window of PCM audio in RAM, pass it to a local Biometric Model (e.g., SpeechBrain/ECAPA-TDNN), extract a voice embedding, and return a `Speaker_ID` and `Confidence_Score`.

3. Async Data Logger
- Upon connection close, spawn a non-blocking background process.
- Take the complete PCM audio payload from the RAM buffer, prepend a standard WAV header, and write to the mapped persistent volume.
- Generate and write a corresponding JSON sidecar file containing: `Transcript`, `Speaker_ID`, `Confidence_Score`, and `Routing_Destination`.

4. Routing Engine
- Step 1: Send the STT text transcript to Home Assistant via its REST API (`http://supervisor/core/api/intent/handle`) or WebSocket using a Long-Lived Access Token.
- Step 2: Parse the HA response. If HA successfully handles the intent natively, terminate the flow.
- Step 3: If HA returns an intent failure, construct a JSON payload containing the text `Transcript` and the `Speaker_ID`. Route this payload via HTTP POST to the Custom LLM API endpoint.

**System Constraints:**
- Latency: STT piping and biometric extraction must occur concurrently.
- Disk I/O: Audio must be held entirely in RAM during the active request. Disk writes must be handled strictly asynchronously after routing logic completes to prevent blocking the response.
- Integration: The proxy must expose a TCP port (e.g., 10400) for HA to connect to via the standard Wyoming integration.