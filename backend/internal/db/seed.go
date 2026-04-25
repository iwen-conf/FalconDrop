package db

import (
	"context"
	"errors"
	"time"

	"falcondrop/backend/internal/config"
	"github.com/google/uuid"
)

type HashPasswordFn func(string) (string, error)

func (s *Store) SeedDefaults(ctx context.Context, cfg config.Config, hashPassword HashPasswordFn) error {
	now := time.Now().UTC()

	_, err := s.GetSystemAccount(ctx)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return err
		}
		hash, hErr := hashPassword(cfg.DefaultSystemPassword)
		if hErr != nil {
			return hErr
		}
		if cErr := s.CreateSystemAccount(ctx, SystemAccount{
			ID:                uuid.New(),
			Username:          cfg.DefaultSystemUsername,
			PasswordHash:      hash,
			PasswordUpdatedAt: now,
			CreatedAt:         now,
			UpdatedAt:         now,
		}); cErr != nil {
			return cErr
		}
	}

	_, err = s.GetFtpAccount(ctx)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return err
		}
		hash, hErr := hashPassword(cfg.DefaultFTPPassword)
		if hErr != nil {
			return hErr
		}
		if cErr := s.CreateFtpAccount(ctx, FtpAccount{
			ID:                1,
			Username:          cfg.DefaultFTPUsername,
			PasswordHash:      hash,
			AnonymousEnabled:  cfg.DefaultFTPAnonymous,
			PasswordUpdatedAt: now,
			CreatedAt:         now,
			UpdatedAt:         now,
		}); cErr != nil {
			return cErr
		}
	}

	if err := s.UpsertAppSetting(ctx, "ftp.port", cfg.FTPPort); err != nil {
		return err
	}
	if err := s.UpsertAppSetting(ctx, "ftp.passive_ports", cfg.FTPPassivePorts); err != nil {
		return err
	}
	if err := s.UpsertAppSetting(ctx, "ftp.public_host", cfg.FTPPublicHost); err != nil {
		return err
	}
	if err := s.UpsertAppSetting(ctx, "storage.root", cfg.StorageRoot); err != nil {
		return err
	}
	if err := s.UpsertAppSetting(ctx, "storage.tmp_root", cfg.TmpRoot); err != nil {
		return err
	}

	return nil
}
