package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedactProxyCredentials(t *testing.T) {
	got := redact("HTTP_PROXY=http://user:secret@host.docker.internal:7890")
	if strings.Contains(got, "secret") {
		t.Fatalf("proxy password was not redacted: %s", got)
	}
	if !strings.Contains(got, "http://user:<redacted>@host.docker.internal:7890") {
		t.Fatalf("unexpected redacted proxy URL: %s", got)
	}
}

func TestAtomicWriteFileReplacesContentAndMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteFile(path, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("content = %q, want new", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
}
