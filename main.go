package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

var cfg = Config{
	PORT:         "8080",
	userAPIkey:   os.Getenv("ANTHROPIC_API_KEY"),
	stubMessage:  4,
	truncateHead: 50,
	truncateTail: 100,
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
		Addr:    ":" + cfg.PORT,
		Handler: mux,
	}

	fmt.Printf("Server up and running on port :%s\n", cfg.PORT)
	server.ListenAndServe()

}
