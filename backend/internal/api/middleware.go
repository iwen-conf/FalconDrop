package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"falcondrop/backend/internal/auth"
	"falcondrop/backend/internal/db"
	"github.com/gin-gonic/gin"
)

const (
	requestIDKey    = "requestID"
	accountIDKey    = "accountID"
	accountModelKey = "accountModel"
)

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := strings.TrimSpace(c.GetHeader("X-Request-Id"))
		if reqID == "" {
			b := make([]byte, 16)
			if _, err := rand.Read(b); err != nil {
				reqID = "req_fallback"
			} else {
				reqID = "req_" + hex.EncodeToString(b)
			}
		}
		c.Set(requestIDKey, reqID)
		c.Writer.Header().Set("X-Request-Id", reqID)
		c.Next()
	}
}

func authMiddleware(svc *auth.Service, sessions *auth.SessionManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(sessions.CookieName())
		if err != nil || strings.TrimSpace(cookie) == "" {
			writeError(c, http.StatusUnauthorized, AppError{
				Code:    "AUTH_REQUIRED",
				Message: "请先登录",
			})
			c.Abort()
			return
		}

		sess, acc, err := svc.SessionFromToken(c.Request.Context(), cookie)
		if err != nil {
			writeError(c, http.StatusUnauthorized, AppError{
				Code:    "SESSION_INVALID",
				Message: "会话已失效，请重新登录",
			})
			c.Abort()
			return
		}

		c.Set(accountIDKey, sess.AccountID)
		c.Set(accountModelKey, acc)
		c.Next()
	}
}

func accountFromContext(c *gin.Context) (db.SystemAccount, bool) {
	v, ok := c.Get(accountModelKey)
	if !ok {
		return db.SystemAccount{}, false
	}
	acc, ok := v.(db.SystemAccount)
	return acc, ok
}

func requestIDFromContext(c *gin.Context) string {
	v, ok := c.Get(requestIDKey)
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func ctxWithRequestID(ctx context.Context, reqID string) context.Context {
	type key struct{}
	return context.WithValue(ctx, key{}, reqID)
}
