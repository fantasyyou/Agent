// Package controller exposes the Go application's HTTP API.
package controller

import (
	"agent-go/config"
	"agent-go/service"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type Server struct {
	auth        *service.AuthService
	chatService *service.ChatService
	authConfig  config.AuthConfig
}

func NewServer(auth *service.AuthService, chat *service.ChatService, cfg config.AuthConfig) *Server {
	return &Server{auth: auth, chatService: chat, authConfig: cfg}
}
func (s *Server) Run(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("POST /api/auth/register", s.register)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("POST /api/auth/logout", s.logout)
	mux.HandleFunc("GET /api/auth/me", s.me)
	mux.HandleFunc("POST /api/chat", s.chat)
	server := &http.Server{Addr: addr, Handler: requestLogger(mux), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 90 * time.Second, IdleTimeout: 120 * time.Second}
	slog.Info("http_server_started", "address", addr)
	return server.ListenAndServe()
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		writer := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(writer, r)
		if r.URL.Path != "/health" {
			slog.Info("http_request_completed", "method", r.Method, "path", r.URL.Path, "status", writer.status, "duration_ms", time.Since(started).Milliseconds())
		}
	})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
