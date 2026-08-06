package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
)

type cacheElement struct {
	hash     string
	response string
}

type cacheMemory struct {
	mu   sync.RWMutex
	data map[string]any
}

func newCacheMemory() *cacheMemory {
	return &cacheMemory{
		data: make(map[string]any),
	}
}

func hashRequest(req *Req) (string, error) {
	data, err := json.Marshal(req) // convert to json bytes
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)             // hash these bytes
	return hex.EncodeToString(hash[:]), nil // convert them into string
}

func parseCache(hash string, m *cacheMemory) (any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.data[hash]
	if ok {
		return m.data[hash], nil
	}
	return "", fmt.Errorf("Cached response not found")
}

func (m *cacheMemory) saveCache(hash string, val any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[hash] = val
}
