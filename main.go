package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// json structs

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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

type Req struct {
	Model      string    `json:"model"`
	Max_tokens int       `json:"max_tokens"`
	System     string    `json:"system"`
	Messages   []Message `json:"messages"`
	Tools      []Tool    `json:"tools"`
}

func handleApiGateway(w http.ResponseWriter, r *http.Request) {

	headerAPIkey := r.Header.Get("x-api-key")
	userAPIkey := os.Getenv("ANTHROPIC_API_KEY")
	if headerAPIkey != userAPIkey {
		http.Error(w, "Invalid API key.", http.StatusBadRequest)
		return
	}

	var req Req
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Couldnt decode", http.StatusInternalServerError)
		return
	}

	fmt.Print(req.Model)

	w.WriteHeader(http.StatusOK)

}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /v1/messages", handleApiGateway)

	server := &http.Server{
		Addr:    ":" + "8080",
		Handler: mux,
	}
	server.ListenAndServe()
	fmt.Print("Server up and running on port :8080")
}
