package system

import (
	"context"
	"time"

	"falcondrop/backend/internal/config"
	"falcondrop/backend/internal/db"
	"falcondrop/backend/internal/ftpserver"
	"falcondrop/backend/internal/storage"
)

type Service struct {
	cfg     config.Config
	store   *db.Store
	storage *storage.Local
	ftp     *ftpserver.Manager
}

func NewService(cfg config.Config, store *db.Store, st *storage.Local, ftp *ftpserver.Manager) *Service {
	return &Service{
		cfg:     cfg,
		store:   store,
		storage: st,
		ftp:     ftp,
	}
}

type AccountSummary struct {
	Username  string    `json:"username"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type FtpAccountSummary struct {
	Username         string    `json:"username"`
	AnonymousEnabled bool      `json:"anonymousEnabled"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type SystemInfo struct {
	Version       string            `json:"version"`
	BuildHash     string            `json:"buildHash"`
	SystemTime    time.Time         `json:"systemTime"`
	SystemAccount AccountSummary    `json:"systemAccount"`
	FtpAccount    FtpAccountSummary `json:"ftpAccount"`
	FtpStatus     ftpserver.Status  `json:"ftpStatus"`
	Storage       storage.Status    `json:"storage"`
}

func (s *Service) Info(ctx context.Context) (SystemInfo, error) {
	acc, err := s.store.GetSystemAccount(ctx)
	if err != nil {
		return SystemInfo{}, err
	}
	ftpAcc, err := s.store.GetFtpAccount(ctx)
	if err != nil {
		return SystemInfo{}, err
	}

	stStatus, stErr := s.storage.Stat(s.storage.Root())
	if stErr != nil {
		stStatus.Root = s.storage.Root()
		stStatus.Writable = false
	}
	ftpStatus := s.ftp.Snapshot()

	return SystemInfo{
		Version:    s.cfg.Version,
		BuildHash:  s.cfg.BuildHash,
		SystemTime: time.Now().UTC(),
		SystemAccount: AccountSummary{
			Username:  acc.Username,
			UpdatedAt: acc.UpdatedAt,
		},
		FtpAccount: FtpAccountSummary{
			Username:         ftpAcc.Username,
			AnonymousEnabled: ftpAcc.AnonymousEnabled,
			UpdatedAt:        ftpAcc.UpdatedAt,
		},
		FtpStatus: ftpStatus,
		Storage:   stStatus,
	}, nil
}
