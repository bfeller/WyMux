package routing

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
)

// HandleIntent sends the transcript to HA's intent handler, returns true if handled
func HandleIntent(transcript string) (bool, error) {
	// Assume http://supervisor/core/api/intent/handle inside HA Addon environment
	haUrl := "http://supervisor/core/api/intent/handle"
	token := os.Getenv("SUPERVISOR_TOKEN")
	if token == "" {
		log.Println("No SUPERVISOR_TOKEN, skipping HA intent route")
		return false, nil
	}

	payload := map[string]interface{}{
		"name": "HassTurnOn", // In a real scenario, this requires structured text for the conversational agent API
		"data": map[string]string{
			"text": transcript,
		},
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", haUrl, bytes.NewBuffer(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200, nil
}

// FallbackLLM routes the text transcript and speaker identity to a custom backend LLM compatible with OpenAI endpoints.
func FallbackLLM(transcript string, speakerID string) {
	llmUrl := os.Getenv("CUSTOM_LLM_URL")
	if llmUrl == "" {
		llmUrl = "https://api.openai.com/v1/chat/completions"
	}

	model := os.Getenv("CUSTOM_LLM_MODEL")
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	key := os.Getenv("CUSTOM_LLM_API_KEY")

	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are a smart home agent talking to " + speakerID,
			},
			{
				"role":    "user",
				"content": transcript,
			},
		},
		"temperature": 0.7,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", llmUrl, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to reach Fallback LLM: %v", err)
		return
	}
	defer resp.Body.Close()

	log.Printf("Successfully routed unhandled intent to Fallback LLM (%s) for speaker: %s", model, speakerID)
}
