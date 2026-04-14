# Implementation Guide: Audio Storage Server API

This document is designed as a prompt/specification guide for a coding agent tasked with building the Remote Audio Storage Backend.

## System Overview
The WyMux Home Assistant proxy routinely intercepts voice commands, generates transcripts, identifies speakers, and calculates logical intent routes. After pipeline execution, it compiles this telemetry into a single interaction packet (a `.wav` audio file and a `.json` sidecar file).

Your objective is to implement the ingestion endpoint (e.g., in Node.js Express, Python FastAPI, or Golang) to receive, validate, and archive these multipart sessions into a scalable object storage system (like an S3 bucket with a PostgreSQL mapping index) for future model fine-tuning.

## Expected Contract

### `POST /upload`

**Authentication:**
WyMux will send an optional Bearer token in the `Authorization` header if the user has configured `audio_storage_api_key` in their Home Assistant add-on settings. Your server should validate this token if authentication is required.

**Request details:**
*   **Content-Type:** `multipart/form-data`
*   **Authorization:** `Bearer <audio_storage_api_key>` *(optional, only present when configured)*
*   **Form fields:**
    *   `audio`: A file byte stream representing the `.wav` interaction. 
    *   `metadata`: A file byte stream parsing out as a `.json` document with telemetry data.

**JSON Metadata Target Schema:**
The `metadata` file stream will unpack into the following schema context:
```json
{
  "transcript": "turn off the living room lights",
  "speaker_id": "john_doe",
  "confidence_score": 0.94,
  "routing_destination": "home_assistant",
  "timestamp": "20260414_150405"
}
```

**Execution Requirements:**
1. Read the HTTP Multipart Form boundaries synchronously.
2. Read the `metadata` JSON to generate a logical primary key tagging for the interaction.
3. Upload the `audio` file buffer to persistence (e.g. AWS S3). You should logically partition the file paths by `speaker_id` for compliance and accessibility.
4. Write the JSON metadata parameters to a searchable database system that retains a foreign URL/URI reference to where the `audio` blob was persisted.
5. Only return `200 OK` once both database and object storage workflows settle.

**Response format:**
*   **Success:** HTTP `200 OK` (Body is arbitrary, a simple `{"status": "success"}` response is fine).
*   **Failure:** `400 Bad Request` (Invalid multipart) or `500 Internal Server Error` (If S3/Database routines fail). 
