package service

import (
	"agent-go/model"
	"testing"
	"time"
)

func TestAuthToken(t *testing.T) {
	auth := &AuthService{secret: []byte("12345678901234567890123456789012"), ttl: time.Hour}
	token, err := auth.Issue(model.User{ID: "usr_1", Username: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := auth.Parse(token)
	if err != nil || claims.UserID != "usr_1" || claims.Username != "alice" {
		t.Fatalf("unexpected claims: %#v, %v", claims, err)
	}
	if _, err := auth.Parse(token + "x"); err == nil {
		t.Fatal("tampered token was accepted")
	}
}
