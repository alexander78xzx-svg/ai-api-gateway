package main

import (
	"fmt"
	"net/http"
	"time"
)

/* this is a mock server that is used for testing to see how the gateway
would handle response streaming */

func mockAnthropicHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	// simulate delay
	time.Sleep(700 * time.Millisecond)

	chunks := []string{
		"event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_fake123\",\"role\":\"assistant\",\"content\":[],\"model\":\"claude-3-5-sonnet-20241022\",\"stop_reason\":null,\"stop_sequence\":null,\"usage\":{\"input_tokens\":15,\"output_tokens\":2}}}\n\n",
		"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n",
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"This \"}}\n\n",
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"is \"}}\n\n",
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"a mock response!\"}}\n\n",
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n",
	}

	for _, chunk := range chunks {
		w.Write([]byte(chunk))
		flusher.Flush()
		time.Sleep(100 * time.Millisecond)
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/messages", mockAnthropicHandler)

	fmt.Println("Mock server running on port :8081")
	http.ListenAndServe(":8081", mux)
}
