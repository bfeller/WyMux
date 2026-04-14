#!/usr/bin/env bashio

bashio::log.info "Starting WyMux Proxy Service..."

bashio::log.info "--- DIAGNOSTICS: OPTIONS.JSON ---"
if [ -f /data/options.json ]; then
    bashio::log.info "File exists. Contents:"
    cat /data/options.json
    echo ""
else
    bashio::log.error "/data/options.json DOES NOT EXIST! User has not hit 'Save' or Supervisor failed to write it."
fi
bashio::log.info "---------------------------------"

# Check config options
export STT_WHISPER_URL=$(bashio::config 'stt_whisper_url')
export BIOMETRIC_SERVER_URL=$(bashio::config 'biometric_server_url')
export AUDIO_STORAGE_URL=$(bashio::config 'audio_storage_url')
export CUSTOM_LLM_URL=$(bashio::config 'custom_llm_url')
export CUSTOM_LLM_API_KEY=$(bashio::config 'custom_llm_api_key')
export CUSTOM_LLM_MODEL=$(bashio::config 'custom_llm_model')

bashio::log.info "STT Endpoint: $STT_WHISPER_URL"

# The SUPERVISOR_TOKEN is automatically set in the environment by Home Assistant

exec /app/wymux
