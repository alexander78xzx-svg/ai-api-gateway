package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"aiapigateway/pkg/cache"
	"aiapigateway/pkg/config"
	"aiapigateway/pkg/decoder"
	"aiapigateway/pkg/router"
	"aiapigateway/pkg/security"
	"aiapigateway/pkg/stream"
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

	// strip private data
	security.RedactPayload(&req)

	// truncate
	truncator.TruncateLogs(&req, &cfg)

	// if task is simple downgrade to a cheaper model
	router.RouteModel(&req, &cfg)

	// we normalized data, now comes Exact cache
	hash := cache.HashRequest(&req)

	res, err := cache.ParseCache(hash, cacheMem)
	if err == nil {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Cache", "HIT")
		w.Write(res)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		return
	}

	// send data and save the cache

	resp, err := stream.SendUpstreamRequest(r, &req, &cfg)
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("X-Cache", "MISS")

	fullResponseBytes, err := stream.StreamAndCache(w, resp)

	if err == nil && len(fullResponseBytes) > 0 {
		go func(key string, data []byte) {
			cacheMem.SaveCache(key, data)
		}(hash, fullResponseBytes)
	}
}

func main() {
	cfg = config.Config{
		PORT:          "8080",
		TargetURL:     "http://localhost:8081/v1/messages", // "https://api.anthropic.com/v1/messages"
		UserAPIkey:    os.Getenv("ANTHROPIC_API_KEY"),
		StubMessage:   4,
		TruncateHead:  50,
		TruncateTail:  100,
		CheapModel:    "claude-haiku-4-5-20251001",
		RetryAttempts: 3,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /v1/messages", handleApiGateway)
	mux.HandleFunc("POST /v1/chat/completions", handleApiGateway)

	server := &http.Server{
		Addr:    ":" + cfg.PORT,
		Handler: mux,
	}

	fmt.Printf("Server up and running on port :%s\n", cfg.PORT)
	server.ListenAndServe()

}
