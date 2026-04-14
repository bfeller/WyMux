#!/usr/bin/env bashio

bashio::log.info "Starting WyMux Proxy Service..."

# Check config options
export STT_WHISPER_URL=$(bashio::config 'stt_whisper_url')
export BIOMETRIC_SERVER_URL=$(bashio::config 'biometric_server_url')
export AUDIO_STORAGE_URL=$(bashio::config 'audio_storage_url')
export CUSTOM_LLM_URL=$(bashio::config 'custom_llm_url')

bashio::log.info "STT Endpoint: $STT_WHISPER_URL"

# The SUPERVISOR_TOKEN is automatically set in the environment by Home Assistant

exec /app/wymux
