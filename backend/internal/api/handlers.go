package api

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"falcondrop/backend/internal/auth"
	"falcondrop/backend/internal/db"
	"falcondrop/backend/internal/ftpserver"
	"falcondrop/backend/internal/media"
	"falcondrop/backend/internal/realtime"
	"falcondrop/backend/internal/storage"
	"falcondrop/backend/internal/system"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Handler struct {
	store    *db.Store
	authSvc  *auth.Service
	sessions *auth.SessionManager
	mediaSvc *media.Service
	system   *system.Service
	hub      *realtime.Hub
	ftp      *ftpserver.Manager
	storage  *storage.Local
}

func NewHandler(
	store *db.Store,
	authSvc *auth.Service,
	sessions *auth.SessionManager,
	mediaSvc *media.Service,
	systemSvc *system.Service,
	hub *realtime.Hub,
	ftp *ftpserver.Manager,
	st *storage.Local,
) *Handler {
	return &Handler{
		store:    store,
		authSvc:  authSvc,
		sessions: sessions,
		mediaSvc: mediaSvc,
		system:   systemSvc,
		hub:      hub,
		ftp:      ftp,
		storage:  st,
	}
}

func (h *Handler) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) Readyz(c *gin.Context) {
	if err := h.store.ReadinessCheck(c.Request.Context()); err != nil {
		writeError(c, http.StatusServiceUnavailable, AppError{
			Code:    "DB_NOT_READY",
			Message: "数据库尚未就绪",
		})
		return
	}
	if err := h.storage.EnsureWritable(c.Request.Context()); err != nil {
		writeError(c, http.StatusServiceUnavailable, AppError{
			Code:    "STORAGE_NOT_WRITABLE",
			Message: "存储目录不可写",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, AppError{
			Code:    "AUTH_INVALID_CREDENTIALS",
			Message: "用户名或密码错误",
		})
		return
	}

	res, err := h.authSvc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		status, appErr := mapError(err)
		writeError(c, status, appErr)
		return
	}
	h.sessions.SetCookie(c.Writer, res.Token, res.Expires)

	c.JSON(http.StatusOK, gin.H{
		"account": gin.H{
			"username":  res.Account.Username,
			"updatedAt": res.Account.UpdatedAt,
		},
	})
}

func (h *Handler) Logout(c *gin.Context) {
	token, _ := c.Cookie(h.sessions.CookieName())
	_ = h.authSvc.Logout(c.Request.Context(), token)
	h.sessions.ClearCookie(c.Writer)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) Me(c *gin.Context) {
	acc, ok := accountFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, AppError{
			Code:    "AUTH_REQUIRED",
			Message: "请先登录",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"account": gin.H{
			"id":        acc.ID,
			"username":  acc.Username,
			"updatedAt": acc.UpdatedAt,
		},
	})
}

type updateSystemAccountRequest struct {
	Username        string `json:"username"`
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (h *Handler) UpdateSystemAccount(c *gin.Context) {
	var req updateSystemAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, AppError{
			Code:    "SYSTEM_ACCOUNT_INVALID",
			Message: "系统账号参数不正确",
		})
		return
	}
	acc, ok := accountFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, AppError{
			Code:    "AUTH_REQUIRED",
			Message: "请先登录",
		})
		return
	}
	updated, err := h.authSvc.UpdateSystemAccount(c.Request.Context(), acc.ID, auth.UpdateSystemAccountInput{
		Username:        req.Username,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	})
	if err != nil {
		status, appErr := mapError(err)
		writeError(c, status, appErr)
		return
	}

	token, _ := c.Cookie(h.sessions.CookieName())
	_ = h.authSvc.Logout(c.Request.Context(), token)
	h.sessions.ClearCookie(c.Writer)

	c.JSON(http.StatusOK, gin.H{
		"account": gin.H{
			"id":        updated.ID,
			"username":  updated.Username,
			"updatedAt": updated.UpdatedAt,
		},
		"requiresRelogin": true,
	})
}

type updateFtpAccountRequest struct {
	Username         string `json:"username"`
	Password         string `json:"password"`
	AnonymousEnabled bool   `json:"anonymousEnabled"`
}

func (h *Handler) UpdateFtpAccount(c *gin.Context) {
	var req updateFtpAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, AppError{
			Code:    "FTP_ACCOUNT_INVALID",
			Message: "FTP 账号参数不正确",
		})
		return
	}
	ftpAcc, err := h.authSvc.UpdateFtpAccount(c.Request.Context(), auth.UpdateFtpAccountInput{
		Username:         req.Username,
		Password:         req.Password,
		AnonymousEnabled: req.AnonymousEnabled,
	})
	if err != nil {
		status, appErr := mapError(err)
		writeError(c, status, appErr)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ftpAccount": gin.H{
			"username":         ftpAcc.Username,
			"anonymousEnabled": ftpAcc.AnonymousEnabled,
			"updatedAt":        ftpAcc.UpdatedAt,
		},
	})
	h.hub.Broadcast("system-status", gin.H{
		"ftpAccount": gin.H{
			"username":         ftpAcc.Username,
			"anonymousEnabled": ftpAcc.AnonymousEnabled,
			"updatedAt":        ftpAcc.UpdatedAt,
		},
	})
}

func (h *Handler) FtpStatus(c *gin.Context) {
	c.JSON(http.StatusOK, h.ftp.Snapshot())
}

func (h *Handler) FtpStart(c *gin.Context) {
	status, err := h.ftp.Start(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusBadRequest, AppError{
			Code:    "FTP_START_FAILED",
			Message: "FTP 启动失败",
		})
		return
	}
	h.hub.Broadcast("ftp-started", status)
	h.hub.Broadcast("system-status", gin.H{"ftpStatus": status})
	c.JSON(http.StatusOK, status)
}

func (h *Handler) FtpStop(c *gin.Context) {
	status, err := h.ftp.Stop(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusBadRequest, AppError{
			Code:    "FTP_STOP_FAILED",
			Message: "FTP 停止失败",
		})
		return
	}
	h.hub.Broadcast("ftp-stopped", status)
	h.hub.Broadcast("system-status", gin.H{"ftpStatus": status})
	c.JSON(http.StatusOK, status)
}

func (h *Handler) SystemInfo(c *gin.Context) {
	info, err := h.system.Info(c.Request.Context())
	if err != nil {
		status, appErr := mapError(err)
		writeError(c, status, appErr)
		return
	}
	c.JSON(http.StatusOK, info)
}

func parseCursor(raw string) (*time.Time, *uuid.UUID, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, nil, err
	}
	var payload struct {
		T string `json:"t"`
		I string `json:"i"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		return nil, nil, err
	}
	t, err := time.Parse(time.RFC3339Nano, payload.T)
	if err != nil {
		return nil, nil, err
	}
	id, err := uuid.Parse(payload.I)
	if err != nil {
		return nil, nil, err
	}
	return &t, &id, nil
}

func buildCursor(t time.Time, id uuid.UUID) string {
	b, _ := json.Marshal(map[string]string{
		"t": t.UTC().Format(time.RFC3339Nano),
		"i": id.String(),
	})
	return base64.RawURLEncoding.EncodeToString(b)
}

func (h *Handler) ListPhotos(c *gin.Context) {
	limit := 50
	if raw := c.Query("limit"); strings.TrimSpace(raw) != "" {
		n, err := strconv.Atoi(raw)
		if err == nil {
			limit = n
		}
	}
	cursorT, cursorID, err := parseCursor(c.Query("cursor"))
	if err != nil {
		writeError(c, http.StatusBadRequest, AppError{
			Code:    "INVALID_CURSOR",
			Message: "分页参数无效",
		})
		return
	}
	assets, err := h.mediaSvc.ListPhotos(c.Request.Context(), limit, cursorT, cursorID)
	if err != nil {
		status, appErr := mapError(err)
		writeError(c, status, appErr)
		return
	}
	resp := make([]gin.H, 0, len(assets))
	for _, a := range assets {
		resp = append(resp, h.photoResponse(a))
	}
	var nextCursor string
	if len(assets) == limit && len(assets) > 0 {
		last := assets[len(assets)-1]
		nextCursor = buildCursor(last.UploadedAt, last.ID)
	}
	c.JSON(http.StatusOK, gin.H{
		"items":      resp,
		"nextCursor": nextCursor,
	})
}

func (h *Handler) ListAssets(c *gin.Context) {
	limit := 50
	if raw := c.Query("limit"); strings.TrimSpace(raw) != "" {
		n, err := strconv.Atoi(raw)
		if err == nil {
			limit = n
		}
	}
	cursorT, cursorID, err := parseCursor(c.Query("cursor"))
	if err != nil {
		writeError(c, http.StatusBadRequest, AppError{
			Code:    "INVALID_CURSOR",
			Message: "分页参数无效",
		})
		return
	}

	assets, err := h.mediaSvc.ListAssets(c.Request.Context(), limit, cursorT, cursorID)
	if err != nil {
		status, appErr := mapError(err)
		writeError(c, status, appErr)
		return
	}

	resp := make([]gin.H, 0, len(assets))
	for _, a := range assets {
		resp = append(resp, h.assetResponse(a))
	}
	var nextCursor string
	if len(assets) == limit && len(assets) > 0 {
		last := assets[len(assets)-1]
		nextCursor = buildCursor(last.UploadedAt, last.ID)
	}
	c.JSON(http.StatusOK, gin.H{
		"items":      resp,
		"nextCursor": nextCursor,
	})
}

func (h *Handler) GetPhoto(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, AppError{
			Code:    "INVALID_ID",
			Message: "照片 ID 无效",
		})
		return
	}
	asset, err := h.mediaSvc.FindAsset(c.Request.Context(), id)
	if err != nil {
		status, appErr := mapError(err)
		writeError(c, status, appErr)
		return
	}
	if !asset.IsPhoto {
		writeError(c, http.StatusNotFound, AppError{
			Code:    "NOT_FOUND",
			Message: "照片不存在",
		})
		return
	}
	c.JSON(http.StatusOK, h.photoResponse(asset))
}

func (h *Handler) GetAsset(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, AppError{
			Code:    "INVALID_ID",
			Message: "资产 ID 无效",
		})
		return
	}
	asset, err := h.mediaSvc.FindAsset(c.Request.Context(), id)
	if err != nil {
		status, appErr := mapError(err)
		writeError(c, status, appErr)
		return
	}
	c.JSON(http.StatusOK, h.assetResponse(asset))
}

func (h *Handler) GetPhotoContent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, AppError{
			Code:    "INVALID_ID",
			Message: "照片 ID 无效",
		})
		return
	}
	asset, err := h.mediaSvc.FindAsset(c.Request.Context(), id)
	if err != nil {
		status, appErr := mapError(err)
		writeError(c, status, appErr)
		return
	}
	if !asset.IsPhoto {
		writeError(c, http.StatusNotFound, AppError{
			Code:    "NOT_FOUND",
			Message: "照片不存在",
		})
		return
	}
	path, err := h.mediaSvc.StorageFilePath(asset)
	if err != nil {
		writeError(c, http.StatusInternalServerError, AppError{
			Code:    "MEDIA_FILE_MISSING",
			Message: "照片文件不存在",
		})
		return
	}
	c.File(path)
}

func (h *Handler) DeletePhoto(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, AppError{
			Code:    "INVALID_ID",
			Message: "照片 ID 无效",
		})
		return
	}
	err = h.mediaSvc.DeletePhoto(c.Request.Context(), id, c.ClientIP())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, db.ErrNotFound) {
			writeError(c, http.StatusNotFound, AppError{
				Code:    "NOT_FOUND",
				Message: "照片不存在",
			})
			return
		}
		status, appErr := mapError(err)
		writeError(c, status, appErr)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *Handler) Ws(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	h.hub.Add(conn)
	defer h.hub.Remove(conn)

	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *Handler) photoResponse(a db.MediaAsset) gin.H {
	var exif any
	if a.ExifTakenAt.Valid {
		exif = a.ExifTakenAt.Time
	}
	return gin.H{
		"id":               a.ID,
		"originalFilename": a.OriginalFilename,
		"contentHash":      a.ContentHash,
		"size":             a.Size,
		"mimeType":         a.MimeType,
		"isPhoto":          a.IsPhoto,
		"uploadedAt":       a.UploadedAt,
		"fallbackTakenAt":  a.FallbackTakenAt,
		"exifTakenAt":      exif,
		"contentUrl":       "/api/photos/" + a.ID.String() + "/content",
	}
}

func (h *Handler) assetResponse(a db.MediaAsset) gin.H {
	var exif any
	if a.ExifTakenAt.Valid {
		exif = a.ExifTakenAt.Time
	}
	return gin.H{
		"id":               a.ID,
		"originalFilename": a.OriginalFilename,
		"relativeDir":      a.RelativeDir,
		"contentHash":      a.ContentHash,
		"hashAlgorithm":    a.HashAlgorithm,
		"size":             a.Size,
		"mimeType":         a.MimeType,
		"isPhoto":          a.IsPhoto,
		"uploadedAt":       a.UploadedAt,
		"lastSeenAt":       a.LastSeenAt,
		"fallbackTakenAt":  a.FallbackTakenAt,
		"exifTakenAt":      exif,
	}
}
