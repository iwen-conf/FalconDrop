package media

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"falcondrop/backend/internal/db"
	"falcondrop/backend/internal/realtime"
	"falcondrop/backend/internal/storage"
	"github.com/google/uuid"
	"github.com/rwcarlsen/goexif/exif"
)

type UploadEvent struct {
	OriginalFilename string
	RelativePath     string
	TempPath         string
	Size             int64
	RemoteAddr       string
	CompletedAt      time.Time
}

type Service struct {
	store   *db.Store
	storage *storage.Local
	hub     *realtime.Hub
}

func NewService(store *db.Store, st *storage.Local, hub *realtime.Hub) *Service {
	return &Service{
		store:   store,
		storage: st,
		hub:     hub,
	}
}

func (s *Service) HandleUploadedFile(ctx context.Context, evt UploadEvent) (db.MediaAsset, string, error) {
	tempFile, err := s.storage.Open(evt.TempPath)
	if err != nil {
		_ = s.writeFailedEvent(ctx, evt, "", "临时文件不可读: "+err.Error())
		return db.MediaAsset{}, "", err
	}
	defer tempFile.Close()

	h := sha256.New()
	head := make([]byte, 512)
	n, _ := io.ReadFull(tempFile, head)
	_, _ = h.Write(head[:n])
	if _, err := io.Copy(h, tempFile); err != nil {
		_ = s.writeFailedEvent(ctx, evt, "", "hash 计算失败: "+err.Error())
		return db.MediaAsset{}, "", err
	}
	hash := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	if _, err := tempFile.Seek(0, io.SeekStart); err != nil {
		_ = s.writeFailedEvent(ctx, evt, hash, "文件游标重置失败: "+err.Error())
		return db.MediaAsset{}, "", err
	}

	mimeType := http.DetectContentType(head[:n])
	if mimeType == "application/octet-stream" {
		ext := strings.ToLower(filepath.Ext(evt.OriginalFilename))
		if ext != "" {
			if guessed := mime.TypeByExtension(ext); guessed != "" {
				mimeType = guessed
			}
		}
	}
	isPhoto := strings.HasPrefix(strings.ToLower(mimeType), "image/")
	exifTakenAt := sql.NullTime{}
	if isPhoto {
		if t, ok := parseEXIFTakenAt(evt.TempPath); ok {
			exifTakenAt = sql.NullTime{Time: t, Valid: true}
		}
	}

	existing, findErr := s.store.FindMediaByOriginalNameAndHash(ctx, evt.OriginalFilename, hash)
	now := evt.CompletedAt.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if findErr == nil {
		updated, updErr := s.store.UpdateMediaAssetOverwrite(ctx, existing.ID, evt.Size, mimeType, isPhoto, exifTakenAt, now)
		if updErr != nil {
			_ = s.writeFailedEvent(ctx, evt, hash, "覆盖更新失败: "+updErr.Error())
			return db.MediaAsset{}, "", updErr
		}
		_ = s.store.InsertTransferEvent(ctx, db.TransferEvent{
			ID:               uuid.New(),
			AssetID:          &updated.ID,
			EventType:        "overwritten",
			OriginalFilename: evt.OriginalFilename,
			ContentHash:      hash,
			RemoteAddr:       evt.RemoteAddr,
			Bytes:            evt.Size,
			Message:          "同名同 hash 覆盖",
			CreatedAt:        now,
		})
		_ = s.cleanupTemp(evt.TempPath)
		s.hub.Broadcast("asset-overwritten", toAssetEvent(updated))
		if updated.IsPhoto {
			s.hub.Broadcast("photo-added", toPhotoEvent(updated))
		}
		return updated, existing.StoragePath, nil
	}
	if !errors.Is(findErr, db.ErrNotFound) {
		_ = s.writeFailedEvent(ctx, evt, hash, "查询重复文件失败: "+findErr.Error())
		return db.MediaAsset{}, "", findErr
	}

	storageRel := s.generateStorageRelativePath(evt.RelativePath, evt.OriginalFilename, hash)
	fullPath, err := s.storage.BuildStoragePath(storageRel)
	if err != nil {
		_ = s.writeFailedEvent(ctx, evt, hash, "生成存储路径失败: "+err.Error())
		return db.MediaAsset{}, "", err
	}

	if _, existsErr := s.store.FindMediaByStoragePath(ctx, storageRel); existsErr == nil {
		storageRel = s.generateStorageRelativePath(evt.RelativePath, withHashSuffix(evt.OriginalFilename, hash[:8]), hash)
		fullPath, err = s.storage.BuildStoragePath(storageRel)
		if err != nil {
			_ = s.writeFailedEvent(ctx, evt, hash, "冲突路径生成失败: "+err.Error())
			return db.MediaAsset{}, "", err
		}
	} else if !errors.Is(existsErr, db.ErrNotFound) {
		_ = s.writeFailedEvent(ctx, evt, hash, "查询存储路径冲突失败: "+existsErr.Error())
		return db.MediaAsset{}, "", existsErr
	}

	if err := s.storage.MoveFile(evt.TempPath, fullPath); err != nil {
		_ = s.writeFailedEvent(ctx, evt, hash, "移动文件失败: "+err.Error())
		return db.MediaAsset{}, "", err
	}

	asset := db.MediaAsset{
		ID:               uuid.New(),
		OriginalFilename: evt.OriginalFilename,
		RelativeDir:      filepath.Dir(strings.TrimSpace(evt.RelativePath)),
		StoragePath:      storageRel,
		ContentHash:      hash,
		HashAlgorithm:    "sha256",
		Size:             evt.Size,
		MimeType:         mimeType,
		IsPhoto:          isPhoto,
		ExifTakenAt:      exifTakenAt,
		FallbackTakenAt:  now,
		UploadedAt:       now,
		LastSeenAt:       now,
		UpdatedAt:        now,
	}
	if asset.RelativeDir == "." {
		asset.RelativeDir = ""
	}

	if err := s.store.InsertMediaAsset(ctx, asset); err != nil {
		_ = s.writeFailedEvent(ctx, evt, hash, "资产入库失败: "+err.Error())
		return db.MediaAsset{}, "", err
	}

	_ = s.store.InsertTransferEvent(ctx, db.TransferEvent{
		ID:               uuid.New(),
		AssetID:          &asset.ID,
		EventType:        "uploaded",
		OriginalFilename: evt.OriginalFilename,
		ContentHash:      hash,
		RemoteAddr:       evt.RemoteAddr,
		Bytes:            evt.Size,
		Message:          "上传成功",
		CreatedAt:        now,
	})

	s.hub.Broadcast("asset-uploaded", toAssetEvent(asset))
	if asset.IsPhoto {
		s.hub.Broadcast("photo-added", toPhotoEvent(asset))
	}

	return asset, asset.StoragePath, nil
}

func (s *Service) DeletePhoto(ctx context.Context, id uuid.UUID, remoteAddr string) error {
	asset, err := s.store.FindMediaByID(ctx, id)
	if err != nil {
		return err
	}
	if !asset.IsPhoto {
		return db.ErrNotFound
	}

	fullPath, err := s.storage.BuildStoragePath(asset.StoragePath)
	if err != nil {
		return err
	}
	if err := s.storage.Remove(fullPath); err != nil {
		return err
	}
	if err := s.store.DeleteMediaByID(ctx, id); err != nil {
		return err
	}

	_ = s.store.InsertTransferEvent(ctx, db.TransferEvent{
		ID:               uuid.New(),
		AssetID:          nil,
		EventType:        "deleted",
		OriginalFilename: asset.OriginalFilename,
		ContentHash:      asset.ContentHash,
		RemoteAddr:       remoteAddr,
		Bytes:            asset.Size,
		Message:          "删除成功",
		CreatedAt:        time.Now().UTC(),
	})
	s.hub.Broadcast("photo-deleted", map[string]any{
		"id":         asset.ID,
		"uploadedAt": asset.UploadedAt,
	})
	return nil
}

func (s *Service) ListPhotos(ctx context.Context, limit int, cursorTime *time.Time, cursorID *uuid.UUID) ([]db.MediaAsset, error) {
	flag := true
	return s.store.ListAssets(ctx, &flag, limit, cursorTime, cursorID)
}

func (s *Service) ListAssets(ctx context.Context, limit int, cursorTime *time.Time, cursorID *uuid.UUID) ([]db.MediaAsset, error) {
	return s.store.ListAssets(ctx, nil, limit, cursorTime, cursorID)
}

func (s *Service) FindAsset(ctx context.Context, id uuid.UUID) (db.MediaAsset, error) {
	return s.store.FindMediaByID(ctx, id)
}

func (s *Service) StorageFilePath(asset db.MediaAsset) (string, error) {
	return s.storage.BuildStoragePath(asset.StoragePath)
}

func (s *Service) cleanupTemp(path string) error {
	return s.storage.Remove(path)
}

func (s *Service) writeFailedEvent(ctx context.Context, evt UploadEvent, hash, msg string) error {
	return s.store.InsertTransferEvent(ctx, db.TransferEvent{
		ID:               uuid.New(),
		AssetID:          nil,
		EventType:        "failed",
		OriginalFilename: evt.OriginalFilename,
		ContentHash:      hash,
		RemoteAddr:       evt.RemoteAddr,
		Bytes:            evt.Size,
		Message:          msg,
		CreatedAt:        time.Now().UTC(),
	})
}

func (s *Service) generateStorageRelativePath(relativePath, originalFilename, hash string) string {
	dir := strings.TrimSpace(filepath.Dir(relativePath))
	if dir == "." {
		dir = ""
	}
	filename := strings.TrimSpace(originalFilename)
	if filename == "" {
		filename = "upload"
	}
	if strings.TrimSpace(dir) != "" {
		return filepath.Join(dir, filename)
	}
	return filepath.Join(time.Now().UTC().Format("2006/01/02"), withHashSuffix(filename, hash[:8]))
}

func withHashSuffix(name, suffix string) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	if base == "" {
		base = "file"
	}
	return base + "__" + suffix + ext
}

func toAssetEvent(a db.MediaAsset) map[string]any {
	return map[string]any{
		"id":               a.ID,
		"originalFilename": a.OriginalFilename,
		"contentHash":      a.ContentHash,
		"size":             a.Size,
		"mimeType":         a.MimeType,
		"isPhoto":          a.IsPhoto,
		"uploadedAt":       a.UploadedAt,
	}
}

func toPhotoEvent(a db.MediaAsset) map[string]any {
	v := toAssetEvent(a)
	v["contentUrl"] = "/api/photos/" + a.ID.String() + "/content"
	v["fallbackTakenAt"] = a.FallbackTakenAt
	if a.ExifTakenAt.Valid {
		v["exifTakenAt"] = a.ExifTakenAt.Time
	} else {
		v["exifTakenAt"] = nil
	}
	return v
}

func parseEXIFTakenAt(path string) (time.Time, bool) {
	f, err := os.Open(path)
	if err != nil {
		return time.Time{}, false
	}
	defer f.Close()

	meta, err := exif.Decode(f)
	if err != nil {
		return time.Time{}, false
	}
	tm, err := meta.DateTime()
	if err != nil {
		return time.Time{}, false
	}
	return tm.UTC(), true
}
