package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Raw() *sql.DB {
	return s.db
}

func (s *Store) CreateSystemAccount(ctx context.Context, account SystemAccount) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO system_accounts (id, username, password_hash, password_updated_at, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6)`,
		account.ID,
		account.Username,
		account.PasswordHash,
		account.PasswordUpdatedAt,
		account.CreatedAt,
		account.UpdatedAt,
	)
	return err
}

func (s *Store) GetSystemAccount(ctx context.Context) (SystemAccount, error) {
	const q = `SELECT id, username, password_hash, password_updated_at, created_at, updated_at
               FROM system_accounts
               ORDER BY created_at ASC
               LIMIT 1`

	var v SystemAccount
	err := s.db.QueryRowContext(ctx, q).Scan(
		&v.ID,
		&v.Username,
		&v.PasswordHash,
		&v.PasswordUpdatedAt,
		&v.CreatedAt,
		&v.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return SystemAccount{}, ErrNotFound
	}
	return v, err
}

func (s *Store) UpdateSystemAccount(ctx context.Context, id uuid.UUID, username, passwordHash string, passwordUpdatedAt time.Time) (SystemAccount, error) {
	const q = `
UPDATE system_accounts
SET username=$2, password_hash=$3, password_updated_at=$4, updated_at=$5
WHERE id=$1
RETURNING id, username, password_hash, password_updated_at, created_at, updated_at
`
	now := time.Now().UTC()
	var v SystemAccount
	err := s.db.QueryRowContext(ctx, q, id, username, passwordHash, passwordUpdatedAt, now).Scan(
		&v.ID,
		&v.Username,
		&v.PasswordHash,
		&v.PasswordUpdatedAt,
		&v.CreatedAt,
		&v.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return SystemAccount{}, ErrNotFound
	}
	return v, err
}

func (s *Store) CreateFtpAccount(ctx context.Context, account FtpAccount) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO ftp_account (id, username, password_hash, anonymous_enabled, password_updated_at, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		account.ID,
		account.Username,
		account.PasswordHash,
		account.AnonymousEnabled,
		account.PasswordUpdatedAt,
		account.CreatedAt,
		account.UpdatedAt,
	)
	return err
}

func (s *Store) GetFtpAccount(ctx context.Context) (FtpAccount, error) {
	const q = `SELECT id, username, password_hash, anonymous_enabled, password_updated_at, created_at, updated_at
               FROM ftp_account
               WHERE id = 1`

	var v FtpAccount
	err := s.db.QueryRowContext(ctx, q).Scan(
		&v.ID,
		&v.Username,
		&v.PasswordHash,
		&v.AnonymousEnabled,
		&v.PasswordUpdatedAt,
		&v.CreatedAt,
		&v.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return FtpAccount{}, ErrNotFound
	}
	return v, err
}

func (s *Store) UpdateFtpAccount(ctx context.Context, username, passwordHash string, anonymous bool, passwordUpdatedAt time.Time) (FtpAccount, error) {
	const q = `
UPDATE ftp_account
SET username=$1, password_hash=$2, anonymous_enabled=$3, password_updated_at=$4, updated_at=$5
WHERE id=1
RETURNING id, username, password_hash, anonymous_enabled, password_updated_at, created_at, updated_at
`
	now := time.Now().UTC()
	var v FtpAccount
	err := s.db.QueryRowContext(ctx, q, username, passwordHash, anonymous, passwordUpdatedAt, now).Scan(
		&v.ID,
		&v.Username,
		&v.PasswordHash,
		&v.AnonymousEnabled,
		&v.PasswordUpdatedAt,
		&v.CreatedAt,
		&v.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return FtpAccount{}, ErrNotFound
	}
	return v, err
}

func (s *Store) UpsertAppSetting(ctx context.Context, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO app_settings (key, value_json, updated_at)
         VALUES ($1, $2, $3)
         ON CONFLICT (key) DO UPDATE SET value_json = EXCLUDED.value_json, updated_at = EXCLUDED.updated_at`,
		key, raw, time.Now().UTC(),
	)
	return err
}

func (s *Store) CreateSession(ctx context.Context, session Session) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO sessions (id, account_id, session_token_hash, expires_at, revoked_at, created_at, updated_at)
         VALUES ($1, $2, $3, $4, NULL, $5, $6)`,
		session.ID,
		session.AccountID,
		session.SessionTokenHash,
		session.ExpiresAt,
		session.CreatedAt,
		session.UpdatedAt,
	)
	return err
}

func (s *Store) GetSessionAccountByHash(ctx context.Context, tokenHash string) (Session, SystemAccount, error) {
	const q = `
SELECT
  s.id, s.account_id, s.session_token_hash, s.expires_at, s.revoked_at, s.created_at, s.updated_at,
  a.id, a.username, a.password_hash, a.password_updated_at, a.created_at, a.updated_at
FROM sessions s
JOIN system_accounts a ON a.id = s.account_id
WHERE s.session_token_hash = $1
  AND s.revoked_at IS NULL
  AND s.expires_at > NOW()
LIMIT 1
`
	var sess Session
	var acc SystemAccount
	err := s.db.QueryRowContext(ctx, q, tokenHash).Scan(
		&sess.ID,
		&sess.AccountID,
		&sess.SessionTokenHash,
		&sess.ExpiresAt,
		&sess.RevokedAt,
		&sess.CreatedAt,
		&sess.UpdatedAt,
		&acc.ID,
		&acc.Username,
		&acc.PasswordHash,
		&acc.PasswordUpdatedAt,
		&acc.CreatedAt,
		&acc.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, SystemAccount{}, ErrNotFound
	}
	return sess, acc, err
}

func (s *Store) RevokeSessionByHash(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE sessions
         SET revoked_at = NOW(), updated_at = NOW()
         WHERE session_token_hash = $1 AND revoked_at IS NULL`,
		tokenHash,
	)
	return err
}

func (s *Store) RevokeSessionsByAccount(ctx context.Context, accountID uuid.UUID) error {
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE sessions
         SET revoked_at = NOW(), updated_at = NOW()
         WHERE account_id = $1 AND revoked_at IS NULL`,
		accountID,
	)
	return err
}

func (s *Store) InsertMediaAsset(ctx context.Context, asset MediaAsset) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO media_assets (
            id, original_filename, relative_dir, storage_path, content_hash, hash_algorithm,
            size, mime_type, is_photo, exif_taken_at, fallback_taken_at, uploaded_at, last_seen_at, updated_at
         ) VALUES (
            $1, $2, $3, $4, $5, $6,
            $7, $8, $9, $10, $11, $12, $13, $14
         )`,
		asset.ID,
		asset.OriginalFilename,
		asset.RelativeDir,
		asset.StoragePath,
		asset.ContentHash,
		asset.HashAlgorithm,
		asset.Size,
		asset.MimeType,
		asset.IsPhoto,
		asset.ExifTakenAt,
		asset.FallbackTakenAt,
		asset.UploadedAt,
		asset.LastSeenAt,
		asset.UpdatedAt,
	)
	return err
}

func (s *Store) FindMediaByOriginalNameAndHash(ctx context.Context, filename, hash string) (MediaAsset, error) {
	const q = `
SELECT id, original_filename, relative_dir, storage_path, content_hash, hash_algorithm, size, mime_type, is_photo,
       exif_taken_at, fallback_taken_at, uploaded_at, last_seen_at, updated_at
FROM media_assets
WHERE original_filename = $1 AND content_hash = $2
ORDER BY uploaded_at DESC
LIMIT 1
`
	return scanMediaAsset(s.db.QueryRowContext(ctx, q, filename, hash))
}

func (s *Store) FindMediaByStoragePath(ctx context.Context, storagePath string) (MediaAsset, error) {
	const q = `
SELECT id, original_filename, relative_dir, storage_path, content_hash, hash_algorithm, size, mime_type, is_photo,
       exif_taken_at, fallback_taken_at, uploaded_at, last_seen_at, updated_at
FROM media_assets
WHERE storage_path = $1
LIMIT 1
`
	return scanMediaAsset(s.db.QueryRowContext(ctx, q, storagePath))
}

func (s *Store) UpdateMediaAssetOverwrite(ctx context.Context, id uuid.UUID, size int64, mimeType string, isPhoto bool, exif sql.NullTime, uploadedAt time.Time) (MediaAsset, error) {
	const q = `
UPDATE media_assets
SET size = $2,
    mime_type = $3,
    is_photo = $4,
    exif_taken_at = $5,
    uploaded_at = $6,
    last_seen_at = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING id, original_filename, relative_dir, storage_path, content_hash, hash_algorithm, size, mime_type, is_photo,
          exif_taken_at, fallback_taken_at, uploaded_at, last_seen_at, updated_at
`
	return scanMediaAsset(s.db.QueryRowContext(ctx, q, id, size, mimeType, isPhoto, exif, uploadedAt))
}

func (s *Store) FindMediaByID(ctx context.Context, id uuid.UUID) (MediaAsset, error) {
	const q = `
SELECT id, original_filename, relative_dir, storage_path, content_hash, hash_algorithm, size, mime_type, is_photo,
       exif_taken_at, fallback_taken_at, uploaded_at, last_seen_at, updated_at
FROM media_assets
WHERE id = $1
LIMIT 1
`
	return scanMediaAsset(s.db.QueryRowContext(ctx, q, id))
}

func (s *Store) DeleteMediaByID(ctx context.Context, id uuid.UUID) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM media_assets WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) InsertTransferEvent(ctx context.Context, evt TransferEvent) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO transfer_events (id, asset_id, event_type, original_filename, content_hash, remote_addr, bytes, message, created_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		evt.ID,
		evt.AssetID,
		evt.EventType,
		evt.OriginalFilename,
		evt.ContentHash,
		evt.RemoteAddr,
		evt.Bytes,
		evt.Message,
		evt.CreatedAt,
	)
	return err
}

func (s *Store) ListAssets(ctx context.Context, onlyPhoto *bool, limit int, cursorTime *time.Time, cursorID *uuid.UUID) ([]MediaAsset, error) {
	args := []any{}
	conds := []string{}
	if onlyPhoto != nil {
		args = append(args, *onlyPhoto)
		conds = append(conds, fmt.Sprintf("is_photo = $%d", len(args)))
	}
	if cursorTime != nil && cursorID != nil {
		args = append(args, *cursorTime, *cursorID)
		conds = append(conds, fmt.Sprintf("(uploaded_at, id) < ($%d, $%d)", len(args)-1, len(args)))
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + joinAnd(conds)
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	args = append(args, limit)

	q := fmt.Sprintf(`
SELECT id, original_filename, relative_dir, storage_path, content_hash, hash_algorithm, size, mime_type, is_photo,
       exif_taken_at, fallback_taken_at, uploaded_at, last_seen_at, updated_at
FROM media_assets
%s
ORDER BY uploaded_at DESC, id DESC
LIMIT $%d`, where, len(args))

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]MediaAsset, 0, limit)
	for rows.Next() {
		var v MediaAsset
		if err := rows.Scan(
			&v.ID,
			&v.OriginalFilename,
			&v.RelativeDir,
			&v.StoragePath,
			&v.ContentHash,
			&v.HashAlgorithm,
			&v.Size,
			&v.MimeType,
			&v.IsPhoto,
			&v.ExifTakenAt,
			&v.FallbackTakenAt,
			&v.UploadedAt,
			&v.LastSeenAt,
			&v.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) ReadinessCheck(ctx context.Context) error {
	var one int
	if err := s.db.QueryRowContext(ctx, "SELECT 1").Scan(&one); err != nil {
		return err
	}
	return nil
}

func scanMediaAsset(row *sql.Row) (MediaAsset, error) {
	var v MediaAsset
	err := row.Scan(
		&v.ID,
		&v.OriginalFilename,
		&v.RelativeDir,
		&v.StoragePath,
		&v.ContentHash,
		&v.HashAlgorithm,
		&v.Size,
		&v.MimeType,
		&v.IsPhoto,
		&v.ExifTakenAt,
		&v.FallbackTakenAt,
		&v.UploadedAt,
		&v.LastSeenAt,
		&v.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return MediaAsset{}, ErrNotFound
	}
	return v, err
}

func joinAnd(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += " AND " + parts[i]
	}
	return out
}
