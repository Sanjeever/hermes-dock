package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRegisterDingTalkBotAndPersistCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected request: %s content-type=%q", r.Method, r.Header.Get("Content-Type"))
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		switch r.URL.Path {
		case dingtalkRegistrationInitPath:
			if payload["source"] != "openClaw" {
				t.Fatalf("unexpected init payload: %#v", payload)
			}
			_, _ = w.Write([]byte(`{"errcode":0,"nonce":"nonce"}`))
		case dingtalkRegistrationBeginPath:
			if payload["nonce"] != "nonce" {
				t.Fatalf("unexpected begin payload: %#v", payload)
			}
			_, _ = w.Write([]byte(`{"errcode":0,"device_code":"device-code","verification_uri_complete":"https://scan.example/path","interval":0}`))
		case dingtalkRegistrationPollPath:
			if payload["device_code"] != "device-code" {
				t.Fatalf("unexpected poll payload: %#v", payload)
			}
			_, _ = w.Write([]byte(`{"errcode":0,"status":"SUCCESS","client_id":"app-key","client_secret":"secret-test"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	originalBaseURL := dingtalkRegistrationBaseURL
	dingtalkRegistrationBaseURL = server.URL
	t.Cleanup(func() { dingtalkRegistrationBaseURL = originalBaseURL })

	credentials, err := NewApp().registerDingTalkBot(context.Background(), defaultProfileID)
	if err != nil {
		t.Fatal(err)
	}
	if credentials.ClientID != "app-key" || credentials.ClientSecret != "secret-test" {
		t.Fatalf("unexpected credentials: %#v", credentials)
	}

	app := newTestApp(t)
	profileID := "sales"
	if err := app.CreateProfile(CreateProfileRequest{ID: profileID, Name: "销售助手", Enabled: true, CopyMode: profileCopyClean}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(app.profileDataDir(profileID), ".env")
	if err := app.saveEnvironmentTo(path, []EnvVar{{Key: "DINGTALK_HOME_CHANNEL", Value: "chat-existing"}}); err != nil {
		t.Fatal(err)
	}
	if err := app.persistDingTalkCredentials(context.Background(), profileID, credentials); err != nil {
		t.Fatal(err)
	}
	env, err := readEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		"DINGTALK_CLIENT_ID":       "app-key",
		"DINGTALK_CLIENT_SECRET":   "secret-test",
		"DINGTALK_ALLOW_ALL_USERS": "true",
		"DINGTALK_REQUIRE_MENTION": "true",
		"DINGTALK_HOME_CHANNEL":    "chat-existing",
	} {
		if got := envValue(env, key); got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestPersistDingTalkCredentialsCancelledDoesNotWrite(t *testing.T) {
	app := newTestApp(t)
	path := app.profileEnvPath(defaultProfileID)
	if err := app.saveEnvironmentTo(path, []EnvVar{{Key: "CUSTOM_VALUE", Value: "keep"}}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := app.persistDingTalkCredentials(ctx, defaultProfileID, dingtalkCredentials{ClientID: "app-key", ClientSecret: "secret"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("persist error = %v, want context canceled", err)
	}
	env, err := readEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := envValue(env, "DINGTALK_CLIENT_ID"); got != "" {
		t.Fatalf("DINGTALK_CLIENT_ID = %q, want empty", got)
	}
	if got := envValue(env, "CUSTOM_VALUE"); got != "keep" {
		t.Fatalf("CUSTOM_VALUE = %q, want keep", got)
	}
}

func TestPostDingTalkRegistrationRejectsUnsafeResponses(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		transient bool
	}{
		{name: "server error", status: http.StatusInternalServerError, body: `{"errcode":0}`, transient: true},
		{name: "oversized", status: http.StatusOK, body: strings.Repeat("x", dingtalkResponseLimit+1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			originalBaseURL := dingtalkRegistrationBaseURL
			dingtalkRegistrationBaseURL = server.URL
			t.Cleanup(func() { dingtalkRegistrationBaseURL = originalBaseURL })

			_, err := postDingTalkRegistration(context.Background(), "/test", map[string]string{"test": "value"})
			if err == nil {
				t.Fatal("expected error")
			}
			var transient *dingtalkTransientError
			if got := errors.As(err, &transient); got != tt.transient {
				t.Fatalf("transient = %t, want %t; error = %v", got, tt.transient, err)
			}
		})
	}
}

func TestDingTalkExpiryUsesServerValueWithinLimit(t *testing.T) {
	if got := dingtalkExpiry(float64(30)); got != 30*time.Second {
		t.Fatalf("expiry = %s, want 30s", got)
	}
	if got := dingtalkExpiry(float64(24 * 60 * 60)); got != dingtalkLoginTimeout {
		t.Fatalf("expiry = %s, want max %s", got, dingtalkLoginTimeout)
	}
}
