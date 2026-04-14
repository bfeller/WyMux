#!/usr/bin/env bash

echo "[INFO] Starting WyMux Proxy Service..."

echo "[INFO] --- DIAGNOSTICS: OPTIONS.JSON ---"
if [ -f /data/options.json ]; then
    echo "[INFO] File exists. Contents:"
    cat /data/options.json
    echo ""
else
    echo "[ERROR] /data/options.json DOES NOT EXIST!"
fi
echo "[INFO] ---------------------------------"

# Read config directly from options.json using jq
export STT_WHISPER_URL=$(jq -r '.stt_whisper_url // empty' /data/options.json)
export BIOMETRIC_SERVER_URL=$(jq -r '.biometric_server_url // empty' /data/options.json)
export AUDIO_STORAGE_URL=$(jq -r '.audio_storage_url // empty' /data/options.json)
export CUSTOM_LLM_URL=$(jq -r '.custom_llm_url // empty' /data/options.json)
export CUSTOM_LLM_API_KEY=$(jq -r '.custom_llm_api_key // empty' /data/options.json)
export CUSTOM_LLM_MODEL=$(jq -r '.custom_llm_model // empty' /data/options.json)

echo "[INFO] STT Endpoint: $STT_WHISPER_URL"
echo "[INFO] LLM Endpoint: $CUSTOM_LLM_URL"
echo "[INFO] LLM Model: $CUSTOM_LLM_MODEL"

# The SUPERVISOR_TOKEN is automatically set in the environment by Home Assistant

exec /app/wymux
