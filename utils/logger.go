package utils

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"time"

	"log/slog"
)

type (
	// struct for holding response details
	responseData struct {
		Status      int               `json:"status"`
		Size        int               `json:"size"`
		Duration    float64           `json:"duration"`
		RespHeaders map[string]string `json:"resp_headers"`
	}

	// our http.ResponseWriter implementation
	loggingResponseWriter struct {
		http.ResponseWriter // compose original http.ResponseWriter
		responseData        *responseData
	}

	// TLS information
	tlsInfo struct {
		Resumed     bool   `json:"resumed"`
		Version     int    `json:"version"`
		CipherSuite int    `json:"cipher_suite"`
		Proto       string `json:"proto"`
		ServerName  string `json:"server_name"`
	}

	// Request information
	requestInfo struct {
		RemoteIP   string            `json:"remote_ip"`
		RemotePort string            `json:"remote_port"`
		ClientIP   string            `json:"client_ip"`
		Proto      string            `json:"proto"`
		Method     string            `json:"method"`
		Host       string            `json:"host"`
		URI        string            `json:"uri"`
		Headers    map[string]string `json:"headers"`
		TLS        *tlsInfo          `json:"tls,omitempty"`
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.Size += size
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

type Logger struct {
	*slog.Logger
}

func NewLogger(writer io.Writer) *Logger {
	return &Logger{
		Logger: slog.New(slog.NewJSONHandler(writer, nil)),
	}
}

func (me *Logger) HTTPResponseLogger(h http.Handler) http.Handler {
	loggingFn := func(rw http.ResponseWriter, req *http.Request) {
		start := time.Now()

		responseData := &responseData{
			Status:      0,
			Size:        0,
			RespHeaders: make(map[string]string),
		}
		lrw := loggingResponseWriter{
			ResponseWriter: rw,
			responseData:   responseData,
		}

		h.ServeHTTP(&lrw, req)

		duration := time.Since(start)
		responseData.Duration = duration.Seconds()

		// Capture response headers
		for k, v := range lrw.Header() {
			responseData.RespHeaders[k] = v[0]
		}

		remoteIP, remotePort := parseRemoteAddr(req.RemoteAddr)

		// Prepare request information
		reqInfo := requestInfo{
			RemoteIP:   remoteIP,
			RemotePort: remotePort,
			ClientIP:   req.Header.Get("X-Forwarded-For"),
			Proto:      req.Proto,
			Method:     req.Method,
			Host:       req.Host,
			URI:        req.RequestURI,
			Headers:    make(map[string]string),
		}

		// Capture request headers
		for k, v := range req.Header {
			reqInfo.Headers[k] = v[0]
		}

		// Capture TLS information if available
		if req.TLS != nil {
			reqInfo.TLS = &tlsInfo{
				Resumed:     req.TLS.DidResume,
				Version:     int(req.TLS.Version),
				CipherSuite: int(req.TLS.CipherSuite),
				Proto:       req.TLS.NegotiatedProtocol,
				ServerName:  req.TLS.ServerName,
			}
		}

		// Prepare log attributes
		attrs := []slog.Attr{
			slog.String("level", "info"),
			slog.Float64("ts", float64(time.Now().UnixNano())/1e9),
			slog.String("logger", "http.log.access"),
			slog.String("msg", "handled request"),
			slog.Any("request", reqInfo),
			slog.Int("bytes_read", int(req.ContentLength)),
			slog.String("user_id", ""), // You may want to implement user identification
			slog.Float64("duration", responseData.Duration),
			slog.Int("size", responseData.Size),
			slog.Int("status", responseData.Status),
			slog.Any("resp_headers", responseData.RespHeaders),
		}

		me.Logger.LogAttrs(req.Context(), slog.LevelInfo, "request", attrs...)
	}
	return http.HandlerFunc(loggingFn)
}

func parseRemoteAddr(remoteAddr string) (string, string) {
	ip, port, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// If there's an error, return the whole address as IP and empty string as port
		return remoteAddr, ""
	}
	return ip, port
}
