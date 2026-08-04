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
