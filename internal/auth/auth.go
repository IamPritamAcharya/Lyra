// Package auth provides a single configured admin account with server-side sessions.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid admin credentials")
var ErrUnauthenticated = errors.New("admin authentication required")
var ErrCSRF = errors.New("invalid csrf token")

type SessionStore interface {
	Create(context.Context, []byte, []byte, time.Time) error
	Get(context.Context, []byte) (expires time.Time, csrfHash []byte, err error)
	Delete(context.Context, []byte) error
}
type Service struct {
	username, passwordHash string
	store                  SessionStore
	ttl                    time.Duration
}

func New(username, passwordHash string, store SessionStore) *Service {
	return &Service{username: username, passwordHash: passwordHash, store: store, ttl: 12 * time.Hour}
}
func (s *Service) Login(ctx context.Context, username, password string) (token, csrf string, expires time.Time, err error) {
	validUser := subtle.ConstantTimeCompare([]byte(username), []byte(s.username)) == 1
	if bcrypt.CompareHashAndPassword([]byte(s.passwordHash), []byte(password)) != nil || !validUser {
		return "", "", time.Time{}, ErrInvalidCredentials
	}
	token, err = randomToken()
	if err != nil {
		return "", "", time.Time{}, err
	}
	csrf, err = randomToken()
	if err != nil {
		return "", "", time.Time{}, err
	}
	expires = time.Now().UTC().Add(s.ttl)
	if err := s.store.Create(ctx, hash(token), hash(csrf), expires); err != nil {
		return "", "", time.Time{}, fmt.Errorf("create admin session: %w", err)
	}
	return token, csrf, expires, nil
}
func (s *Service) Validate(ctx context.Context, token string) error {
	if token == "" {
		return ErrUnauthenticated
	}
	expires, _, err := s.store.Get(ctx, hash(token))
	if err != nil {
		return ErrUnauthenticated
	}
	if !expires.After(time.Now().UTC()) {
		_ = s.store.Delete(ctx, hash(token))
		return ErrUnauthenticated
	}
	return nil
}
func (s *Service) ValidateCSRF(ctx context.Context, token, csrf string) error {
	if token == "" || csrf == "" {
		return ErrUnauthenticated
	}
	expires, csrfHash, err := s.store.Get(ctx, hash(token))
	if err != nil {
		return ErrUnauthenticated
	}
	if !expires.After(time.Now().UTC()) {
		_ = s.store.Delete(ctx, hash(token))
		return ErrUnauthenticated
	}
	expected := hash(csrf)
	if subtle.ConstantTimeCompare(expected, csrfHash) != 1 {
		return ErrCSRF
	}
	return nil
}
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.store.Delete(ctx, hash(token))
}
func hash(value string) []byte { sum := sha256.Sum256([]byte(value)); return sum[:] }
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
