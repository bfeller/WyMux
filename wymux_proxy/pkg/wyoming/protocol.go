package wyoming

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
)

var debugEnabled = false

func init() {
	debugEnabled = os.Getenv("DEBUG_LOGGING") == "true"
}

// DebugLog prints a log message only when debug logging is enabled.
func DebugLog(format string, v ...interface{}) {
	if debugEnabled {
		log.Printf(format, v...)
	}
}

// Msg represents a Wyoming protocol event.
type Msg struct {
	Type       string
	Data       map[string]interface{}
	PayloadLen int
}

// ReadMessage reads a Wyoming protocol event from the reader.
func ReadMessage(r *bufio.Reader) (*Msg, []byte, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, nil, err
	}

	line = bytes.TrimSpace(line)

	if len(line) > 0 {
		DebugLog("[DEBUG-WYM] Received raw line: %s", string(line))
	}

	if len(line) == 0 {
		return &Msg{}, nil, nil
	}

	// Parse the JSON header line
	var header map[string]interface{}
	if err := json.Unmarshal(line, &header); err != nil {
		log.Printf("Failed to parse Wyoming message: %s", string(line))
		return nil, nil, err
	}

	msg := &Msg{}

	if t, ok := header["type"].(string); ok {
		msg.Type = t
	}

	// Get inline data if present
	data := make(map[string]interface{})
	if d, ok := header["data"].(map[string]interface{}); ok {
		data = d
	}

	// Handle external data_length: read additional JSON bytes after the header line
	if dl, ok := header["data_length"].(float64); ok && int(dl) > 0 {
		dataBytes := make([]byte, int(dl))
		if _, err := io.ReadFull(r, dataBytes); err != nil {
			return nil, nil, err
		}
		var externalData map[string]interface{}
		if err := json.Unmarshal(dataBytes, &externalData); err == nil {
			for k, v := range externalData {
				data[k] = v
			}
		}
	}

	msg.Data = data

	// Handle binary payload
	if pl, ok := header["payload_length"].(float64); ok {
		msg.PayloadLen = int(pl)
	}

	var payload []byte
	if msg.PayloadLen > 0 {
		payload = make([]byte, msg.PayloadLen)
		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, nil, err
		}
	}

	return msg, payload, nil
}

// WriteMessage sends a Wyoming protocol event.
func WriteMessage(w io.Writer, msg Msg, payload []byte) error {
	// Build the header
	header := map[string]interface{}{
		"type":    msg.Type,
		"version": "1.7.2",
	}

	// Serialize data separately (Wyoming wire format)
	var dataBytes []byte
	if len(msg.Data) > 0 {
		var err error
		dataBytes, err = json.Marshal(msg.Data)
		if err != nil {
			return err
		}
		header["data_length"] = len(dataBytes)
	}

	if len(payload) > 0 {
		header["payload_length"] = len(payload)
	}

	// Write header JSON line
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return err
	}

	DebugLog("[DEBUG-WYM] Sending header: %s", string(headerJSON))
	if dataBytes != nil {
		DebugLog("[DEBUG-WYM] Sending data: %s", string(dataBytes))
	}

	headerJSON = append(headerJSON, '\n')
	if _, err := w.Write(headerJSON); err != nil {
		return err
	}

	// Write data bytes
	if dataBytes != nil {
		if _, err := w.Write(dataBytes); err != nil {
			return err
		}
	}

	// Write binary payload
	if len(payload) > 0 {
		if _, err := w.Write(payload); err != nil {
			return err
		}
	}

	return nil
}
