package controller

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"agent-go/model"
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{3,32}$`)

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeCredentials(w, r)
	if !ok {
		return
	}
	user, token, err := s.auth.Register(r.Context(), input.Username, input.Password)
	if err != nil {
		if errors.Is(err, model.ErrUsernameExists) {
			slog.Warn("user_register_rejected", "reason", "username_exists")
			writeJSON(w, http.StatusConflict, map[string]string{"error": "\u8d26\u53f7\u5df2\u5b58\u5728"})
			return
		}
		slog.Error("user_register_failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "\u6ce8\u518c\u5931\u8d25"})
		return
	}
	slog.Info("user_registered", "user_id", user.ID)
	s.setCookie(w, token)
	writeJSON(w, http.StatusCreated, publicUser(user))
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeCredentials(w, r)
	if !ok {
		return
	}
	user, token, err := s.auth.Login(r.Context(), input.Username, input.Password)
	if err != nil {
		slog.Warn("user_login_rejected", "reason", "invalid_credentials")
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "\u8d26\u53f7\u6216\u5bc6\u7801\u9519\u8bef"})
		return
	}
	slog.Info("user_logged_in", "user_id", user.ID)
	s.setCookie(w, token)
	writeJSON(w, http.StatusOK, publicUser(user))
}

func (s *Server) logout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: s.authConfig.CookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.authConfig.SecureCookie, SameSite: http.SameSiteLaxMode})
	w.WriteHeader(http.StatusNoContent)
	slog.Info("user_logged_out")
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	claims, ok := s.claims(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "\u672a\u767b\u5f55"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"user_id": claims.UserID, "username": claims.Username})
}

func decodeCredentials(w http.ResponseWriter, r *http.Request) (model.Credentials, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	var input model.Credentials
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "\u8bf7\u6c42\u683c\u5f0f\u4e0d\u6b63\u786e"})
		return input, false
	}
	input.Username = strings.TrimSpace(input.Username)
	if !usernamePattern.MatchString(input.Username) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "\u8d26\u53f7\u9700\u4e3a 3-32 \u4f4d\u5b57\u6bcd\u3001\u6570\u5b57\u3001\u4e0b\u5212\u7ebf\u6216\u77ed\u6a2a\u7ebf"})
		return input, false
	}
	if len(input.Password) < 8 || len(input.Password) > 72 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "\u5bc6\u7801\u957f\u5ea6\u9700\u4e3a 8-72 \u4f4d"})
		return input, false
	}
	return input, true
}
func (s *Server) claims(r *http.Request) (model.AuthClaims, bool) {
	cookie, err := r.Cookie(s.authConfig.CookieName)
	if err != nil {
		return model.AuthClaims{}, false
	}
	claims, err := s.auth.Parse(cookie.Value)
	return claims, err == nil
}
func (s *Server) setCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{Name: s.authConfig.CookieName, Value: token, Path: "/", MaxAge: int(s.auth.TTL().Seconds()), HttpOnly: true, Secure: s.authConfig.SecureCookie, SameSite: http.SameSiteLaxMode})
}
func publicUser(user model.User) map[string]string {
	return map[string]string{"user_id": user.ID, "username": user.Username}
}
