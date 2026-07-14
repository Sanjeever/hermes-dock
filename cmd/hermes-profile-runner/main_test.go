package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadEnvFileParsesQuotedValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := "PLAIN=value\nQUOTED=\"hello world\"\nESCAPED=\"a\\\\b\\\"c\"\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := readEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["PLAIN"] != "value" || got["QUOTED"] != "hello world" || got["ESCAPED"] != `a\b"c` {
		t.Fatalf("unexpected env values: %#v", got)
	}
}

func TestReadEnvFileRejectsInvalidKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("BAD KEY=value\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readEnvFile(path); err == nil {
		t.Fatal("readEnvFile should reject an invalid key")
	}
}

func TestSetEnvReplacesExactKey(t *testing.T) {
	got := setEnv([]string{"TOKEN_OLD=keep", "TOKEN=old"}, "TOKEN", "new")
	if strings.Join(got, ",") != "TOKEN_OLD=keep,TOKEN=new" {
		t.Fatalf("unexpected environment: %#v", got)
	}
}

func TestPrefixLinesAddsProfileAndRedactsSecrets(t *testing.T) {
	var out bytes.Buffer
	prefixLinesTo(&out, "sales", strings.NewReader("ready\napi_key=secret-value\n\n"))
	got := out.String()
	if !strings.Contains(got, "[sales] ready\n") || strings.Contains(got, "secret-value") {
		t.Fatalf("unexpected prefixed log: %q", got)
	}
}

func TestTooManyRecentFailuresUsesFiveMinuteWindow(t *testing.T) {
	now := time.Now()
	failures := []time.Time{
		now.Add(-6 * time.Minute),
		now.Add(-4 * time.Minute),
		now.Add(-3 * time.Minute),
		now.Add(-2 * time.Minute),
		now.Add(-time.Minute),
		now,
	}
	if !tooManyRecentFailures(failures, now) {
		t.Fatal("five recent failures should stop restarts")
	}
	if tooManyRecentFailures(failures[:5], now) {
		t.Fatal("four recent failures should still allow restart")
	}
}

func TestInitialRuntimeStatusKeepsManifestGeneration(t *testing.T) {
	manifest := RuntimeManifest{
		Generation: "generation-1",
		Profiles: []RuntimeManifestProfile{{
			ID:       "sales",
			Enabled:  true,
			Runnable: true,
		}},
	}
	status := initialRuntimeStatus(manifest)
	if status.Generation != manifest.Generation {
		t.Fatalf("status generation = %q, want %q", status.Generation, manifest.Generation)
	}
	if got := status.Profiles["sales"].State; got != "starting" {
		t.Fatalf("initial sales state = %q, want starting", got)
	}
}
