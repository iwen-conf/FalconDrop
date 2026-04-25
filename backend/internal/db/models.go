package db

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type SystemAccount struct {
	ID                uuid.UUID `json:"id"`
	Username          string    `json:"username"`
	PasswordHash      string    `json:"-"`
	PasswordUpdatedAt time.Time `json:"passwordUpdatedAt"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type FtpAccount struct {
	ID                int16     `json:"id"`
	Username          string    `json:"username"`
	PasswordHash      string    `json:"-"`
	AnonymousEnabled  bool      `json:"anonymousEnabled"`
	PasswordUpdatedAt time.Time `json:"passwordUpdatedAt"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type AppSetting struct {
	Key       string          `json:"key"`
	ValueJSON json.RawMessage `json:"valueJson"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type MediaAsset struct {
	ID               uuid.UUID    `json:"id"`
	OriginalFilename string       `json:"originalFilename"`
	RelativeDir      string       `json:"relativeDir"`
	StoragePath      string       `json:"-"`
	ContentHash      string       `json:"contentHash"`
	HashAlgorithm    string       `json:"hashAlgorithm"`
	Size             int64        `json:"size"`
	MimeType         string       `json:"mimeType"`
	IsPhoto          bool         `json:"isPhoto"`
	ExifTakenAt      sql.NullTime `json:"exifTakenAt"`
	FallbackTakenAt  time.Time    `json:"fallbackTakenAt"`
	UploadedAt       time.Time    `json:"uploadedAt"`
	LastSeenAt       time.Time    `json:"lastSeenAt"`
	UpdatedAt        time.Time    `json:"updatedAt"`
}

type TransferEvent struct {
	ID               uuid.UUID  `json:"id"`
	AssetID          *uuid.UUID `json:"assetId,omitempty"`
	EventType        string     `json:"eventType"`
	OriginalFilename string     `json:"originalFilename"`
	ContentHash      string     `json:"contentHash"`
	RemoteAddr       string     `json:"remoteAddr"`
	Bytes            int64      `json:"bytes"`
	Message          string     `json:"message"`
	CreatedAt        time.Time  `json:"createdAt"`
}

type Session struct {
	ID               uuid.UUID    `json:"id"`
	AccountID        uuid.UUID    `json:"accountId"`
	SessionTokenHash string       `json:"-"`
	ExpiresAt        time.Time    `json:"expiresAt"`
	RevokedAt        sql.NullTime `json:"revokedAt"`
	CreatedAt        time.Time    `json:"createdAt"`
	UpdatedAt        time.Time    `json:"updatedAt"`
}
