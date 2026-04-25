package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"time"

	"falcondrop/backend/internal/config"
	"falcondrop/backend/internal/db"
	"github.com/google/uuid"
)

type SessionManager struct {
	cfg config.Config
}

func NewSessionManager(cfg config.Config) *SessionManager {
	return &SessionManager{cfg: cfg}
}

func (m *SessionManager) CookieName() string {
	return m.cfg.SessionCookieName
}

func (m *SessionManager) NewToken() (token string, tokenHash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	tokenHash = m.hashToken(token)
	return token, tokenHash, nil
}

func (m *SessionManager) hashToken(token string) string {
	sum := sha256.Sum256([]byte(m.cfg.SessionSecret + ":" + token))
	return hex.EncodeToString(sum[:])
}

func (m *SessionManager) HashToken(token string) string {
	return m.hashToken(token)
}

func (m *SessionManager) NewSession(accountID uuid.UUID, tokenHash string, now time.Time) db.Session {
	return db.Session{
		ID:               uuid.New(),
		AccountID:        accountID,
		SessionTokenHash: tokenHash,
		ExpiresAt:        now.Add(m.cfg.SessionTTL),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func (m *SessionManager) SetCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     m.cfg.SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   m.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	})
}

func (m *SessionManager) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     m.cfg.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   m.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}
