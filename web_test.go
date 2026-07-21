package main

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHandleWebRPCValidatesRequestBoundary(t *testing.T) {
	app := NewApp()
	app.instanceRoot = t.TempDir()

	tests := []struct {
		name       string
		method     string
		origin     string
		body       string
		wantStatus int
		wantBody   string
	}{
		{name: "method", method: http.MethodGet, body: `{}`, wantStatus: http.StatusMethodNotAllowed, wantBody: "method not allowed"},
		{name: "origin", method: http.MethodPost, origin: "https://example.com", body: `{}`, wantStatus: http.StatusForbidden, wantBody: "origin rejected"},
		{name: "json", method: http.MethodPost, body: `{`, wantStatus: http.StatusBadRequest, wantBody: "请求格式错误"},
		{name: "rpc method", method: http.MethodPost, body: `{"method":"Unknown","params":[]}`, wantStatus: http.StatusBadRequest, wantBody: "Web 不支持该操作：Unknown"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(test.method, "http://dock.local/api/rpc", bytes.NewBufferString(test.body))
			if test.origin != "" {
				req.Header.Set("Origin", test.origin)
			}
			response := httptest.NewRecorder()

			app.handleWebRPC(response, req)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if !strings.Contains(response.Body.String(), test.wantBody) {
				t.Fatalf("body = %q, want %q", response.Body.String(), test.wantBody)
			}
		})
	}
}

func TestHandleWebRPCReturnsSuccessfulResult(t *testing.T) {
	app := NewApp()
	app.instanceRoot = t.TempDir()
	app.web = newWebRuntime()
	app.web.running = true

	req := httptest.NewRequest(http.MethodPost, "http://dock.local/api/rpc", bytes.NewBufferString(`{"method":"GetWebStatus","params":[]}`))
	response := httptest.NewRecorder()
	app.handleWebRPC(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var body struct {
		OK     bool      `json:"ok"`
		Result WebStatus `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.OK || !body.Result.Running {
		t.Fatalf("response = %+v", body)
	}
}

func TestRegisterWebRoutesRequiresSessionForRPC(t *testing.T) {
	app := NewApp()
	app.instanceRoot = t.TempDir()
	app.web = newWebRuntime()
	app.web.running = true
	mux := http.NewServeMux()
	app.registerWebRoutes(mux, http.NotFoundHandler())

	rpcBody := `{"method":"GetWebStatus","params":[]}`
	unauthenticated := httptest.NewRecorder()
	mux.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodPost, "http://dock.local/api/rpc", bytes.NewBufferString(rpcBody)))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want %d", unauthenticated.Code, http.StatusUnauthorized)
	}

	login := httptest.NewRecorder()
	mux.ServeHTTP(login, httptest.NewRequest(http.MethodPost, "http://dock.local/api/login", bytes.NewBufferString(`{"password":"123456"}`)))
	if login.Code != http.StatusOK || len(login.Result().Cookies()) != 1 {
		t.Fatalf("login status = %d, cookies = %d, body = %q", login.Code, len(login.Result().Cookies()), login.Body.String())
	}

	authenticatedRequest := httptest.NewRequest(http.MethodPost, "http://dock.local/api/rpc", bytes.NewBufferString(rpcBody))
	authenticatedRequest.AddCookie(login.Result().Cookies()[0])
	authenticated := httptest.NewRecorder()
	mux.ServeHTTP(authenticated, authenticatedRequest)
	if authenticated.Code != http.StatusOK {
		t.Fatalf("authenticated status = %d, body = %q", authenticated.Code, authenticated.Body.String())
	}
	var body struct {
		OK     bool      `json:"ok"`
		Result WebStatus `json:"result"`
	}
	if err := json.Unmarshal(authenticated.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.OK || !body.Result.Running {
		t.Fatalf("authenticated response = %+v", body)
	}
}

func TestIsVirtualNetworkInterface(t *testing.T) {
	for _, name := range []string{"utun3", "bridge0", "docker0", "vEthernet (Default Switch)", "VMware Network Adapter VMnet1", "Tailscale", "cni0", "flannel.1", "cali123", "podman0", "lxdbr0", "incusbr0", "nordlynx", "ppp0"} {
		if !isVirtualNetworkInterface(name) {
			t.Errorf("%q should be treated as virtual", name)
		}
	}
	for _, name := range []string{"en0", "eth0", "wlan0", "Ethernet"} {
		if isVirtualNetworkInterface(name) {
			t.Errorf("%q should be treated as a LAN interface", name)
		}
	}
}

func TestSaveWebSettingsRejectsOccupiedPortWithoutChangingConfig(t *testing.T) {
	app := newTestApp(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	before, err := app.readWebConfig()
	if err != nil {
		t.Fatal(err)
	}
	if err := app.SaveWebSettings(WebSettingsRequest{Enabled: true, Host: "127.0.0.1", Port: port}); err == nil {
		t.Fatal("occupied port should be rejected")
	}
	after, err := app.readWebConfig()
	if err != nil {
		t.Fatal(err)
	}
	if after.Host != before.Host || after.Port != before.Port || after.Enabled != before.Enabled {
		t.Fatalf("web config changed after failed preflight: before=%+v after=%+v", before, after)
	}
}

func TestWebTextFilePathAllowsProfileEnv(t *testing.T) {
	app := newTestApp(t)
	if err := app.CreateProfile(CreateProfileRequest{ID: "sales", CopyMode: profileCopyClean}); err != nil {
		t.Fatal(err)
	}

	got, err := app.webTextFilePath("sales", "profile_env")
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join("data", "profiles", "sales", ".env")
	if got != want {
		t.Fatalf("profile_env path = %q, want %q", got, want)
	}
}

func TestWebRPCProfileOperationsRequireExplicitProfile(t *testing.T) {
	app := NewApp()
	handlers := app.webRPCHandlers()
	for _, name := range []string{
		"GetAppStateForProfile",
		"SaveModelConfigForProfile",
		"SaveFeishuConfigForProfile",
		"SaveDingTalkConfigForProfile",
		"StartDingTalkLoginForProfile",
		"ListProfileSkillsForProfile",
		"BatchDeleteSkillsForProfile",
		"ReadWebTextFile",
		"BatchCopyProfileConfig",
		"SyncBundledContent",
	} {
		if handlers[name] == nil {
			t.Fatalf("missing profile-scoped Web RPC handler %s", name)
		}
	}
	for _, name := range []string{"GetAppState", "SaveModelConfig", "SaveFeishuConfig", "ListProfileSkills"} {
		if handlers[name] != nil {
			t.Fatalf("legacy current-profile Web RPC handler remains exposed: %s", name)
		}
	}
}

func TestAddWebSessionReturnsReadErrorWithoutOverwritingFile(t *testing.T) {
	app := NewApp()
	app.instanceRoot = t.TempDir()
	if err := ensureDir(app.hermesDockDir()); err != nil {
		t.Fatal(err)
	}
	path := app.webSessionsPath()
	if err := os.WriteFile(path, []byte("not-json"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := app.addWebSession(webSession{IDHash: "new"}); err == nil {
		t.Fatal("addWebSession should reject a corrupted session file")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "not-json" {
		t.Fatalf("corrupted session file was overwritten: %q", got)
	}
}

func TestAddWebSessionSerializesConcurrentUpdates(t *testing.T) {
	app := NewApp()
	app.instanceRoot = t.TempDir()
	if err := ensureDir(app.hermesDockDir()); err != nil {
		t.Fatal(err)
	}

	const count = 20
	expires := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	errs := make(chan error, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			errs <- app.addWebSession(webSession{IDHash: strconv.Itoa(index), ExpiresAt: expires})
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	file, err := app.readWebSessions()
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Sessions) != count {
		t.Fatalf("session count = %d, want %d", len(file.Sessions), count)
	}
}
