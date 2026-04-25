package storage

import (
	"os"
	"path/filepath"
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
