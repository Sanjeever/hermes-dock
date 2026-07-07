package main

import (
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
