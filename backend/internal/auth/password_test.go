package auth

import "testing"

func TestPasswordHashAndVerify(t *testing.T) {
	hash, err := HashPassword("secret-123")
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}
	if !VerifyPassword(hash, "secret-123") {
		t.Fatalf("verify should pass")
	}
	if VerifyPassword(hash, "wrong") {
		t.Fatalf("verify should fail")
	}
}
