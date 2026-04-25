package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHTTPAddr         = ":8080"
	defaultFTPHost          = "0.0.0.0"
	defaultFTPPort          = 2121
	defaultFTPPassivePorts  = "30000-30009"
	defaultSessionTTL       = 24 * time.Hour
	defaultCookieSecure     = false
	defaultSystemUsername   = "admin"
	defaultFTPUsername      = "camera"
	defaultAnonymousEnabled = true
)

type Config struct {
	HTTPAddr    string
	DatabaseURL string

	StorageRoot string
	TmpRoot     string

	SessionSecret     string
	SessionTTL        time.Duration
	SessionCookieName string
	CookieSecure      bool

	DefaultSystemUsername string
	DefaultSystemPassword string
	DefaultFTPUsername    string
	DefaultFTPPassword    string
	DefaultFTPAnonymous   bool

	FTPHost         string
	FTPPublicHost   string
	FTPPort         int
	FTPPassivePorts string

	Version   string
	BuildHash string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:    get("HTTP_ADDR", defaultHTTPAddr),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		StorageRoot: os.Getenv("STORAGE_ROOT"),
		TmpRoot:     os.Getenv("TMP_ROOT"),

		SessionSecret:     os.Getenv("SESSION_SECRET"),
		SessionCookieName: get("SESSION_COOKIE_NAME", "falcondrop_session"),
		CookieSecure:      getBool("COOKIE_SECURE", defaultCookieSecure),
		SessionTTL:        getDuration("SESSION_TTL", defaultSessionTTL),

		DefaultSystemUsername: get("DEFAULT_SYSTEM_USERNAME", defaultSystemUsername),
		DefaultSystemPassword: os.Getenv("DEFAULT_SYSTEM_PASSWORD"),
		DefaultFTPUsername:    get("DEFAULT_FTP_USERNAME", defaultFTPUsername),
		DefaultFTPPassword:    os.Getenv("DEFAULT_FTP_PASSWORD"),
		DefaultFTPAnonymous:   getBool("DEFAULT_FTP_ANONYMOUS_ENABLED", defaultAnonymousEnabled),

		FTPHost:         get("FTP_HOST", defaultFTPHost),
		FTPPublicHost:   os.Getenv("FTP_PUBLIC_HOST"),
		FTPPassivePorts: get("FTP_PASSIVE_PORTS", defaultFTPPassivePorts),
		FTPPort:         getInt("FTP_PORT", defaultFTPPort),

		Version:   get("APP_VERSION", "dev"),
		BuildHash: get("APP_BUILD_HASH", "unknown"),
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if strings.TrimSpace(c.StorageRoot) == "" {
		return fmt.Errorf("STORAGE_ROOT is required")
	}
	if strings.TrimSpace(c.TmpRoot) == "" {
		return fmt.Errorf("TMP_ROOT is required")
	}
	if strings.TrimSpace(c.SessionSecret) == "" {
		return fmt.Errorf("SESSION_SECRET is required")
	}
	if strings.TrimSpace(c.DefaultSystemPassword) == "" {
		return fmt.Errorf("DEFAULT_SYSTEM_PASSWORD is required")
	}
	if !c.DefaultFTPAnonymous && strings.TrimSpace(c.DefaultFTPPassword) == "" {
		return fmt.Errorf("DEFAULT_FTP_PASSWORD is required when anonymous ftp is disabled")
	}
	if c.FTPPort < 1 || c.FTPPort > 65535 {
		return fmt.Errorf("FTP_PORT must be between 1 and 65535")
	}
	if _, _, err := ParsePassivePorts(c.FTPPassivePorts); err != nil {
		return err
	}
	if c.SessionTTL <= 0 {
		return fmt.Errorf("SESSION_TTL must be positive")
	}

	return nil
}

func ParsePassivePorts(raw string) (int, int, error) {
	parts := strings.Split(raw, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid FTP_PASSIVE_PORTS: %q", raw)
	}

	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid FTP_PASSIVE_PORTS start: %w", err)
	}
	end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid FTP_PASSIVE_PORTS end: %w", err)
	}
	if start < 1 || end < 1 || start > 65535 || end > 65535 || start > end {
		return 0, 0, fmt.Errorf("invalid FTP_PASSIVE_PORTS range: %q", raw)
	}

	return start, end, nil
}

func get(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func getInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
