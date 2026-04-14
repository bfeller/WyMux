# WyMux: Wyoming Protocol Proxy Add-on

WyMux acts as a middleware proxy that orchestrates live Wyoming protocol audio streams. 

## Key Features

1. **Multiplexes Wyoming Streams:** Intercepts incoming audio from network voice satellites.
2. **Concurrent Forking:** Processes transcription via STT (e.g., Whisper) while concurrently buffering for Biometrics (Speaker Identification).
3. **Smart Intent Routing:** Detects when Home Assistant fails to map an intent and gracefully cascades the query to an external Custom LLM.
4. **Data Logging:** Asynchronously saves interaction artifacts (`.wav` and `.json`) into your HA `/share/voice_training_data` folder and conditionally pushes them to a remote storage API.

## Configuration Options

Please map your desired service endpoints within the **Configuration** tab before running:

*   **`stt_whisper_url`**: Network address of your primary STT server. (Default: `tcp://core-whisper:10300` - Official HA Whisper Add-on).
*   **`biometric_server_url`**: HTTP Endpoint (`/identify`) hosting your Biometric service.
*   **`audio_storage_url`**: Remote POST API Endpoint for uploading dataset `.wav`/`.json` payloads.
*   **`custom_llm_url`**: The Fallback LLM API (e.g. Ollama `api/generate`).

## Usage

1. Configure your endpoint options in the Configuration tab and verify the Add-on is **Started**.
2. Go to **Settings** > **Devices & Services** > **Add Integration** > **Wyoming**.
3. Point the integration to: Host `localhost` | Port `10400`.
4. Map your active Voice Assistants/Pipelines to this new integration!
