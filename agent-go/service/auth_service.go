// Package service contains application use cases and business rules.
package service

import (
	"agent-go/config"
	"agent-go/model"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"time"
)

type UserRepository interface {
	Create(context.Context, model.User) error
	GetByUsername(context.Context, string) (model.User, error)
}
type AuthService struct {
	users  UserRepository
	secret []byte
	ttl    time.Duration
}

func NewAuthService(users UserRepository, cfg config.AuthConfig) *AuthService {
	return &AuthService{users: users, secret: []byte(cfg.SigningSecret), ttl: config.Duration(cfg.TTL, 24*time.Hour)}
}
func (s *AuthService) Register(ctx context.Context, username, password string) (model.User, string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, "", err
	}
	id, err := newID("usr_")
	if err != nil {
		return model.User{}, "", err
	}
	now := time.Now().UTC()
	user := model.User{ID: id, Username: username, PasswordHash: string(hash), Status: model.UserStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := s.users.Create(ctx, user); err != nil {
		return model.User{}, "", err
	}
	token, err := s.Issue(user)
	return user, token, err
}
func (s *AuthService) Login(ctx context.Context, username, password string) (model.User, string, error) {
	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return model.User{}, "", err
	}
	if user.Status != model.UserStatusActive || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return model.User{}, "", model.ErrUserNotFound
	}
	token, err := s.Issue(user)
	return user, token, err
}
func (s *AuthService) Issue(user model.User) (string, error) {
	claims := model.AuthClaims{UserID: user.ID, Username: user.Username, Expires: time.Now().Add(s.ttl).Unix()}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + s.sign(encoded), nil
}
func (s *AuthService) Parse(token string) (model.AuthClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 || !hmac.Equal([]byte(s.sign(parts[0])), []byte(parts[1])) {
		return model.AuthClaims{}, errors.New("invalid token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return model.AuthClaims{}, errors.New("invalid token")
	}
	var claims model.AuthClaims
	if json.Unmarshal(payload, &claims) != nil || claims.UserID == "" || claims.Expires <= time.Now().Unix() {
		return model.AuthClaims{}, errors.New("expired or invalid token")
	}
	return claims, nil
}
func (s *AuthService) TTL() time.Duration { return s.ttl }
func (s *AuthService) sign(payload string) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func newID(prefix string) (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(value), nil
}
