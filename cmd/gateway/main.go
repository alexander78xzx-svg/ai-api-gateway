package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"aiapigateway/pkg/cache"
	"aiapigateway/pkg/config"
	"aiapigateway/pkg/decoder"
	"aiapigateway/pkg/router"
	"aiapigateway/pkg/truncator"
)

var cfg config.Config

var cacheMem = cache.NewCacheMemory()

func handleApiGateway(w http.ResponseWriter, r *http.Request) {

	// auth
	headerAPIkey := strings.TrimSpace(r.Header.Get("x-api-key"))
	if headerAPIkey != cfg.UserAPIkey {
		http.Error(w, "Invalid API key.", http.StatusUnauthorized)
		return
	}
	// decoder
	var req config.Req

	err := decoder.DecodeRequest(&req, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Exact cache
	hash := cache.HashRequest(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	res, err := cache.ParseCache(hash, cacheMem)
	if err == nil {
		if err = json.NewEncoder(w).Encode(res); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Println("CACHE HIT!")
		return

	}

	// truncate
	truncator.TruncateLogs(&req, &cfg)

	// if task is simple downgrade to a cheaper model
	router.RouteModel(&req, &cfg)

	// temp
	if err := json.NewEncoder(w).Encode(req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// send data and save the cache

	/*
		bodyBytes, err := json.Marshal(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		request, err := http.NewRequest("POST", cfg.targetURL, bytes.NewBuffer(bodyBytes))
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
	*/

}

func main() {
	cfg = config.Config{
		PORT:         "8080",
		TargetURL:    "https://api.anthropic.com/v1/messages",
		UserAPIkey:   os.Getenv("ANTHROPIC_API_KEY"),
		StubMessage:  4,
		TruncateHead: 50,
		TruncateTail: 100,
		CheapModel:   "claude-3-5-haiku-20241022",
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /v1/messages", handleApiGateway)

	server := &http.Server{
		Addr:    ":" + cfg.PORT,
		Handler: mux,
	}

	fmt.Printf("Server up and running on port :%s\n", cfg.PORT)
	server.ListenAndServe()

}
