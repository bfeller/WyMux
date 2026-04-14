# WyMux: Wyoming Protocol Proxy Add-on

WyMux is a custom Home Assistant Local Add-on built in Go. It acts as a middleware proxy that orchestrates live Wyoming protocol audio streams. 

## Key Features

1. **Multiplexes Wyoming Streams:** intercepts incoming audio from voice satellites (like Wyoming satellites, ESP32 speakers, etc.).
2. **Concurrent Forking:**
   - **Fork A (Real-time):** Forwards audio directly to your local STT engine (like Whisper) for immediate transcript handling.
   - **Fork B (Buffered):** Aggregates a rolling buffer of raw PCM audio and interfaces with an external Biometric Server to identify the speaker (`Speaker_ID`).
3. **Smart Intent Routing:** Routes transcripts into Home Assistant. If HA returns an intent failure (unrecognized intent), it automatically cascades handling to a custom LLM URL mapped in the add-on settings to gracefully handle open conversational requests.
4. **Data Logging:** Generates comprehensive training datasets transparently and asynchronously. Drops `.wav` audio and JSON sidecar files natively into Home Assistant's `/share/voice_training_data` directory, and conditionally mirrors them to a remote Audio Storage API.

## Setup & Installation

### Installation via GitHub Repository

To easily distribute and install WyMux, add this GitHub repository to your Home Assistant Add-on Store.

1. In Home Assistant, navigate to **Settings** > **Add-ons**.
2. Click the **Add-on Store** button in the bottom right corner.
3. Click the three dots (menu) in the top right corner and select **Repositories**.
4. Paste the URL of this GitHub repository into the field and click **Add**.
5. Close the dialog and scroll through your Add-on store (or hit **Check for updates** in the menu).
6. Locate the **WyMux Proxy** card under the new repository section.
7. Click the card and select **Install**. This will automatically build the `Dockerfile` into a container on your Home Assistant OS device.

## Configuration Instructions

Once installed, navigate to the **Configuration** tab of the `WyMux` add-on page to adjust the core URLs for the endpoints:

*   **`stt_whisper_url`**: The URL directing to your running Whisper server handling STT text translation (e.g. `tcp://core-whisper:10300`).
*   **`biometric_server_url`**: The HTTP URL hosting your Speaker ID extraction model.
*   **`audio_storage_url`**: (Optional) URL for the remote server that receives multipart file uploads for saving historical datasets.
*   **`custom_llm_url`**: REST API URL where WyMux forwards transcripts that Home Assistant fails to action natively.

Click **Save** and then click **Start** in the Info tab!

## Usage

1. By default, WyMux listens locally on TCP port `10400`.
2. To use it, simply configure your Home Assistant **Wyoming Integration**.
3. Go to **Settings** > **Devices & Services** > **Add Integration** > **Wyoming**.
4. Enter the physical IP address of your Home Assistant box (e.g., `192.168.1.X`) as the Host and `10400` as the Port. DO NOT use `localhost` because the integration runs inside a separate container and `localhost` resolves to the core container instead of the add-on.
5. Your voice assistant pipeline in Home Assistant will now utilize the `WyMux` integration as its engine, routing all pipeline audio seamlessly through our multiplexer proxy!
