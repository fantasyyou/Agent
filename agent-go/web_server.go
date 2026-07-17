package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

type chatRequest struct {
	SessionID string `json:"session_id"`
	Question  string `json:"question"`
}

type chatResponse struct {
	Answer string `json:"answer,omitempty"`
	Error  string `json:"error,omitempty"`
}

func RunHTTPServer(addr string, service *CustomerService) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/chat", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
		var request chatRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeJSON(w, http.StatusBadRequest, chatResponse{Error: "请求格式不正确"})
			return
		}
		request.SessionID = strings.TrimSpace(request.SessionID)
		request.Question = strings.TrimSpace(request.Question)
		if request.SessionID == "" || request.Question == "" {
			writeJSON(w, http.StatusBadRequest, chatResponse{Error: "session_id 和 question 不能为空"})
			return
		}

		answer, err := service.Ask(r.Context(), request.SessionID, request.Question)
		if err != nil {
			log.Printf("chat failed session=%s: %v", request.SessionID, err)
			writeJSON(w, http.StatusBadGateway, chatResponse{Error: "智能客服暂时不可用，请稍后重试"})
			return
		}
		writeJSON(w, http.StatusOK, chatResponse{Answer: answer})
	})

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	log.Printf("customer-service web UI listening on %s", addr)
	return server.ListenAndServe()
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
