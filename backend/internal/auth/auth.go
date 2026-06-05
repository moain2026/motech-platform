// Package auth provides JWT issuing/verification, bcrypt helpers, and the
// HTTP middleware that protects /api routes. Auth is fully self-contained
// (no external identity provider) for portability.
package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type ctxKey string

const (
	// CtxSubject is the context key holding the authenticated subject id.
	CtxSubject ctxKey = "subject"
	// CtxKind is the context key holding the token kind ("admin" or "agent").
	CtxKind ctxKey = "kind"
)

// Manager signs and verifies JWTs with a shared secret.
type Manager struct{ secret []byte }

// NewManager builds an auth Manager from the JWT secret.
func NewManager(secret string) *Manager { return &Manager{secret: []byte(secret)} }

// HashPassword returns a bcrypt hash of a plaintext password.
func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword verifies a plaintext password against a bcrypt hash.
func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// Issue creates a signed JWT for a subject id and kind, valid for ttl.
func (m *Manager) Issue(subject, kind string, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"sub":  subject,
		"kind": kind,
		"exp":  time.Now().Add(ttl).Unix(),
		"iat":  time.Now().Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(m.secret)
}

// Parse validates a token string and returns (subject, kind).
func (m *Manager) Parse(tokenStr string) (subject, kind string, err error) {
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil || !tok.Valid {
		return "", "", errors.New("invalid token")
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", errors.New("invalid claims")
	}
	sub, _ := claims["sub"].(string)
	k, _ := claims["kind"].(string)
	return sub, k, nil
}

// Middleware enforces a valid Bearer token of one of the allowed kinds.
func (m *Manager) Middleware(allowedKinds ...string) func(http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, k := range allowedKinds {
		allowed[k] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				http.Error(w, `{"error":"missing bearer token"}`, http.StatusUnauthorized)
				return
			}
			sub, kind, err := m.Parse(strings.TrimPrefix(h, "Bearer "))
			if err != nil || (len(allowed) > 0 && !allowed[kind]) {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), CtxSubject, sub)
			ctx = context.WithValue(ctx, CtxKind, kind)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
