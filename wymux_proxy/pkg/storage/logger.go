package storage

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// AddWAVHeader prepends a standard RIFF WAV header to raw PCM data.
func AddWAVHeader(pcmData []byte, sampleRate int, numChannels int, bitsPerSample int) []byte {
	blockAlign := numChannels * bitsPerSample / 8
	byteRate := sampleRate * blockAlign
	dataSize := len(pcmData)

	buf := new(bytes.Buffer)

	// RIFF chunk descriptor
	buf.WriteString("RIFF")
	binary.Write(buf, binary.LittleEndian, uint32(36+dataSize))
	buf.WriteString("WAVE")

	// fmt sub-chunk
	buf.WriteString("fmt ")
	binary.Write(buf, binary.LittleEndian, uint32(16)) // Subchunk1Size for PCM
	binary.Write(buf, binary.LittleEndian, uint16(1))  // AudioFormat (1 = PCM)
	binary.Write(buf, binary.LittleEndian, uint16(numChannels))
	binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	binary.Write(buf, binary.LittleEndian, uint32(byteRate))
	binary.Write(buf, binary.LittleEndian, uint16(blockAlign))
	binary.Write(buf, binary.LittleEndian, uint16(bitsPerSample))

	// data sub-chunk
	buf.WriteString("data")
	binary.Write(buf, binary.LittleEndian, uint32(dataSize))

	buf.Write(pcmData)

	return buf.Bytes()
}

// SaveData saves the WAV and JSON payload locally and uploads to the Audio Storage Server.
func SaveData(pcmData []byte, transcript, speakerID string, confidence float64, routing string) {
	timestamp := time.Now().Format("20060102_150405")
	baseName := fmt.Sprintf("wymux_%s_%s", speakerID, timestamp)

	wavData := AddWAVHeader(pcmData, 16000, 1, 16)
	
	metaData := map[string]interface{}{
		"transcript":          transcript,
		"speaker_id":          speakerID,
		"confidence_score":    confidence,
		"routing_destination": routing,
		"timestamp":           timestamp,
	}
	metaBytes, _ := json.MarshalIndent(metaData, "", "  ")

	// Write to mapped generic volume
	shareDir := "/share/voice_training_data"
	os.MkdirAll(shareDir, 0755)

	wavPath := filepath.Join(shareDir, baseName+".wav")
	os.WriteFile(wavPath, wavData, 0644)

	jsonPath := filepath.Join(shareDir, baseName+".json")
	os.WriteFile(jsonPath, metaBytes, 0644)

	log.Printf("Saved session internally to %s", shareDir)

	// Forward to Remote Audio File Storage Server
	uploadURL := os.Getenv("AUDIO_STORAGE_URL")
	if uploadURL == "" {
		return
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	audioPart, _ := writer.CreateFormFile("audio", baseName+".wav")
	audioPart.Write(wavData)

	metaPart, _ := writer.CreateFormFile("metadata", baseName+".json")
	metaPart.Write(metaBytes)

	writer.Close()

	req, _ := http.NewRequest("POST", uploadURL, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to upload session to remote storage: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		log.Printf("Successfully uploaded session to remote storage")
	} else {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("Upload failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}
}
