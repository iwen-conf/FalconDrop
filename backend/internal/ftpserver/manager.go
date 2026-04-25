package ftpserver

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"falcondrop/backend/internal/config"
	"falcondrop/backend/internal/db"
	"falcondrop/backend/internal/storage"
	"github.com/goftp/server"
)

type UploadCallback func(ctx context.Context, evt UploadedFileEvent) error

type UploadedFileEvent struct {
	OriginalFilename string
	RelativePath     string
	TempPath         string
	Size             int64
	RemoteAddr       string
	CompletedAt      time.Time
}

type Status struct {
	IsRunning        bool      `json:"isRunning"`
	Host             string    `json:"host"`
	PublicHost       string    `json:"publicHost"`
	Port             int       `json:"port"`
	PassivePorts     string    `json:"passivePorts"`
	URL              string    `json:"url"`
	AnonymousEnabled bool      `json:"anonymousEnabled"`
	Username         string    `json:"username"`
	ConnectedClients int64     `json:"connectedClients"`
	FilesReceived    int64     `json:"filesReceived"`
	BytesReceived    int64     `json:"bytesReceived"`
	LastFile         string    `json:"lastFile"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type Manager struct {
	cfg      config.Config
	store    *db.Store
	storage  *storage.Local
	onUpload UploadCallback

	mu      sync.RWMutex
	server  *server.Server
	running bool
	status  Status
}

func NewManager(cfg config.Config, store *db.Store, st *storage.Local, onUpload UploadCallback) *Manager {
	m := &Manager{
		cfg:      cfg,
		store:    store,
		storage:  st,
		onUpload: onUpload,
	}
	m.status = Status{
		IsRunning:    false,
		Host:         cfg.FTPHost,
		PublicHost:   cfg.FTPPublicHost,
		Port:         cfg.FTPPort,
		PassivePorts: cfg.FTPPassivePorts,
		URL:          fmt.Sprintf("ftp://%s:%d", displayFTPHost(cfg.FTPPublicHost, cfg.FTPHost), cfg.FTPPort),
		UpdatedAt:    time.Now().UTC(),
	}
	return m
}

func (m *Manager) Start(ctx context.Context) (Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running {
		return m.status, nil
	}
	if err := m.storage.EnsureWritable(ctx); err != nil {
		return Status{}, err
	}

	ftpAcc, err := m.store.GetFtpAccount(ctx)
	if err != nil {
		return Status{}, err
	}

	driver := &ftpDriver{
		storage:  m.storage,
		onUpload: m.onUpload,
		stats:    &m.status,
	}
	opts := &server.ServerOpts{
		Name:         "FalconDrop FTP",
		Hostname:     m.cfg.FTPHost,
		Port:         m.cfg.FTPPort,
		Factory:      driver,
		Auth:         &ftpAuth{store: m.store},
		PassivePorts: m.cfg.FTPPassivePorts,
		PublicIp:     m.cfg.FTPPublicHost,
		Logger:       &ftpLogger{},
	}
	srv := server.NewServer(opts)
	m.server = srv
	m.running = true
	m.status.IsRunning = true
	m.status.AnonymousEnabled = ftpAcc.AnonymousEnabled
	m.status.Username = ftpAcc.Username
	m.status.UpdatedAt = time.Now().UTC()

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			m.mu.Lock()
			defer m.mu.Unlock()
			m.running = false
			m.status.IsRunning = false
			m.status.UpdatedAt = time.Now().UTC()
		}
	}()

	return m.status, nil
}

func (m *Manager) Stop(ctx context.Context) (Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.running || m.server == nil {
		m.status.IsRunning = false
		m.status.UpdatedAt = time.Now().UTC()
		return m.status, nil
	}
	err := m.server.Shutdown()
	if err != nil {
		return Status{}, err
	}
	_ = ctx
	m.running = false
	m.server = nil
	m.status.IsRunning = false
	m.status.UpdatedAt = time.Now().UTC()
	return m.status, nil
}

func (m *Manager) Snapshot() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cp := m.status
	cp.ConnectedClients = atomic.LoadInt64(&m.status.ConnectedClients)
	cp.FilesReceived = atomic.LoadInt64(&m.status.FilesReceived)
	cp.BytesReceived = atomic.LoadInt64(&m.status.BytesReceived)
	return cp
}

type ftpAuth struct {
	store *db.Store
}

func (a *ftpAuth) CheckPasswd(username, password string) (bool, error) {
	ctx := context.Background()
	acc, err := a.store.GetFtpAccount(ctx)
	if err != nil {
		return false, err
	}
	if acc.AnonymousEnabled && strings.EqualFold(strings.TrimSpace(username), "anonymous") {
		return true, nil
	}
	if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		return false, nil
	}
	if !strings.EqualFold(strings.TrimSpace(username), strings.TrimSpace(acc.Username)) {
		return false, nil
	}
	return checkPassword(acc.PasswordHash, password), nil
}

type ftpDriver struct {
	storage  *storage.Local
	onUpload UploadCallback
	stats    *Status
}

func (d *ftpDriver) NewDriver() (server.Driver, error) {
	return &driverInstance{
		storage:  d.storage,
		onUpload: d.onUpload,
		stats:    d.stats,
		perm:     server.NewSimplePerm("ftp", "ftp"),
	}, nil
}

type driverInstance struct {
	storage  *storage.Local
	onUpload UploadCallback
	stats    *Status
	perm     server.Perm
}

func (d *driverInstance) Init(conn *server.Conn) {
	_ = conn
	d.stats.UpdatedAt = time.Now().UTC()
}

func (d *driverInstance) ChangeDir(path string) error {
	full, err := d.storage.BuildStoragePath(path)
	if err != nil {
		return err
	}
	fi, err := os.Stat(full)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("not a directory")
	}
	return nil
}

func (d *driverInstance) Stat(path string) (server.FileInfo, error) {
	full, err := d.storage.BuildStoragePath(path)
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(full)
	if err != nil {
		return nil, err
	}
	mode, _ := d.perm.GetMode(path)
	if fi.IsDir() {
		mode |= os.ModeDir
	}
	owner, _ := d.perm.GetOwner(path)
	group, _ := d.perm.GetGroup(path)
	return &fileInfo{
		FileInfo: fi,
		mode:     mode,
		owner:    owner,
		group:    group,
	}, nil
}

func (d *driverInstance) ListDir(path string, cb func(server.FileInfo) error) error {
	full, err := d.storage.BuildStoragePath(path)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(full)
	if err != nil {
		return err
	}
	for _, e := range entries {
		info, iErr := e.Info()
		if iErr != nil {
			return iErr
		}
		mode, _ := d.perm.GetMode(e.Name())
		if e.IsDir() {
			mode |= os.ModeDir
		}
		owner, _ := d.perm.GetOwner(e.Name())
		group, _ := d.perm.GetGroup(e.Name())
		if err := cb(&fileInfo{
			FileInfo: info,
			mode:     mode,
			owner:    owner,
			group:    group,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (d *driverInstance) DeleteDir(path string) error {
	full, err := d.storage.BuildStoragePath(path)
	if err != nil {
		return err
	}
	return os.Remove(full)
}

func (d *driverInstance) DeleteFile(path string) error {
	full, err := d.storage.BuildStoragePath(path)
	if err != nil {
		return err
	}
	return os.Remove(full)
}

func (d *driverInstance) Rename(fromPath, toPath string) error {
	from, err := d.storage.BuildStoragePath(fromPath)
	if err != nil {
		return err
	}
	to, err := d.storage.BuildStoragePath(toPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	return os.Rename(from, to)
}

func (d *driverInstance) MakeDir(path string) error {
	full, err := d.storage.BuildStoragePath(path)
	if err != nil {
		return err
	}
	return os.MkdirAll(full, 0o755)
}

func (d *driverInstance) GetFile(path string, offset int64) (int64, io.ReadCloser, error) {
	full, err := d.storage.BuildStoragePath(path)
	if err != nil {
		return 0, nil, err
	}
	f, err := os.Open(full)
	if err != nil {
		return 0, nil, err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return 0, nil, err
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		_ = f.Close()
		return 0, nil, err
	}
	return info.Size(), f, nil
}

func (d *driverInstance) PutFile(destPath string, data io.Reader, appendData bool) (int64, error) {
	if appendData {
		return 0, fmt.Errorf("append not supported")
	}
	base := filepath.Base(destPath)
	tmpPath, size, err := d.storage.WriteTemp(base, data)
	if err != nil {
		return 0, err
	}

	evt := UploadedFileEvent{
		OriginalFilename: base,
		RelativePath:     strings.TrimPrefix(destPath, "/"),
		TempPath:         tmpPath,
		Size:             size,
		RemoteAddr:       "",
		CompletedAt:      time.Now().UTC(),
	}
	if d.onUpload != nil {
		if err := d.onUpload(context.Background(), evt); err != nil {
			_ = os.Remove(tmpPath)
			return 0, err
		}
	}
	atomic.AddInt64(&d.stats.FilesReceived, 1)
	atomic.AddInt64(&d.stats.BytesReceived, size)
	d.stats.LastFile = base
	d.stats.UpdatedAt = time.Now().UTC()
	return size, nil
}

type fileInfo struct {
	os.FileInfo
	mode  os.FileMode
	owner string
	group string
}

func (f *fileInfo) Mode() os.FileMode {
	return f.mode
}

func (f *fileInfo) Owner() string {
	return f.owner
}

func (f *fileInfo) Group() string {
	return f.group
}

func displayFTPHost(publicHost, host string) string {
	if strings.TrimSpace(publicHost) != "" {
		return publicHost
	}
	if host == "0.0.0.0" {
		return "127.0.0.1"
	}
	return host
}
