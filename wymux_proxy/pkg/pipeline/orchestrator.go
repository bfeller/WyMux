package pipeline

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"

	"wymux/pkg/routing"
	"wymux/pkg/storage"
	"wymux/pkg/wyoming"
)

// HandleConnection receives a Wyoming client, aggregates audio, forks it for STT and Identification.
func HandleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	var audioBuffer bytes.Buffer

	for {
		msg, payload, err := wyoming.ReadMessage(reader)
		if err != nil {
			log.Printf("Connection closed or error: %v", err)
			break
		}

		if msg == nil || msg.Type == "" {
			continue // empty keep-alive
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
		
		default:
			log.Printf("[DEBUG-WYM] Unhandled message type: %s", msg.Type)

		case "audio-start":
			audioBuffer.Reset()

		case "audio-chunk":
			audioBuffer.Write(payload)
			// TODO: Fork A: In parallel, stream this payload directly to STT service via TCP client.

		case "audio-stop":
			pcmData := audioBuffer.Bytes()

			// Fork B: Send sufficiently pooled data to Biometrics endpoint (WAV wrapped usually, using PCM for now)
			speakerID, confidence := runBiometrics(pcmData)

			// Execute mock STT (in full implementation, this awaits the stream done from Fork A)
			transcript := "simulated transcript text"

			// Route Intent
			routed, err := routing.HandleIntent(transcript)
			if err != nil || !routed {
				routing.FallbackLLM(transcript, speakerID)
			}

			// Background task to save audio and metadata
			go storage.SaveData(pcmData, transcript, speakerID, confidence, "completed_route")

			// Return response to HA client
			// This tells HA the intent was handled and the pipeline is over.
			wyoming.WriteMessage(conn, wyoming.Msg{Type: "run-pipeline-ended"}, nil)
			return
		}
	}
}

func runBiometrics(pcmData []byte) (string, float64) {
	url := os.Getenv("BIOMETRIC_SERVER_URL")
	if url == "" {
		url = "http://localhost:8000/identify"
	}

	// We append a simple WAV header for 16kHz, 16-bit, Mono in the actual implementation, 
	// assuming raw PCM for brevity unless specifically tested by the remote server.
	wavData := storage.AddWAVHeader(pcmData, 16000, 1, 16)

	req, err := http.NewRequest("POST", url, bytes.NewReader(wavData))
	if err != nil {
		log.Printf("Failed to create biometric request: %v", err)
		return "unknown", 0.0
	}
	req.Header.Set("Content-Type", "audio/wav")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Biometrics request failed: %v", err)
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
