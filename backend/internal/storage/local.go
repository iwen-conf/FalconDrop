package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type Status struct {
	Root      string    `json:"root"`
	Writable  bool      `json:"writable"`
	FreeBytes *int64    `json:"freeBytes,omitempty"`
	LastError string    `json:"lastError,omitempty"`
	CheckedAt time.Time `json:"checkedAt"`
}

type Local struct {
	root string
	tmp  string
}

func NewLocal(root, tmp string) *Local {
	return &Local{
		root: filepath.Clean(root),
		tmp:  filepath.Clean(tmp),
	}
}

func (l *Local) Root() string {
	return l.root
}

func (l *Local) TmpRoot() string {
	return l.tmp
}

func (l *Local) EnsureWritable(ctx context.Context) error {
	_ = ctx
	for _, p := range []string{l.root, l.tmp} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return err
		}
		test := filepath.Join(p, ".writecheck")
		if err := os.WriteFile(test, []byte("ok"), 0o644); err != nil {
			return fmt.Errorf("path not writable %s: %w", p, err)
		}
		_ = os.Remove(test)
	}
	return nil
}

func (l *Local) SafeRelPath(rel string) (string, error) {
	rel = strings.TrimSpace(rel)
	rel = strings.ReplaceAll(rel, "\\", "/")
	rel = strings.TrimPrefix(rel, "/")
	rel = filepath.Clean(rel)
	if rel == "." {
		return "", nil
	}
	if strings.Contains(rel, "\x00") {
		return "", fmt.Errorf("invalid path")
	}
	if strings.HasPrefix(rel, "../") || rel == ".." || filepath.IsAbs(rel) {
		return "", fmt.Errorf("path traversal blocked")
	}
	return rel, nil
}

func (l *Local) BuildStoragePath(rel string) (string, error) {
	rel, err := l.SafeRelPath(rel)
	if err != nil {
		return "", err
	}
	full := filepath.Join(l.root, rel)
	cleanRoot := filepath.Clean(l.root)
	cleanFull := filepath.Clean(full)
	relToRoot, err := filepath.Rel(cleanRoot, cleanFull)
	if err != nil {
		return "", err
	}
	relToRoot = filepath.ToSlash(relToRoot)
	if relToRoot == ".." || strings.HasPrefix(relToRoot, "../") {
		return "", fmt.Errorf("path traversal blocked")
	}
	return cleanFull, nil
}

func (l *Local) BuildTmpPath(name string) (string, error) {
	safeName := strings.TrimSpace(filepath.Base(name))
	if safeName == "" || safeName == "." {
		safeName = "upload"
	}
	if strings.Contains(safeName, "\x00") {
		return "", fmt.Errorf("invalid filename")
	}
	return filepath.Join(l.tmp, fmt.Sprintf("%d_%s", time.Now().UnixNano(), safeName)), nil
}

func (l *Local) MoveFile(tempPath, finalFullPath string) error {
	if err := os.MkdirAll(filepath.Dir(finalFullPath), 0o755); err != nil {
		return err
	}
	return os.Rename(tempPath, finalFullPath)
}

func (l *Local) Remove(fullPath string) error {
	return os.Remove(fullPath)
}

func (l *Local) Open(fullPath string) (*os.File, error) {
	return os.Open(fullPath)
}

func (l *Local) WriteTemp(name string, src io.Reader) (string, int64, error) {
	tmpPath, err := l.BuildTmpPath(name)
	if err != nil {
		return "", 0, err
	}
	f, err := os.Create(tmpPath)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	n, err := io.Copy(f, src)
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", 0, err
	}
	return tmpPath, n, nil
}

func (l *Local) Stat(root string) (Status, error) {
	checked := time.Now().UTC()
	var fs syscall.Statfs_t
	if err := syscall.Statfs(root, &fs); err != nil {
		return Status{
			Root:      root,
			Writable:  false,
			LastError: err.Error(),
			CheckedAt: checked,
		}, err
	}
	free := int64(fs.Bavail * uint64(fs.Bsize))
	return Status{
		Root:      root,
		Writable:  true,
		FreeBytes: &free,
		CheckedAt: checked,
	}, nil
}
