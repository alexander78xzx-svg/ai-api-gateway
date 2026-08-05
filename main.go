package main

import (
	"bytes"
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

var cache = newCacheMemory()

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
	// Exact cache
	hash, err := hashRequest(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	res, err := parseCache(hash, cache)
	if err == nil {
		if err = json.NewEncoder(w).Encode(res); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return

	}

	// truncate
	if err = truncateLogs(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// send data and save the cache

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	request, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(bodyBytes))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("x-api-key", cfg.userAPIkey)
	request.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(request)

	if err != nil {
		http.Error(w, "Http request to the api failed", http.StatusBadGateway)
		return
	}

	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		http.Error(w, "Error decoding api response", http.StatusBadRequest)
		return
	}

	cache.saveCache(hash, apiResp)

	if err = json.NewEncoder(w).Encode(apiResp); err != nil {
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
