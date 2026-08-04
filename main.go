package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// json structs

type Message struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"` // content block goes here, remains raw until the response back , we will inject one of the two next vars

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

// req.message.content:
type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`

	ContentString string         `json:"-"` // if content is a string
	ContentBlocks []ContentBlock `json:"-"` // if content is an array
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

func truncateLogs(req *Req) error {
	for i := range req.Messages {
		// we search for the tool output made by the user role
		if req.Messages[i].Role == "user" && len(req.Messages[i].ParsedBlocks) > 0 {

			// we check if the message is older than n turns
			if len(req.Messages) > 4 && i < len(req.Messages)-4 {

				for j := range req.Messages[i].ParsedBlocks {
					if req.Messages[i].ParsedBlocks[j].Type != "tool_result" {
						continue
					}

					// check if tool result is a string
					trim := bytes.TrimSpace(req.Messages[i].ParsedBlocks[j].Content)
					if len(trim) == 0 {
						continue
					}
					switch trim[0] {
					case '"':
						var s string
						if err := json.Unmarshal(req.Messages[i].ParsedBlocks[j].Content, &s); err != nil {
							continue
						}

						s = fmt.Sprintf("Stubbed output, %d chars removed", len(s))
						t, err := json.Marshal(s)
						if err != nil {
							return err
						}

						req.Messages[i].ParsedBlocks[j].Content = json.RawMessage(t)
					case '[':
						var block []map[string]any
						err := json.Unmarshal(trim, &block)
						if err != nil {

							// fallback, if the content is other that strings, we just count raw json bytes
							logLen := len(trim)
							s := fmt.Sprintf("Stubbed output, %d chars removed", logLen)
							t, _ := json.Marshal(s)
							req.Messages[i].ParsedBlocks[j].Content = json.RawMessage(t)
							continue
						}
						var logLen int
						for _, b := range block {
							if text, ok := b["text"].(string); ok {
								logLen += len(text)
							}
						}
						if logLen > 0 {
							s := fmt.Sprintf("Stubbed output, %d chars removed", logLen)
							t, err := json.Marshal(s)
							if err != nil {
								return err

							}

							req.Messages[i].ParsedBlocks[j].Content = json.RawMessage(t)
						}
					}
				}
			} else {
				// message isnt older than n turns (needs to be preserved)

				// each parsed block:
				for j := range req.Messages[i].ParsedBlocks {

					if req.Messages[i].ParsedBlocks[j].Type != "tool_result" {
						continue
					}

					// check if tool result is a string
					trim := bytes.TrimSpace(req.Messages[i].ParsedBlocks[j].Content)
					if len(trim) == 0 {
						continue
					}

					switch trim[0] {
					case '"':
						var s string
						if err := json.Unmarshal(req.Messages[i].ParsedBlocks[j].Content, &s); err != nil {
							continue
						}

						parts := strings.Split(s, "\n")

						// truncate and leave only first 50 and last 100 lines, if enough lines
						if len(parts) > 150 {

							var temp []string
							head := parts[:50]
							tail := parts[len(parts)-100:]
							text := fmt.Sprintf("[%d lines truncated]", len(parts)-150) // add a note

							temp = append(temp, head...)
							temp = append(temp, text)
							temp = append(temp, tail...)

							// readd an \\n at the end of each log
							truncated := strings.Join(temp, "\n")

							t, err := json.Marshal(truncated)
							if err != nil {
								return err
							}

							req.Messages[i].ParsedBlocks[j].Content = json.RawMessage(t)
						}
					case '[':
						continue
					}

				}
			}

			// we convert parsed blocks back to content
			p, err := json.Marshal(req.Messages[i].ParsedBlocks)
			if err != nil {
				return err
			}
			req.Messages[i].Content = json.RawMessage(p)
		}

	}
	return nil
}

func handleApiGateway(w http.ResponseWriter, r *http.Request) {

	// auth
	headerAPIkey := r.Header.Get("x-api-key")
	userAPIkey := os.Getenv("ANTHROPIC_API_KEY")
	if headerAPIkey != userAPIkey {
		http.Error(w, "Invalid API key.", http.StatusUnauthorized)
		return
	}
	// decoder
	var req Req

	err := decodeRequest(&req, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// truncate
	if err = truncateLogs(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
