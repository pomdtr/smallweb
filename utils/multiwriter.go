package utils

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// MultiWriter replicates log writes to multiple clients.
type MultiWriter struct {
	mu      sync.Mutex
	clients map[chan []byte]struct{}
	timeout time.Duration
}

// NewMultiWriter creates a new MultiWriter with a specified timeout.
func NewMultiWriter() *MultiWriter {
	return &MultiWriter{
		clients: make(map[chan []byte]struct{}),
		timeout: 5 * time.Second,
	}
}

func (mw *MultiWriter) Write(p []byte) (n int, err error) {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	var wg sync.WaitGroup
	for client := range mw.clients {
		wg.Add(1)
		data := make([]byte, len(p))
		copy(data, p)

		go func(ch chan []byte, data []byte) {
			defer wg.Done()
			select {
			case ch <- data:
			case <-time.After(mw.timeout):
				fmt.Fprintln(os.Stderr, "client timed out")
			}
		}(client, data)
	}
	wg.Wait()

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
