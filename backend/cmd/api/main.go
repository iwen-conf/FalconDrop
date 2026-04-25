package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"falcondrop/backend/internal/api"
	"falcondrop/backend/internal/auth"
	"falcondrop/backend/internal/config"
	"falcondrop/backend/internal/db"
	"falcondrop/backend/internal/ftpserver"
	"falcondrop/backend/internal/media"
	"falcondrop/backend/internal/realtime"
	"falcondrop/backend/internal/storage"
	"falcondrop/backend/internal/system"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config failed", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	sqlDB, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		logger.Error("open db failed", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	if err := db.Ping(ctx, sqlDB); err != nil {
		logger.Error("ping db failed", "error", err)
		os.Exit(1)
	}

	if err := db.Migrate(sqlDB); err != nil {
		logger.Error("migrate db failed", "error", err)
		os.Exit(1)
	}

	store := db.NewStore(sqlDB)

	if err := store.SeedDefaults(context.Background(), cfg, auth.HashPassword); err != nil {
		logger.Error("seed defaults failed", "error", err)
		os.Exit(1)
	}

	localStorage := storage.NewLocal(cfg.StorageRoot, cfg.TmpRoot)
	if err := localStorage.EnsureWritable(context.Background()); err != nil {
		logger.Error("storage check failed", "error", err)
		os.Exit(1)
	}

	hub := realtime.NewHub()
	mediaSvc := media.NewService(store, localStorage, hub)

	ftpMgr := ftpserver.NewManager(cfg, store, localStorage, func(ctx context.Context, evt ftpserver.UploadedFileEvent) error {
		_, _, err := mediaSvc.HandleUploadedFile(ctx, media.UploadEvent{
			OriginalFilename: evt.OriginalFilename,
			RelativePath:     evt.RelativePath,
			TempPath:         evt.TempPath,
			Size:             evt.Size,
			RemoteAddr:       evt.RemoteAddr,
			CompletedAt:      evt.CompletedAt,
		})
		return err
	})

	sessionMgr := auth.NewSessionManager(cfg)
	authSvc := auth.NewService(store, sessionMgr)
	systemSvc := system.NewService(cfg, store, localStorage, ftpMgr)

	handler := api.NewHandler(store, authSvc, sessionMgr, mediaSvc, systemSvc, hub, ftpMgr, localStorage)
	router := api.NewRouter(handler, authSvc, sessionMgr)

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("http server starting", "addr", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	_, _ = ftpMgr.Stop(shutdownCtx)
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("http shutdown failed", "error", err)
	}
	logger.Info("server stopped")
}
