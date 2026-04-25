//go:build integration
// +build integration

package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"falcondrop/backend/internal/auth"
	"falcondrop/backend/internal/config"
	"falcondrop/backend/internal/db"
	"falcondrop/backend/internal/ftpserver"
	"falcondrop/backend/internal/media"
	"falcondrop/backend/internal/realtime"
	"falcondrop/backend/internal/storage"
	"falcondrop/backend/internal/system"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jlaffaye/ftp"
)

type integrationEnv struct {
	cfg    config.Config
	router *gin.Engine
	base   *httptest.Server
	client *http.Client
}

func TestIntegrationAuthRequired(t *testing.T) {
	env, err := buildIntegrationEnv(t)
	if err != nil {
		t.Skipf("integration env not ready: %v", err)
	}

	res, body := doRequest(t, env.client, http.MethodGet, env.base.URL+"/api/photos", nil)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", res.StatusCode, body)
	}
}

func TestIntegrationAuthLoginAndMe(t *testing.T) {
	env, err := buildIntegrationEnv(t)
	if err != nil {
		t.Skipf("integration env not ready: %v", err)
	}

	login(t, env)
	res, body := doRequest(t, env.client, http.MethodGet, env.base.URL+"/api/auth/me", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("auth/me expected 200, got %d body=%s", res.StatusCode, body)
	}
}

func TestIntegrationFTPUploadAssetAndDeletePhotoWithWebSocket(t *testing.T) {
	if os.Getenv("RUN_FTP_INTEGRATION") != "1" {
		t.Skip("set RUN_FTP_INTEGRATION=1 to enable ftp upload integration")
	}

	env, err := buildIntegrationEnv(t)
	if err != nil {
		t.Skipf("integration env not ready: %v", err)
	}
	login(t, env)

	// Start FTP through API.
	res, body := doRequest(t, env.client, http.MethodPost, env.base.URL+"/api/ftp/start", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("ftp/start expected 200, got %d body=%s", res.StatusCode, body)
	}
	t.Cleanup(func() {
		_, _ = doRequestNoFail(env.client, http.MethodPost, env.base.URL+"/api/ftp/stop", nil)
	})

	wsConn := dialWSWithCookie(t, env)
	defer wsConn.Close()

	fileName := "integration-upload.jpg"
	payload := bytes.Repeat([]byte{0xff, 0xd8, 0xff, 0xdb, 0x00, 0x43}, 32)

	ftpAddr := "127.0.0.1:" + strconv.Itoa(env.cfg.FTPPort)
	c, err := ftp.Dial(ftpAddr, ftp.DialWithTimeout(10*time.Second))
	if err != nil {
		t.Fatalf("ftp dial: %v", err)
	}
	defer c.Quit()

	if err := c.Login(env.cfg.DefaultFTPUsername, env.cfg.DefaultFTPPassword); err != nil {
		t.Fatalf("ftp login: %v", err)
	}
	if err := c.MakeDir("integration"); err != nil && !strings.Contains(err.Error(), "exists") {
		// ignore if already exists
	}
	if err := c.Stor("integration/"+fileName, bytes.NewReader(payload)); err != nil {
		t.Fatalf("ftp stor: %v", err)
	}

	event := waitWSEvent(t, wsConn, "asset-uploaded", 15*time.Second)
	if event.Type != "asset-uploaded" {
		t.Fatalf("expected asset-uploaded event, got %s", event.Type)
	}

	res, body = doRequest(t, env.client, http.MethodGet, env.base.URL+"/api/assets?limit=50", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("assets expected 200, got %d body=%s", res.StatusCode, body)
	}
	assetID := findAssetIDByFilename(t, body, fileName)
	if assetID == "" {
		t.Fatalf("uploaded asset not found: %s", fileName)
	}

	res, body = doRequest(t, env.client, http.MethodGet, env.base.URL+"/api/photos?limit=50", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("photos expected 200, got %d body=%s", res.StatusCode, body)
	}
	photoID := findAssetIDByFilename(t, body, fileName)
	if photoID == "" {
		t.Fatalf("uploaded photo not found in photos: %s", fileName)
	}

	res, body = doRequest(t, env.client, http.MethodDelete, env.base.URL+"/api/photos/"+photoID, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("delete photo expected 200, got %d body=%s", res.StatusCode, body)
	}

	delEvent := waitWSEvent(t, wsConn, "photo-deleted", 10*time.Second)
	if delEvent.Type != "photo-deleted" {
		t.Fatalf("expected photo-deleted event, got %s", delEvent.Type)
	}

	res, _ = doRequest(t, env.client, http.MethodGet, env.base.URL+"/api/photos/"+photoID, nil)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted photo 404, got %d", res.StatusCode)
	}
}

func buildIntegrationEnv(t *testing.T) (*integrationEnv, error) {
	gin.SetMode(gin.TestMode)

	for _, key := range []string{
		"DATABASE_URL",
		"STORAGE_ROOT",
		"TMP_ROOT",
		"SESSION_SECRET",
		"DEFAULT_SYSTEM_PASSWORD",
		"DEFAULT_FTP_PASSWORD",
	} {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			return nil, errors.New("missing required env " + key)
		}
	}

	if os.Getenv("FTP_PORT") == "" {
		p, err := getFreeTCPPort()
		if err == nil {
			t.Setenv("FTP_PORT", strconv.Itoa(p))
			if p+20 <= 65535 {
				t.Setenv("FTP_PASSIVE_PORTS", strconv.Itoa(p+1)+"-"+strconv.Itoa(p+20))
			}
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	sqlDB := mustOpenDB(t, cfg.DatabaseURL)
	t.Cleanup(func() { _ = sqlDB.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.Ping(ctx, sqlDB); err != nil {
		return nil, err
	}
	if err := db.Migrate(sqlDB); err != nil {
		return nil, err
	}

	store := db.NewStore(sqlDB)
	if err := store.SeedDefaults(ctx, cfg, auth.HashPassword); err != nil {
		return nil, err
	}

	st := storage.NewLocal(cfg.StorageRoot, cfg.TmpRoot)
	if err := st.EnsureWritable(ctx); err != nil {
		return nil, err
	}

	hub := realtime.NewHub()
	mediaSvc := media.NewService(store, st, hub)
	ftpMgr := ftpserver.NewManager(cfg, store, st, func(ctx context.Context, evt ftpserver.UploadedFileEvent) error {
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
	t.Cleanup(func() {
		_, _ = ftpMgr.Stop(context.Background())
	})

	systemSvc := system.NewService(cfg, store, st, ftpMgr)
	sessionMgr := auth.NewSessionManager(cfg)
	authSvc := auth.NewService(store, sessionMgr)
	handler := NewHandler(store, authSvc, sessionMgr, mediaSvc, systemSvc, hub, ftpMgr, st)
	router := NewRouter(handler, authSvc, sessionMgr)

	base := httptest.NewServer(router)
	t.Cleanup(base.Close)

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 20 * time.Second}

	return &integrationEnv{
		cfg:    cfg,
		router: router,
		base:   base,
		client: client,
	}, nil
}

func mustOpenDB(t *testing.T, databaseURL string) *sql.DB {
	sqlDB, err := db.Open(databaseURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return sqlDB
}

func login(t *testing.T, env *integrationEnv) {
	payload, _ := json.Marshal(map[string]string{
		"username": env.cfg.DefaultSystemUsername,
		"password": env.cfg.DefaultSystemPassword,
	})
	res, body := doRequest(t, env.client, http.MethodPost, env.base.URL+"/api/auth/login", payload)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login expected 200, got %d body=%s", res.StatusCode, body)
	}
}

type wsEvent struct {
	EventID   string          `json:"eventId"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"createdAt"`
}

func dialWSWithCookie(t *testing.T, env *integrationEnv) *websocket.Conn {
	u, _ := url.Parse(env.base.URL)
	wsURL := "ws://" + u.Host + "/api/ws"
	header := http.Header{}

	cookies := env.client.Jar.Cookies(&url.URL{Scheme: "http", Host: u.Host})
	if len(cookies) > 0 {
		parts := make([]string, 0, len(cookies))
		for _, c := range cookies {
			parts = append(parts, c.Name+"="+c.Value)
		}
		header.Set("Cookie", strings.Join(parts, "; "))
	}

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		if resp != nil {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("ws dial failed: %v status=%d body=%s", err, resp.StatusCode, string(b))
		}
		t.Fatalf("ws dial failed: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(20 * time.Second))
	return conn
}

func waitWSEvent(t *testing.T, conn *websocket.Conn, expectedType string, timeout time.Duration) wsEvent {
	deadline := time.Now().Add(timeout)
	_ = conn.SetReadDeadline(deadline)
	for time.Now().Before(deadline) {
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read ws event failed: %v", err)
		}
		var evt wsEvent
		if err := json.Unmarshal(data, &evt); err != nil {
			t.Fatalf("unmarshal ws event failed: %v raw=%s", err, string(data))
		}
		if evt.Type == expectedType {
			return evt
		}
	}
	t.Fatalf("timeout waiting ws event type=%s", expectedType)
	return wsEvent{}
}

func doRequest(t *testing.T, client *http.Client, method, target string, body []byte) (*http.Response, string) {
	req, err := http.NewRequest(method, target, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

func doRequestNoFail(client *http.Client, method, target string, body []byte) (*http.Response, string) {
	req, err := http.NewRequest(method, target, bytes.NewReader(body))
	if err != nil {
		return nil, err.Error()
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err.Error()
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

func findAssetIDByFilename(t *testing.T, body, filename string) string {
	var data struct {
		Items []struct {
			ID               string `json:"id"`
			OriginalFilename string `json:"originalFilename"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		t.Fatalf("unmarshal items failed: %v body=%s", err, body)
	}
	for _, it := range data.Items {
		if it.OriginalFilename == filename {
			if _, err := uuid.Parse(it.ID); err == nil {
				return it.ID
			}
		}
	}
	return ""
}

func getFreeTCPPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	addr := l.Addr().(*net.TCPAddr)
	return addr.Port, nil
}
