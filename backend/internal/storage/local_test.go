package storage

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestSafeRelPathBlocksTraversal(t *testing.T) {
	s := NewLocal("/tmp/root", "/tmp/tmp")
	if _, err := s.SafeRelPath("../etc/passwd"); err == nil {
		t.Fatalf("expected traversal error")
	}
}

func TestBuildStoragePath(t *testing.T) {
	base := t.TempDir()
	s := NewLocal(base, filepath.Join(base, "tmp"))
	p, err := s.BuildStoragePath("a/b/c.jpg")
	if err != nil {
		t.Fatalf("build path error: %v", err)
	}
	if filepath.Clean(p) != filepath.Join(base, "a/b/c.jpg") {
		t.Fatalf("unexpected path %s", p)
	}
}

func TestEnsureWritable(t *testing.T) {
	base := t.TempDir()
	s := NewLocal(filepath.Join(base, "uploads"), filepath.Join(base, "tmp"))
	if err := s.EnsureWritable(nil); err != nil {
		t.Fatalf("ensure writable failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, "uploads")); err != nil {
		t.Fatalf("uploads dir missing: %v", err)
	}
}

func TestMoveFileFallbackOnCrossDeviceLink(t *testing.T) {
	base := t.TempDir()
	s := NewLocal(filepath.Join(base, "uploads"), filepath.Join(base, "tmp"))
	if err := s.EnsureWritable(nil); err != nil {
		t.Fatalf("ensure writable failed: %v", err)
	}

	src := filepath.Join(s.TmpRoot(), "from.txt")
	if err := os.WriteFile(src, []byte("payload"), 0o644); err != nil {
		t.Fatalf("write src failed: %v", err)
	}
	dst := filepath.Join(s.Root(), "nested", "to.txt")

	orig := renameFile
	t.Cleanup(func() { renameFile = orig })
	renameFile = func(oldPath, newPath string) error {
		if oldPath == src && newPath == dst {
			return &os.LinkError{
				Op:  "rename",
				Old: oldPath,
				New: newPath,
				Err: syscall.EXDEV,
			}
		}
		return os.Rename(oldPath, newPath)
	}

	if err := s.MoveFile(src, dst); err != nil {
		t.Fatalf("move with exdev fallback failed: %v", err)
	}
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst failed: %v", err)
	}
	if string(content) != "payload" {
		t.Fatalf("unexpected dst content: %q", string(content))
	}
	if _, err := os.Stat(src); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("src should be removed, err=%v", err)
	}
}
