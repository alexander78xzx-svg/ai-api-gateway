package stream

import (
	"aiapigateway/pkg/config"
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

func SendUpstreamRequest(r *http.Request, req *config.Req, cfg *config.Config) (*http.Response, error) {
	requestBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	upstreamRequest, err := http.NewRequestWithContext(
		r.Context(),
		"POST",
		cfg.TargetURL,
		bytes.NewBuffer(requestBytes),
	)
	if err != nil {
		return nil, err
	}

	// add needed headers
	headers := []string{
		"x-api-key",
		"anthropic-version",
		"anthropic-beta",
		"authorization",
		"user-agent",
	}
	for _, h := range headers {
		if val := r.Header.Get(h); val != "" {
			upstreamRequest.Header.Set(h, val)
		}
	}

	upstreamRequest.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	return client.Do(upstreamRequest)
}

func StreamAndCache(w http.ResponseWriter, resp *http.Response) ([]byte, error) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported by client connection", http.StatusInternalServerError)
		return nil, io.ErrUnexpectedEOF
	}

	w.WriteHeader(resp.StatusCode)

	var responseBuffer bytes.Buffer
	reader := bufio.NewReader(resp.Body)

	// read chunck by chunk
	for {
		chunk, err := reader.ReadBytes('\n') // handle newline sse events
		if len(chunk) > 0 {
			w.Write(chunk)
			flusher.Flush()

			responseBuffer.Write(chunk) // send to bufer for caching
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return responseBuffer.Bytes(), err
		}
	}

	return responseBuffer.Bytes(), nil
}
