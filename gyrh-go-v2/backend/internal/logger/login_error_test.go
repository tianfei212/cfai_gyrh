package logger

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoginErrorWritesDedicatedAuditFile(t *testing.T) {
	dir := t.TempDir()
	Init(Config{Level: DebugLevel, Directory: dir})

	LoginError("203.0.113.77", "admin", "127.0.0.1:12345", "login-error-test", errors.New("bad password"))

	content, err := os.ReadFile(filepath.Join(dir, "login_error.log"))
	if err != nil {
		t.Fatalf("read login_error.log: %v", err)
	}
	got := string(content)
	for _, want := range []string{
		"[LOGIN_ERROR]",
		"real_ip=203.0.113.77",
		"username=\"admin\"",
		"remote_addr=127.0.0.1:12345",
		"user_agent=\"login-error-test\"",
		"error=\"bad password\"",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("login_error.log missing %q in %q", want, got)
		}
	}
}
