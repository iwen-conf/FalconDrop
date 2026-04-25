package config

import "testing"

func TestParsePassivePorts(t *testing.T) {
	start, end, err := ParsePassivePorts("30000-30009")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if start != 30000 || end != 30009 {
		t.Fatalf("unexpected range: %d-%d", start, end)
	}
}

func TestParsePassivePortsInvalid(t *testing.T) {
	_, _, err := ParsePassivePorts("30009-30000")
	if err == nil {
		t.Fatalf("expected error")
	}
}
