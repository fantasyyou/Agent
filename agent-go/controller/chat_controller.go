package controller

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"agent-go/model"
)

func (s *Server) chat(w http.ResponseWriter, r *http.Request) {
	claims, ok := s.claims(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, model.ChatResponse{Error: "\u8bf7\u5148\u767b\u5f55"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	var input model.ChatRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil {
		writeJSON(w, http.StatusBadRequest, model.ChatResponse{Error: "\u8bf7\u6c42\u683c\u5f0f\u4e0d\u6b63\u786e"})
		return
	}
	input.SessionID = strings.TrimSpace(input.SessionID)
	input.Question = strings.TrimSpace(input.Question)
	if input.SessionID == "" || input.Question == "" {
		writeJSON(w, http.StatusBadRequest, model.ChatResponse{Error: "session_id \u548c question \u4e0d\u80fd\u4e3a\u7a7a"})
		return
	}
	answer, err := s.chatService.Ask(r.Context(), claims.UserID, input.SessionID, input.Question)
	if err != nil {
		slog.Error("chat_request_failed", "user_id", claims.UserID, "session_id", input.SessionID, "error", err)
		writeJSON(w, http.StatusBadGateway, model.ChatResponse{Error: "\u667a\u80fd\u5ba2\u670d\u6682\u65f6\u4e0d\u53ef\u7528\uff0c\u8bf7\u7a0d\u540e\u91cd\u8bd5"})
		return
	}
	writeJSON(w, http.StatusOK, model.ChatResponse{Answer: answer})
}
