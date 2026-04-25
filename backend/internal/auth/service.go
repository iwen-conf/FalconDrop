package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"falcondrop/backend/internal/db"
	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrCurrentPassword     = errors.New("current password invalid")
	ErrInvalidFTPAccount   = errors.New("ftp account invalid")
	ErrAccountNeedsRelogin = errors.New("account update requires relogin")
)

type Service struct {
	store    *db.Store
	sessions *SessionManager
}

func NewService(store *db.Store, sessions *SessionManager) *Service {
	return &Service{store: store, sessions: sessions}
}

type LoginResult struct {
	Account db.SystemAccount
	Token   string
	Expires time.Time
}

func (s *Service) Login(ctx context.Context, username, password string) (LoginResult, error) {
	acc, err := s.store.GetSystemAccount(ctx)
	if err != nil {
		return LoginResult{}, err
	}
	if !strings.EqualFold(acc.Username, strings.TrimSpace(username)) {
		return LoginResult{}, ErrInvalidCredentials
	}
	if !VerifyPassword(acc.PasswordHash, password) {
		return LoginResult{}, ErrInvalidCredentials
	}

	token, tokenHash, err := s.sessions.NewToken()
	if err != nil {
		return LoginResult{}, err
	}
	now := time.Now().UTC()
	sess := s.sessions.NewSession(acc.ID, tokenHash, now)
	if err := s.store.CreateSession(ctx, sess); err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		Account: acc,
		Token:   token,
		Expires: sess.ExpiresAt,
	}, nil
}

func (s *Service) SessionFromToken(ctx context.Context, token string) (db.Session, db.SystemAccount, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return db.Session{}, db.SystemAccount{}, db.ErrNotFound
	}
	return s.store.GetSessionAccountByHash(ctx, s.sessions.HashToken(token))
}

func (s *Service) Logout(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	return s.store.RevokeSessionByHash(ctx, s.sessions.HashToken(token))
}

type UpdateSystemAccountInput struct {
	Username        string
	CurrentPassword string
	NewPassword     string
}

func (s *Service) UpdateSystemAccount(ctx context.Context, accountID uuid.UUID, in UpdateSystemAccountInput) (db.SystemAccount, error) {
	acc, err := s.store.GetSystemAccount(ctx)
	if err != nil {
		return db.SystemAccount{}, err
	}
	if acc.ID != accountID {
		return db.SystemAccount{}, db.ErrNotFound
	}
	if !VerifyPassword(acc.PasswordHash, in.CurrentPassword) {
		return db.SystemAccount{}, ErrCurrentPassword
	}

	username := strings.TrimSpace(in.Username)
	if username == "" {
		username = acc.Username
	}
	passwordHash := acc.PasswordHash
	pwdUpdatedAt := acc.PasswordUpdatedAt
	if strings.TrimSpace(in.NewPassword) != "" {
		hash, hErr := HashPassword(in.NewPassword)
		if hErr != nil {
			return db.SystemAccount{}, hErr
		}
		passwordHash = hash
		pwdUpdatedAt = time.Now().UTC()
	}

	updated, err := s.store.UpdateSystemAccount(ctx, acc.ID, username, passwordHash, pwdUpdatedAt)
	if err != nil {
		return db.SystemAccount{}, err
	}
	if err := s.store.RevokeSessionsByAccount(ctx, acc.ID); err != nil {
		return db.SystemAccount{}, err
	}
	return updated, nil
}

type UpdateFtpAccountInput struct {
	Username         string
	Password         string
	AnonymousEnabled bool
}

func (s *Service) UpdateFtpAccount(ctx context.Context, in UpdateFtpAccountInput) (db.FtpAccount, error) {
	curr, err := s.store.GetFtpAccount(ctx)
	if err != nil {
		return db.FtpAccount{}, err
	}

	username := strings.TrimSpace(in.Username)
	if username == "" {
		username = curr.Username
	}

	passwordHash := curr.PasswordHash
	pwdUpdatedAt := curr.PasswordUpdatedAt
	if strings.TrimSpace(in.Password) != "" {
		hash, hErr := HashPassword(in.Password)
		if hErr != nil {
			return db.FtpAccount{}, hErr
		}
		passwordHash = hash
		pwdUpdatedAt = time.Now().UTC()
	}

	if !in.AnonymousEnabled && (strings.TrimSpace(username) == "" || strings.TrimSpace(passwordHash) == "") {
		return db.FtpAccount{}, ErrInvalidFTPAccount
	}

	return s.store.UpdateFtpAccount(ctx, username, passwordHash, in.AnonymousEnabled, pwdUpdatedAt)
}
