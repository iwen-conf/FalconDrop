package api

import (
	"net/http"

	"falcondrop/backend/internal/auth"
	"github.com/gin-gonic/gin"
)

func NewRouter(h *Handler, authSvc *auth.Service, sessions *auth.SessionManager) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestIDMiddleware())

	r.GET("/healthz", h.Healthz)
	r.GET("/readyz", h.Readyz)

	r.POST("/api/auth/login", h.Login)

	protected := r.Group("/api")
	protected.Use(authMiddleware(authSvc, sessions))
	{
		protected.POST("/auth/logout", h.Logout)
		protected.GET("/auth/me", h.Me)
		protected.PUT("/system/account", h.UpdateSystemAccount)
		protected.PUT("/ftp/account", h.UpdateFtpAccount)

		protected.GET("/ftp/status", h.FtpStatus)
		protected.POST("/ftp/start", h.FtpStart)
		protected.POST("/ftp/stop", h.FtpStop)

		protected.GET("/system/info", h.SystemInfo)
		protected.GET("/ws", h.Ws)

		protected.GET("/photos", h.ListPhotos)
		protected.GET("/photos/:id", h.GetPhoto)
		protected.GET("/photos/:id/content", h.GetPhotoContent)
		protected.DELETE("/photos/:id", h.DeletePhoto)

		protected.GET("/assets", h.ListAssets)
		protected.GET("/assets/:id", h.GetAsset)
	}

	r.NoRoute(func(c *gin.Context) {
		writeError(c, http.StatusNotFound, AppError{
			Code:    "NOT_FOUND",
			Message: "接口不存在",
		})
	})

	return r
}

func writeError(c *gin.Context, status int, err AppError) {
	c.JSON(status, gin.H{
		"code":      err.Code,
		"message":   err.Message,
		"requestId": requestIDFromContext(c),
	})
}
