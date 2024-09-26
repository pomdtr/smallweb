package utils

import (
	"sync"
)

// MultiWriter replicates log writes to multiple clients.
type MultiWriter struct {
	mu      sync.Mutex
	clients map[chan []byte]struct{}
}

// NewMultiWriter creates a new MultiWriter.
func NewMultiWriter() *MultiWriter {
	return &MultiWriter{
		clients: make(map[chan []byte]struct{}),
	}
}

// Write sends log data to all connected clients.
func (mw *MultiWriter) Write(p []byte) (n int, err error) {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	for client := range mw.clients {
		select {
		case client <- p:
		default:
			// Drop the message if the client is not ready (avoiding blocking)
		}
	}
	return len(p), nil
}

// AddClient registers a new client channel to receive logs.
func (mw *MultiWriter) AddClient(ch chan []byte) {
	mw.mu.Lock()
	defer mw.mu.Unlock()
	mw.clients[ch] = struct{}{}
}

// RemoveClient unregisters a client channel.
func (mw *MultiWriter) RemoveClient(ch chan []byte) {
	mw.mu.Lock()
	defer mw.mu.Unlock()
	delete(mw.clients, ch)
	close(ch)
}
