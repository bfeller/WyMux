package pipeline

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"wymux/pkg/storage"
	"wymux/pkg/wyoming"
)

// HandleConnection receives a Wyoming client, proxies audio to Whisper for STT,
// optionally runs biometrics, and returns the transcript.
func HandleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	var audioBuffer bytes.Buffer
	var audioChunks [][]byte // Store chunks for forwarding to Whisper
	var transcribeData map[string]interface{}

	for {
		msg, payload, err := wyoming.ReadMessage(reader)
		if err != nil {
			wyoming.DebugLog("Connection closed or error: %v", err)
			break
		}

		if msg == nil || msg.Type == "" {
			continue
		}

		switch msg.Type {
		case "describe":
			wyoming.WriteMessage(conn, wyoming.Msg{
				Type: "info",
				Data: map[string]interface{}{
					"asr": []map[string]interface{}{
						{
							"name":        "wymux",
							"description": "WyMux Middleware Pipeline",
							"attribution": map[string]string{"name": "WyMux", "url": "https://github.com/bfeller/WyMux"},
							"installed":   true,
							"models": []map[string]interface{}{
								{
									"name":        "wymux_proxy",
									"description": "WyMux Proxy",
									"attribution": map[string]string{"name": "WyMux", "url": "https://github.com/bfeller/WyMux"},
									"installed":   true,
									"languages":   []string{"en"},
								},
							},
						},
					},
				},
			}, nil)

		case "transcribe":
			// HA tells us about language/model preferences before streaming audio
			transcribeData = msg.Data
			log.Printf("[PIPELINE] Transcribe request received")

		case "audio-start":
			audioBuffer.Reset()
			audioChunks = nil
			wyoming.DebugLog("[PIPELINE] Audio stream started")

		case "audio-chunk":
			audioBuffer.Write(payload)
			// Store a copy for forwarding to Whisper
			chunk := make([]byte, len(payload))
			copy(chunk, payload)
			audioChunks = append(audioChunks, chunk)

		case "audio-stop":
			log.Printf("[PIPELINE] Audio stream stopped. Total bytes: %d, Chunks: %d", audioBuffer.Len(), len(audioChunks))
			pcmData := make([]byte, audioBuffer.Len())
			copy(pcmData, audioBuffer.Bytes())

			// ==== Fork A: Get transcript from Whisper ====
			transcript := forwardToWhisper(transcribeData, audioChunks)
			log.Printf("[PIPELINE] Transcript from Whisper: %q", transcript)

			// ==== Fork B: Biometrics (optional) ====
			speakerID, confidence := "unknown", 0.0
			biometricURL := os.Getenv("BIOMETRIC_SERVER_URL")
			if biometricURL != "" {
				biometricKey := os.Getenv("BIOMETRIC_API_KEY")
				speakerID, confidence = runBiometrics(pcmData, biometricURL, biometricKey)
				log.Printf("[PIPELINE] Speaker: %s (%.2f)", speakerID, confidence)
			}

			// ==== Send transcript back to HA ====
			// NOTE: HA handles intent routing internally after receiving this transcript.
			// Our job as the ASR service is done once we return the text.
			wyoming.WriteMessage(conn, wyoming.Msg{
				Type: "transcript",
				Data: map[string]interface{}{
					"text": transcript,
				},
			}, nil)

			// ==== Storage (optional, async) ====
			go func() {
				if len(pcmData) > 0 {
					storage.SaveData(pcmData, transcript, speakerID, confidence, "asr_proxy")
				}
			}()

			return

		default:
			wyoming.DebugLog("[DEBUG-WYM] Unhandled message type: %s", msg.Type)
		}
	}
}

// forwardToWhisper opens a Wyoming client connection to the Whisper service,
// sends the audio, and returns the transcribed text.
func forwardToWhisper(transcribeData map[string]interface{}, chunks [][]byte) string {
	whisperURL := os.Getenv("STT_WHISPER_URL")
	if whisperURL == "" {
		log.Println("[WHISPER] No STT_WHISPER_URL configured, returning empty transcript")
		return ""
	}

	// Parse the URL (tcp://core-whisper:10300 -> core-whisper:10300)
	addr := whisperURL
	if strings.HasPrefix(addr, "tcp://") {
		addr = strings.TrimPrefix(addr, "tcp://")
	} else {
		parsed, err := url.Parse(addr)
		if err == nil && parsed.Host != "" {
			addr = parsed.Host
		}
	}

	wyoming.DebugLog("[WHISPER] Connecting to Whisper at %s", addr)
	whisperConn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Printf("[WHISPER] Failed to connect: %v", err)
		return ""
	}
	defer whisperConn.Close()

	whisperReader := bufio.NewReader(whisperConn)

	// Send transcribe event
	if transcribeData == nil {
		transcribeData = map[string]interface{}{"language": "en"}
	}
	wyoming.WriteMessage(whisperConn, wyoming.Msg{
		Type: "transcribe",
		Data: transcribeData,
	}, nil)

	// Send audio-start with format info
	wyoming.WriteMessage(whisperConn, wyoming.Msg{
		Type: "audio-start",
		Data: map[string]interface{}{
			"rate":     16000,
			"width":    2,
			"channels": 1,
		},
	}, nil)

	// Forward all audio chunks
	for _, chunk := range chunks {
		wyoming.WriteMessage(whisperConn, wyoming.Msg{
			Type: "audio-chunk",
			Data: map[string]interface{}{
				"rate":     16000,
				"width":    2,
				"channels": 1,
			},
		}, chunk)
	}

	// Send audio-stop
	wyoming.WriteMessage(whisperConn, wyoming.Msg{
		Type: "audio-stop",
		Data: map[string]interface{}{},
	}, nil)

	wyoming.DebugLog("[WHISPER] All audio forwarded, waiting for transcript...")

	// Read response from Whisper - expect a transcript event
	for {
		msg, _, err := wyoming.ReadMessage(whisperReader)
		if err != nil {
			log.Printf("[WHISPER] Error reading response: %v", err)
			return ""
		}
		if msg == nil || msg.Type == "" {
			continue
		}

		wyoming.DebugLog("[WHISPER] Received event: %s", msg.Type)

		if msg.Type == "transcript" {
			if text, ok := msg.Data["text"].(string); ok {
				return text
			}
			return ""
		}
	}
}

func runBiometrics(pcmData []byte, biometricURL string, apiKey string) (string, float64) {
	wavData := storage.AddWAVHeader(pcmData, 16000, 1, 16)

	req, err := http.NewRequest("POST", biometricURL, bytes.NewReader(wavData))
	if err != nil {
		log.Printf("[BIOMETRICS] Failed to create request: %v", err)
		return "unknown", 0.0
	}
	req.Header.Set("Content-Type", "audio/wav")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[BIOMETRICS] Request failed: %v", err)
		return "unknown", 0.0
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
		id, _ := result["speaker_id"].(string)
		conf, _ := result["confidence_score"].(float64)
		if id != "" {
			return id, conf
		}
	}
	return "unknown", 0.0
}
