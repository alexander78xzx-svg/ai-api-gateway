package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sync"

	"aiapigateway/pkg/config"
)

var (
	hexRegex  = regexp.MustCompile(`\b0x[a-fA-F0-9]+\b`)
	uuidRegex = regexp.MustCompile(`\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`)
	dateRegex = regexp.MustCompile(`\b\d{4}[-/]\d{2}[-/]\d{2}(T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?)?\b`)
	timeRegex = regexp.MustCompile(`\b\d{2}:\d{2}:\d{2}(\.\d+)?\b`)
	ipRegex   = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	pidRegex  = regexp.MustCompile(`(?i)\bpid[\s:=]*\d+\b`)
)

type cacheElement struct {
	hash     string
	response string
}

type cacheMemory struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewCacheMemory() *cacheMemory {
	return &cacheMemory{
		data: make(map[string][]byte),
	}
}

func normalizeLogEntropy(text string) string {
	s := uuidRegex.ReplaceAllString(text, "<UUID>")
	s = dateRegex.ReplaceAllString(s, "<DATE>")
	s = timeRegex.ReplaceAllString(s, "<TIME>")
	s = ipRegex.ReplaceAllString(s, "<IP>")
	s = hexRegex.ReplaceAllString(s, "<HEX>")
	s = pidRegex.ReplaceAllString(s, "<PID>")
	return s
}

func HashRequest(req *config.Req) string {
	h := sha256.New()

	h.Write([]byte(req.Model))

	for _, msg := range req.Messages {
		h.Write([]byte(msg.Role))
		switch content := msg.Content.(type) {
		case string:

			h.Write([]byte(normalizeLogEntropy(content)))
		case []config.ContentBlock:

			for _, cb := range content {
				h.Write([]byte(cb.Type))
				h.Write([]byte(normalizeLogEntropy(cb.Text)))
			}
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

func ParseCache(hash string, m *cacheMemory) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.data[hash]
	if ok {
		return m.data[hash], nil
	}
	return []byte(""), fmt.Errorf("Cached response not found")
}

func (m *cacheMemory) SaveCache(hash string, val []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[hash] = val
}
