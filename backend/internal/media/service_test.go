package media

import (
	"os"
	"testing"
)

func TestWithHashSuffix(t *testing.T) {
	got := withHashSuffix("IMG_0001.jpg", "abc12345")
	if got != "IMG_0001__abc12345.jpg" {
		t.Fatalf("unexpected suffix name: %s", got)
	}
}

func TestParseEXIFNoExif(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString("plain text"); err != nil {
		t.Fatal(err)
	}
	if _, ok := parseEXIFTakenAt(f.Name()); ok {
		t.Fatalf("expected no exif datetime")
	}
}
