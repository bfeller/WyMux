package wyoming

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
)

// Msg represents a standard Wyoming protocol JSON message
type Msg struct {
	Type       string                 `json:"type"`
	Data       map[string]interface{} `json:"data,omitempty"`
	PayloadLen int                    `json:"payload_length,omitempty"`
}

// ReadMessage returns the JSON Msg and any trailing payload (PCM data).
func ReadMessage(r *bufio.Reader) (*Msg, []byte, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, nil, err
	}

	var msg Msg
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		// Treat empty line as OK, return empty msg and no payload
		return &Msg{}, nil, nil
	}

	if err := json.Unmarshal(line, &msg); err != nil {
		log.Printf("Failed to parse Wyoming message: %s", string(line))
		return nil, nil, err
	}

	var payload []byte
	if msg.PayloadLen > 0 {
		payload = make([]byte, msg.PayloadLen)
		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, nil, err
		}
	}

	return &msg, payload, nil
}

// WriteMessage sends a Wyoming protocol JSON message and an optional binary payload.
func WriteMessage(w io.Writer, msg Msg, payload []byte) error {
	msg.PayloadLen = len(payload)
	j, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	j = append(j, '\n')

	if _, err := w.Write(j); err != nil {
		return err
	}

	if len(payload) > 0 {
		if _, err := w.Write(payload); err != nil {
			return err
		}
	}

	return nil
}
