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
	// our http.ResponseWriter implementation
	loggingResponseWriter struct {
		http.ResponseWriter // compose original http.ResponseWriter
		Status              int
		Size                int
	}

	HttpLog struct {
		Time    time.Time `json:"time"`
		Level   string    `json:"level"`
		Msg     string    `json:"msg"`
		Request struct {
			Url     string            `json:"url"`
			Host    string            `json:"host"`
			Method  string            `json:"method"`
			Path    string            `json:"path"`
			Headers map[string]string `json:"headers"`
		} `json:"request"`
		Response struct {
			Status  int     `json:"status"`
			Bytes   int     `json:"bytes"`
			Elapsed float64 `json:"elapsed"`
		} `json:"response"`
	}

	Logger struct {
		*slog.Logger
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.Size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.Status = statusCode
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

		lrw := loggingResponseWriter{
			ResponseWriter: w,
		}

		h.ServeHTTP(&lrw, r)

		if lrw.Status == 0 {
			lrw.Status = http.StatusOK
		}

		duration := time.Since(start)

		// Mask sensitive data in headers
		if _, ok := r.Header["Cookie"]; ok {
			r.Header["Cookie"] = []string{"***"}
		}

		if _, ok := r.Header["Authorization"]; ok {
			r.Header["Authorization"] = []string{"***"}
		}

		var headers []any
		for k, v := range r.Header {
			headers = append(headers, slog.String(k, v[0]))
		}

		// Use slog to log the entry
		l.LogAttrs(r.Context(), slog.LevelInfo, fmt.Sprintf("Response: %d %s", lrw.Status, http.StatusText(lrw.Status)),
			slog.Group("request", slog.String("url", fmt.Sprintf("https://%s%s", r.Host, r.URL.String())), slog.String("host", r.Host), slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.Group("headers", headers...)),
			slog.Group("response", slog.Int("status", lrw.Status), slog.Int("bytes", lrw.Size), slog.Float64("elapsed", duration.Seconds())),
		)
	})
}
