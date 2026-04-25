package api

import (
	"errors"
	"net/http"

	"falcondrop/backend/internal/auth"
	"falcondrop/backend/internal/db"
)

type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func mapError(err error) (int, AppError) {
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		return http.StatusUnauthorized, AppError{Code: "AUTH_INVALID_CREDENTIALS", Message: "用户名或密码错误"}
	case errors.Is(err, auth.ErrCurrentPassword):
		return http.StatusBadRequest, AppError{Code: "CURRENT_PASSWORD_INVALID", Message: "当前密码错误"}
	case errors.Is(err, auth.ErrInvalidFTPAccount):
		return http.StatusBadRequest, AppError{Code: "FTP_ACCOUNT_INVALID", Message: "FTP 账号配置无效"}
	case errors.Is(err, db.ErrNotFound):
		return http.StatusNotFound, AppError{Code: "NOT_FOUND", Message: "资源不存在"}
	default:
		return http.StatusInternalServerError, AppError{Code: "INTERNAL_ERROR", Message: "服务器内部错误"}
	}
}
