package utils

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"log/slog"
)

type (
	// struct for holding response details
	responseData struct {
		Status  int     `json:"status"`
		Bytes   int     `json:"bytes"`
		Elapsed float64 `json:"elapsed"`
	}

	// our http.ResponseWriter implementation
	loggingResponseWriter struct {
		http.ResponseWriter // compose original http.ResponseWriter
		responseData        *responseData
	}

	// Request information
	requestInfo struct {
		URL     string            `json:"url"`
		Host    string            `json:"host"`
		Method  string            `json:"method"`
		Path    string            `json:"path"`
		Headers map[string]string `json:"header"`
	}

	// Log entry structure
	logEntry struct {
		HTTPRequest  requestInfo  `json:"httpRequest"`
		HTTPResponse responseData `json:"httpResponse"`
	}

	Logger struct {
		*slog.Logger
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.Bytes += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.Status = statusCode
}

// Implement http.Flusher interface
func (r *loggingResponseWriter) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Implement http.Hijacker interface
func (r *loggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := r.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Implement http.Pusher interface
func (r *loggingResponseWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := r.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

func NewLogger(writer io.Writer) *Logger {
	return &Logger{
		Logger: slog.New(slog.NewJSONHandler(writer, nil)),
	}
}

func (l *Logger) Middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responseData := &responseData{
			Status: 200,
			Bytes:  0,
		}
		lrw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}

		h.ServeHTTP(&lrw, r)

		duration := time.Since(start)
		responseData.Elapsed = duration.Seconds()

		// Prepare request information
		reqInfo := requestInfo{
			URL:     fmt.Sprintf("https://%s%s", r.Host, r.URL.Path),
			Host:    r.Host,
			Method:  r.Method,
			Path:    r.URL.Path,
			Headers: make(map[string]string),
		}

		// Capture request headers
		for k, v := range r.Header {
			reqInfo.Headers[k] = v[0]
		}

		// Mask sensitive data in headers
		if _, ok := reqInfo.Headers["Cookie"]; ok {
			reqInfo.Headers["Cookie"] = "***"
		}

		if _, ok := reqInfo.Headers["Authorization"]; ok {
			reqInfo.Headers["Authorization"] = "***"
		}

		logEntry := logEntry{
			HTTPRequest:  reqInfo,
			HTTPResponse: *responseData,
		}

		// Use slog to log the entry
		l.LogAttrs(r.Context(), slog.LevelInfo, fmt.Sprintf("Response: %d %s", responseData.Status, http.StatusText(responseData.Status)),
			slog.String("type", "http"),
			slog.Any("request", logEntry.HTTPRequest),
			slog.Any("response", logEntry.HTTPResponse),
		)
	})
}
