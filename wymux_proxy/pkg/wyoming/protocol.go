package wyoming

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
)

// Msg represents a Wyoming protocol JSON message.
// Wyoming v1.7+ flattens all event data fields into the top-level JSON object
// alongside "type" and "payload_length", rather than nesting under "data".
type Msg struct {
	Type       string
	Data       map[string]interface{}
	PayloadLen int
}

// MarshalJSON flattens the Msg into a single-level JSON object:
// {"type":"...", "payload_length": N, ...data fields...}
func (m Msg) MarshalJSON() ([]byte, error) {
	out := make(map[string]interface{})
	out["type"] = m.Type
	if m.PayloadLen > 0 {
		out["payload_length"] = m.PayloadLen
	}
	for k, v := range m.Data {
		out[k] = v
	}
	return json.Marshal(out)
}

// UnmarshalJSON reads a flat Wyoming JSON object:
// extracts "type", "payload_length", and puts everything else into Data.
func (m *Msg) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	if t, ok := raw["type"].(string); ok {
		m.Type = t
	}
	delete(raw, "type")

	if pl, ok := raw["payload_length"].(float64); ok {
		m.PayloadLen = int(pl)
	}
	delete(raw, "payload_length")

	m.Data = raw
	return nil
}

// ReadMessage returns the JSON Msg and any trailing payload (PCM data).
func ReadMessage(r *bufio.Reader) (*Msg, []byte, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, nil, err
	}

	line = bytes.TrimSpace(line)

	if len(line) > 0 {
		log.Printf("[DEBUG-WYM] Received raw line: %s", string(line))
	}

	if len(line) == 0 {
		// Treat empty line as OK, return empty msg and no payload
		return &Msg{}, nil, nil
	}

	var msg Msg
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

	log.Printf("[DEBUG-WYM] Sending: %s", string(j))

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
