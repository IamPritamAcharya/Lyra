package auth

import (
	"context"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"testing"
	"time"
)

type memoryStore struct {
	expires time.Time
	csrf    []byte
	exists  bool
}

func (m *memoryStore) Create(_ context.Context, _ []byte, csrf []byte, expires time.Time) error {
	m.expires = expires
	m.csrf = csrf
	m.exists = true
	return nil
}
func (m *memoryStore) Get(_ context.Context, _ []byte) (time.Time, []byte, error) {
	if !m.exists {
		return time.Time{}, nil, errors.New("missing")
	}
	return m.expires, m.csrf, nil
}
func (m *memoryStore) Delete(_ context.Context, _ []byte) error { m.exists = false; return nil }
func TestLoginSessionAndCSRF(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	store := &memoryStore{}
	service := New("admin", string(hash), store)
	token, csrf, _, err := service.Login(context.Background(), "admin", "password")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Validate(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	if err := service.ValidateCSRF(context.Background(), token, csrf); err != nil {
		t.Fatal(err)
	}
	if err := service.ValidateCSRF(context.Background(), token, "wrong"); !errors.Is(err, ErrCSRF) {
		t.Fatalf("%v", err)
	}
}
func TestInvalidCredentials(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	_, _, _, err := New("admin", string(hash), &memoryStore{}).Login(context.Background(), "admin", "wrong")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("%v", err)
	}
}
