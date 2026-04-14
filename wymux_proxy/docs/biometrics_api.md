# Implementation Guide: Biometric Server API

This document is designed as a prompt/specification guide for a coding agent tasked with building the Speaker Identification Backend.

## System Overview
The WyMux Home Assistant proxy receives audio streams via the Wyoming protocol. Every time a user speaks, it buffers the initial 1-2 seconds of the spoken audio, wraps it in a `.wav` file header, and transmits it via HTTP POST to this Biometric Server. 

The goal of this server is to ingest the audio and analyze it against an embedded machine-learning model (such as `speechbrain/spkrec-ecapa-voxceleb` on Hugging Face) to extract a voice embedding, compare it to pre-enrolled user profiles, and return the identified speaker.

## Expected Contract

Your target service must implement an HTTP Server (e.g., using Python with FastAPI) exposing the following endpoint:

### `POST /identify`

**Authentication:**
WyMux will send an optional Bearer token in the `Authorization` header if the user has configured `biometric_api_key` in their Home Assistant add-on settings. Your server should validate this token if authentication is required.

**Request details:**
*   **Content-Type:** `audio/wav`
*   **Authorization:** `Bearer <biometric_api_key>` *(optional, only present when configured)*
*   **Body:** A raw binary WAV file (`RIFF` header, typically 16kHz, 16-bit, Mono).

**Execution Requirements:**
1. Await the entire binary payload (keep in-memory using `io.BytesIO` or similar, do not write to disk to preserve latency).
2. Load the binary WAV buffer into a processing tensor (e.g., using `torchaudio`).
3. Pass the tensor through the biometric model to retrieve the speaker embedding vectors.
4. Calculate cosine similarity against a small local database/cache of known user embeddings.
5. Identify the user with the highest matching probability.

**Response format (200 OK):**
```json
{
  "speaker_id": "john_doe",
  "confidence_score": 0.965
}
```

*Note: If no user profile crosses a reasonable dynamic confidence threshold (e.g., 0.65), the server should return `"speaker_id": "unknown"`.*
