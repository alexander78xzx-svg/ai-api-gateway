package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// json structs

type Message struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"` // content block goes here

	TextContent  string         `json:"-"` // when just a string
	ParsedBlocks []ContentBlock `json:"-"` // when an array
}

type Tool struct {
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Input_schema InputSchema `json:"input_schema"`
}

type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	Content   string          `json:"content,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
}

type Req struct {
	Model      string    `json:"model"`
	Max_tokens int       `json:"max_tokens"`
	System     string    `json:"system"`
	Messages   []Message `json:"messages"`
	Tools      []Tool    `json:"tools"`
}

func decodeRequest(req *Req, r *http.Request) error {

	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return err
	}

	// for each message we check if content is a tool object or just a string
	for i := range req.Messages {
		msg := &req.Messages[i]

		var textMessage string
		if err := json.Unmarshal(msg.Content, &textMessage); err == nil {
			msg.TextContent = textMessage
			continue
		}

		// or

		var blocks []ContentBlock
		if err := json.Unmarshal(msg.Content, &blocks); err == nil {
			msg.ParsedBlocks = blocks
			continue
		}
		// if it didnt parsed both:

		return fmt.Errorf("Failed to parse request's message content.")
	}
	return nil
}
func handleApiGateway(w http.ResponseWriter, r *http.Request) {

	headerAPIkey := r.Header.Get("x-api-key")
	userAPIkey := os.Getenv("ANTHROPIC_API_KEY")
	if headerAPIkey != userAPIkey {
		http.Error(w, "Invalid API key.", http.StatusUnauthorized)
		return
	}

	var req Req
	err := decodeRequest(&req, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err = json.NewEncoder(w).Encode(req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /v1/messages", handleApiGateway)

	server := &http.Server{
		Addr:    ":" + "8080",
		Handler: mux,
	}

	fmt.Print("Server up and running on port :8080\n")
	server.ListenAndServe()

}
