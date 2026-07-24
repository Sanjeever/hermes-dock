package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/crypto/bcrypt"
)

const (
	defaultWebHost     = "0.0.0.0"
	defaultWebPort     = "9876"
	defaultWebPassword = "123456"
	sessionCookieName  = "hermes_dock_session"
	webMaxRequestBody  = 2 << 20
)

type webRuntime struct {
	mu            sync.Mutex
	server        *http.Server
	serverCancel  context.CancelFunc
	serverDone    chan struct{}
	running       bool
	err           string
	clients       map[*webClient]bool
	loginFailures map[string]loginFailure
	logTailRefs   map[string]bool
	operationBusy bool
}

type loginFailure struct {
	Count       int
	DelayedTill time.Time
}

type webClient struct {
	id          string
	conn        *websocket.Conn
	send        chan webEvent
	closed      bool
	stopContext func() bool
}

type webClientCleanup struct {
	conn        *websocket.Conn
	stopContext func() bool
	stopLogs    bool
}

type webConfig struct {
	SchemaVersion        int    `json:"schema_version"`
	Enabled              bool   `json:"enabled"`
	Host                 string `json:"host"`
	Port                 string `json:"port"`
	PasswordHash         string `json:"password_hash"`
	SessionSecret        string `json:"session_secret"`
	UsingDefaultPassword bool   `json:"using_default_password"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
}

type webSessionFile struct {
	Sessions []webSession `json:"sessions"`
}

type webSession struct {
	IDHash     string `json:"id_hash"`
	CreatedAt  string `json:"created_at"`
	LastSeenAt string `json:"last_seen_at"`
	ExpiresAt  string `json:"expires_at"`
}

type webEvent struct {
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}

type rpcRequest struct {
	Method string            `json:"method"`
	Params []json.RawMessage `json:"params"`
}

type rpcResponse struct {
	OK     bool        `json:"ok"`
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

type webRPCHandler func([]json.RawMessage) (interface{}, error)

func (a *App) webServerPath() string {
	return filepath.Join(a.hermesDockDir(), "web-server.json")
}

func (a *App) webSessionsPath() string {
	return filepath.Join(a.hermesDockDir(), "web-sessions.json")
}

func (a *App) webLogPath() string {
	return filepath.Join(a.hermesDockDir(), "logs", "web-server.log")
}

func (a *App) webLogf(format string, args ...interface{}) {
	path := a.webLogPath()
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return
	}
	a.rotateWebLogIfNeeded(path)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return
	}
	defer file.Close()
	line := time.Now().Format(time.RFC3339) + " " + redact(fmt.Sprintf(format, args...)) + "\n"
	_, _ = file.WriteString(line)
}

func (a *App) rotateWebLogIfNeeded(path string) {
	const maxSize = 5 * 1024 * 1024
	info, err := os.Stat(path)
	if err != nil || info.Size() < maxSize {
		return
	}
	for i := 3; i >= 1; i-- {
		oldPath := fmt.Sprintf("%s.%d", path, i)
		if i == 3 {
			_ = os.Remove(oldPath)
			continue
		}
		newPath := fmt.Sprintf("%s.%d", path, i+1)
		_ = os.Rename(oldPath, newPath)
	}
	_ = os.Rename(path, path+".1")
}

func (a *App) ensureWebConfig() error {
	if fileExists(a.webServerPath()) {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	hash, err := bcrypt.GenerateFromPassword([]byte(defaultWebPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	cfg := webConfig{
		SchemaVersion:        1,
		Enabled:              true,
		Host:                 defaultWebHost,
		Port:                 defaultWebPort,
		PasswordHash:         string(hash),
		SessionSecret:        uuid.NewString(),
		UsingDefaultPassword: true,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	return a.writeWebConfig(cfg)
}

func (a *App) readWebConfig() (webConfig, error) {
	if err := a.ensureWebConfig(); err != nil {
		return webConfig{}, err
	}
	var cfg webConfig
	data, err := readJSONFile(a.webServerPath(), &cfg)
	if err != nil {
		return webConfig{}, err
	}
	_ = data
	if cfg.Host == "" {
		cfg.Host = defaultWebHost
	}
	if cfg.Port == "" {
		cfg.Port = defaultWebPort
	}
	if cfg.SessionSecret == "" {
		cfg.SessionSecret = uuid.NewString()
	}
	if cfg.SchemaVersion == 0 {
		cfg.SchemaVersion = 1
	}
	return cfg, nil
}

func (a *App) writeWebConfig(cfg webConfig) error {
	if err := ensureDir(filepath.Dir(a.webServerPath())); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return writeFilePrivate(a.webServerPath(), data)
}

func readJSONFile(path string, out interface{}) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, out); err != nil {
		return nil, err
	}
	return data, nil
}

func writeFilePrivate(path string, data []byte) error {
	return atomicWriteFile(path, data, 0600)
}

func (a *App) startWebServer() {
	_ = a.startWebServerChecked()
}

func (a *App) startWebServerChecked() error {
	a.webLifecycleMu.Lock()
	defer a.webLifecycleMu.Unlock()
	return a.startWebServerCheckedLocked()
}

func (a *App) startWebServerCheckedLocked() error {
	if a.web == nil {
		a.web = newWebRuntime()
	}
	cfg, err := a.readWebConfig()
	if err != nil {
		a.webLogf("server start failed error=%s", err.Error())
		a.setWebError(err.Error())
		return err
	}
	if !cfg.Enabled {
		a.webLogf("server disabled")
		a.stopWebServerLocked(context.Background())
		return nil
	}
	web := a.web
	web.mu.Lock()
	if web.running {
		web.mu.Unlock()
		return nil
	}
	web.mu.Unlock()

	dist, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		a.webLogf("server start failed error=%s", err.Error())
		a.setWebError(err.Error())
		return err
	}
	mux := http.NewServeMux()
	serverCtx, serverCancel := context.WithCancel(context.Background())
	a.registerWebRoutesWithContext(mux, http.FileServer(http.FS(dist)), serverCtx)
	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		serverCancel()
		a.webLogf("server start failed addr=%s error=%s", addr, err.Error())
		a.setWebError(err.Error())
		return err
	}
	server := newWebHTTPServer(addr, mux, serverCtx)
	done := make(chan struct{})

	web.mu.Lock()
	web.server = server
	web.serverCancel = serverCancel
	web.serverDone = done
	web.running = true
	web.err = ""
	web.mu.Unlock()
	a.webLogf("server started addr=%s", addr)

	go func() {
		defer close(done)
		err := server.Serve(ln)
		if web.finishServer(server, err) && err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.webLogf("server error addr=%s error=%s", addr, err.Error())
		}
		a.webLogf("server stopped addr=%s", addr)
	}()
	return nil
}

func (a *App) stopWebServer(ctx context.Context) {
	a.webLifecycleMu.Lock()
	defer a.webLifecycleMu.Unlock()
	a.stopWebServerLocked(ctx)
}

func (a *App) stopWebServerLocked(ctx context.Context) {
	web := a.web
	if web == nil {
		return
	}
	web.mu.Lock()
	server := web.server
	serverCancel := web.serverCancel
	done := web.serverDone
	web.server = nil
	web.serverCancel = nil
	web.serverDone = nil
	web.running = false
	hadLogTailRefs := len(web.logTailRefs) > 0
	var clientCleanups []webClientCleanup
	for client := range web.clients {
		cleanup := detachWebClientLocked(web, client)
		cleanup.stopLogs = false
		clientCleanups = append(clientCleanups, cleanup)
	}
	web.logTailRefs = map[string]bool{}
	web.mu.Unlock()
	for _, cleanup := range clientCleanups {
		a.finishWebClientCleanup(cleanup)
	}
	if serverCancel != nil {
		serverCancel()
	}
	if server != nil {
		_ = server.Shutdown(ctx)
	}
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
		}
	}
	if hadLogTailRefs {
		a.StopTailLogs()
	}
	a.webLogf("server shutdown requested")
}

func (a *App) setWebError(message string) {
	if a.web == nil {
		return
	}
	a.web.mu.Lock()
	a.web.err = message
	a.web.running = false
	a.web.mu.Unlock()
}

func (web *webRuntime) finishServer(server *http.Server, serveErr error) bool {
	web.mu.Lock()
	defer web.mu.Unlock()
	if web.server != server {
		return false
	}
	web.server = nil
	web.serverCancel = nil
	web.serverDone = nil
	web.running = false
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		web.err = serveErr.Error()
	}
	return true
}

func newWebHTTPServer(addr string, handler http.Handler, baseCtx context.Context) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		BaseContext:       func(net.Listener) context.Context { return baseCtx },
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       60 * time.Second,
	}
}

func (a *App) webStatus() WebStatus {
	cfg, err := a.readWebConfig()
	if err != nil {
		return WebStatus{Enabled: true, Host: defaultWebHost, Port: defaultWebPort, Error: err.Error()}
	}
	running := false
	webErr := ""
	if a.web != nil {
		a.web.mu.Lock()
		running = a.web.running
		webErr = a.web.err
		a.web.mu.Unlock()
	}
	localURL := "http://127.0.0.1:" + cfg.Port
	lanURLs := lanWebURLs(cfg.Port)
	primary := localURL
	if cfg.Host == "0.0.0.0" && len(lanURLs) > 0 {
		primary = lanURLs[0]
	}
	return WebStatus{
		Enabled:              cfg.Enabled,
		Running:              running,
		Host:                 cfg.Host,
		Port:                 cfg.Port,
		LocalURL:             localURL,
		LanURLs:              lanURLs,
		PrimaryURL:           primary,
		UsingDefaultPassword: cfg.UsingDefaultPassword,
		Error:                webErr,
	}
}

func newWebRuntime() *webRuntime {
	return &webRuntime{
		clients:       map[*webClient]bool{},
		loginFailures: map[string]loginFailure{},
		logTailRefs:   map[string]bool{},
	}
}

func lanWebURLs(port string) []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 || isVirtualNetworkInterface(iface.Name) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch value := addr.(type) {
			case *net.IPNet:
				ip = value.IP
			case *net.IPAddr:
				ip = value.IP
			}
			if ip == nil || ip.To4() == nil || ip.IsLoopback() {
				continue
			}
			out = append(out, "http://"+ip.String()+":"+port)
		}
	}
	return out
}

func isVirtualNetworkInterface(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, prefix := range []string{
		"docker", "br-", "veth", "virbr", "vmnet", "vboxnet", "utun", "tun", "tap", "wg", "bridge", "awdl", "llw", "anpi", "ipsec", "gif", "stf", "p2p", "vnic", "vethernet", "cni", "flannel", "cali", "podman", "lxdbr", "incusbr", "nordlynx", "ppp",
	} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	for _, marker := range []string{
		"virtual", "hyper-v", "host-only", "tailscale", "zerotier", "wireguard", "openvpn", "vmware", "wsl",
	} {
		if strings.Contains(name, marker) {
			return true
		}
	}
	return false
}

func (a *App) registerWebRoutes(mux *http.ServeMux, static http.Handler) {
	a.registerWebRoutesWithContext(mux, static, context.Background())
}

func (a *App) registerWebRoutesWithContext(mux *http.ServeMux, static http.Handler, serverCtx context.Context) {
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/session", a.handleWebSession)
	mux.HandleFunc("/api/login", a.handleWebLogin)
	mux.HandleFunc("/api/logout", a.requireWebSession(a.handleWebLogout))
	mux.HandleFunc("/api/rpc", a.requireWebSession(a.handleWebRPC))
	mux.HandleFunc("/ws/events", a.requireWebSession(func(w http.ResponseWriter, r *http.Request) {
		a.handleWebSocket(serverCtx, w, r)
	}))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		static.ServeHTTP(w, r)
	})
}

func (a *App) handleWebSession(w http.ResponseWriter, r *http.Request) {
	session, ok := a.currentWebSession(r)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"authenticated":          ok,
		"using_default_password": a.webStatus().UsingDefaultPassword,
		"usingDefaultPassword":   a.webStatus().UsingDefaultPassword,
		"session_expires_at":     session.ExpiresAt,
		"sessionExpiresAt":       session.ExpiresAt,
	})
}

func (a *App) handleWebLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !sameOrigin(r) {
		a.webLogf("login rejected ip=%s reason=origin", clientIP(r))
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	if a.loginLimited(r) {
		a.webLogf("login limited ip=%s", clientIP(r))
		writeJSON(w, http.StatusTooManyRequests, rpcResponse{OK: false, Error: "登录失败次数过多，请稍后再试"})
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := decodeWebRequest(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, rpcResponse{OK: false, Error: "请求格式错误"})
		return
	}
	cfg, err := a.readWebConfig()
	if err != nil {
		a.webLogf("login failed ip=%s error=%s", clientIP(r), err.Error())
		writeJSON(w, http.StatusInternalServerError, rpcResponse{OK: false, Error: err.Error()})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(cfg.PasswordHash), []byte(req.Password)); err != nil {
		a.recordLoginFailure(r)
		a.webLogf("login failed ip=%s reason=bad_password", clientIP(r))
		writeJSON(w, http.StatusUnauthorized, rpcResponse{OK: false, Error: "访问密码错误"})
		return
	}
	a.clearLoginFailure(r)
	rawID := uuid.NewString()
	now := time.Now().UTC()
	session := webSession{
		IDHash:     hashSessionID(rawID, cfg.SessionSecret),
		CreatedAt:  now.Format(time.RFC3339),
		LastSeenAt: now.Format(time.RFC3339),
		ExpiresAt:  now.Add(7 * 24 * time.Hour).Format(time.RFC3339),
	}
	if err := a.addWebSession(session); err != nil {
		a.webLogf("login failed ip=%s error=%s", clientIP(r), err.Error())
		writeJSON(w, http.StatusInternalServerError, rpcResponse{OK: false, Error: err.Error()})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    rawID,
		Path:     "/",
		Expires:  now.Add(7 * 24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	a.webLogf("login ok ip=%s default_password=%t", clientIP(r), cfg.UsingDefaultPassword)
	writeJSON(w, http.StatusOK, rpcResponse{OK: true, Result: map[string]bool{"usingDefaultPassword": cfg.UsingDefaultPassword}})
}

func (a *App) loginLimited(r *http.Request) bool {
	if a.web == nil {
		return false
	}
	ip := clientIP(r)
	now := time.Now()
	a.web.mu.Lock()
	defer a.web.mu.Unlock()
	failure := a.web.loginFailures[ip]
	return !failure.DelayedTill.IsZero() && now.Before(failure.DelayedTill)
}

func (a *App) recordLoginFailure(r *http.Request) {
	if a.web == nil {
		return
	}
	ip := clientIP(r)
	now := time.Now()
	a.web.mu.Lock()
	defer a.web.mu.Unlock()
	failure := a.web.loginFailures[ip]
	failure.Count++
	if failure.Count >= 20 {
		failure.DelayedTill = now.Add(60 * time.Second)
	} else if failure.Count >= 5 {
		failure.DelayedTill = now.Add(5 * time.Second)
	}
	a.web.loginFailures[ip] = failure
}

func (a *App) clearLoginFailure(r *http.Request) {
	if a.web == nil {
		return
	}
	ip := clientIP(r)
	a.web.mu.Lock()
	delete(a.web.loginFailures, ip)
	a.web.mu.Unlock()
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (a *App) handleWebLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		_ = a.removeWebSession(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	writeJSON(w, http.StatusOK, rpcResponse{OK: true})
}

func (a *App) handleWebRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !sameOrigin(r) {
		a.webLogf("rpc rejected ip=%s reason=origin", clientIP(r))
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	var req rpcRequest
	if err := decodeWebRequest(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, rpcResponse{OK: false, Error: "请求格式错误"})
		return
	}
	handler, ok := a.webRPCHandlers()[req.Method]
	if !ok {
		a.webLogf("rpc unsupported ip=%s method=%s", clientIP(r), req.Method)
		writeJSON(w, http.StatusBadRequest, rpcResponse{OK: false, Error: "Web 不支持该操作：" + req.Method})
		return
	}
	result, err := handler(req.Params)
	if err != nil {
		a.webLogf("rpc failed ip=%s method=%s error=%s", clientIP(r), req.Method, err.Error())
		writeJSON(w, http.StatusOK, rpcResponse{OK: false, Error: err.Error()})
		return
	}
	a.webLogf("rpc ok ip=%s method=%s", clientIP(r), req.Method)
	writeJSON(w, http.StatusOK, rpcResponse{OK: true, Result: result})
}

func decodeWebRequest(w http.ResponseWriter, r *http.Request, out interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, webMaxRequestBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("请求包含多余 JSON")
		}
		return err
	}
	return nil
}

func (a *App) handleWebSocket(serverCtx context.Context, w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return sameOrigin(r) }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &webClient{id: strings.TrimSpace(r.URL.Query().Get("client_id")), conn: conn, send: make(chan webEvent, 32)}
	web := a.web
	if web == nil {
		_ = conn.Close()
		return
	}
	web.mu.Lock()
	if serverCtx.Err() != nil {
		web.mu.Unlock()
		_ = conn.Close()
		return
	}
	client.stopContext = context.AfterFunc(serverCtx, func() { a.closeWebClient(client) })
	web.clients[client] = true
	web.mu.Unlock()
	go func() {
		for event := range client.send {
			if err := conn.WriteJSON(event); err != nil {
				a.closeWebClient(client)
				return
			}
		}
	}()
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				a.closeWebClient(client)
				return
			}
		}
	}()
}

func (a *App) closeWebClient(client *webClient) {
	web := a.web
	if web == nil {
		if client != nil && client.conn != nil {
			_ = client.conn.Close()
		}
		return
	}
	web.mu.Lock()
	cleanup := detachWebClientLocked(web, client)
	web.mu.Unlock()
	a.finishWebClientCleanup(cleanup)
}

func (a *App) emitWeb(event string, payload interface{}) {
	web := a.web
	if web == nil {
		return
	}
	web.mu.Lock()
	var cleanups []webClientCleanup
	for client := range web.clients {
		if client.closed {
			cleanups = append(cleanups, detachWebClientLocked(web, client))
			continue
		}
		select {
		case client.send <- webEvent{Event: event, Payload: payload}:
		default:
			cleanups = append(cleanups, detachWebClientLocked(web, client))
		}
	}
	web.mu.Unlock()
	for _, cleanup := range cleanups {
		a.finishWebClientCleanup(cleanup)
	}
}

func detachWebClientLocked(web *webRuntime, client *webClient) webClientCleanup {
	if !client.closed {
		client.closed = true
		close(client.send)
	}
	delete(web.clients, client)
	cleanup := webClientCleanup{conn: client.conn, stopContext: client.stopContext}
	client.stopContext = nil
	if client.id != "" {
		hadLogRef := web.logTailRefs[client.id]
		delete(web.logTailRefs, client.id)
		cleanup.stopLogs = hadLogRef && len(web.logTailRefs) == 0
	}
	return cleanup
}

func (a *App) finishWebClientCleanup(cleanup webClientCleanup) {
	if cleanup.conn != nil {
		_ = cleanup.conn.Close()
	}
	if cleanup.stopContext != nil {
		cleanup.stopContext()
	}
	if cleanup.stopLogs {
		a.StopTailLogs()
	}
}

func (a *App) requireWebSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := a.currentWebSession(r); !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]interface{}{"authenticated": false})
			return
		}
		next(w, r)
	}
}

func (a *App) currentWebSession(r *http.Request) (webSession, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return webSession{}, false
	}
	cfg, err := a.readWebConfig()
	if err != nil {
		return webSession{}, false
	}
	a.webSessionMu.RLock()
	file, err := a.readWebSessions()
	a.webSessionMu.RUnlock()
	if err != nil {
		return webSession{}, false
	}
	now := time.Now().UTC()
	target := hashSessionID(cookie.Value, cfg.SessionSecret)
	for _, session := range file.Sessions {
		expires, err := time.Parse(time.RFC3339, session.ExpiresAt)
		if err != nil || now.After(expires) {
			continue
		}
		if session.IDHash == target {
			return session, true
		}
	}
	return webSession{}, false
}

func (a *App) readWebSessions() (webSessionFile, error) {
	var file webSessionFile
	if !fileExists(a.webSessionsPath()) {
		return file, nil
	}
	_, err := readJSONFile(a.webSessionsPath(), &file)
	return file, err
}

func (a *App) writeWebSessions(file webSessionFile) error {
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	return writeFilePrivate(a.webSessionsPath(), data)
}

func (a *App) addWebSession(session webSession) error {
	a.webSessionMu.Lock()
	defer a.webSessionMu.Unlock()
	file, err := a.readWebSessions()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	var kept []webSession
	for _, item := range file.Sessions {
		expires, err := time.Parse(time.RFC3339, item.ExpiresAt)
		if err == nil && now.Before(expires) {
			kept = append(kept, item)
		}
	}
	file.Sessions = append(kept, session)
	return a.writeWebSessions(file)
}

func (a *App) removeWebSession(rawID string) error {
	a.webSessionMu.Lock()
	defer a.webSessionMu.Unlock()
	cfg, err := a.readWebConfig()
	if err != nil {
		return err
	}
	target := hashSessionID(rawID, cfg.SessionSecret)
	file, err := a.readWebSessions()
	if err != nil {
		return err
	}
	var kept []webSession
	for _, session := range file.Sessions {
		if session.IDHash != target {
			kept = append(kept, session)
		}
	}
	file.Sessions = kept
	return a.writeWebSessions(file)
}

func (a *App) clearWebSessions() error {
	a.webSessionMu.Lock()
	defer a.webSessionMu.Unlock()
	return a.writeWebSessions(webSessionFile{})
}

func hashSessionID(raw string, secret string) string {
	sum := sha256.Sum256([]byte(secret + ":" + raw))
	return hex.EncodeToString(sum[:])
}

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	expected := "http://" + r.Host
	if r.TLS != nil {
		expected = "https://" + r.Host
	}
	return origin == expected
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (a *App) webRPCHandlers() map[string]webRPCHandler {
	handlers := map[string]webRPCHandler{
		"GetAppStateForProfile":        oneArgValue[string, AppState](a.GetAppStateForProfile),
		"StartHermes":                  a.webLocked(a.StartHermes),
		"StopHermes":                   a.webLocked(a.StopHermes),
		"RestartHermes":                a.webLocked(a.RestartHermes),
		"RebuildHermes":                a.webLocked(a.RebuildHermes),
		"ForceRebuildHermes":           a.webLocked(a.ForceRebuildHermes),
		"TailLogs":                     oneArg[string](a.webStartLogTail),
		"StopTailLogs":                 oneArg[string](a.webStopLogTail),
		"CancelWeixinLogin":            noResult(func() error { a.CancelWeixinLogin(); return nil }),
		"CancelFeishuLogin":            noResult(func() error { a.CancelFeishuLogin(); return nil }),
		"CancelDingTalkLogin":          noResult(func() error { a.CancelDingTalkLogin(); return nil }),
		"StartWeixinLoginForProfile":   oneArg[string](a.StartWeixinLoginForProfile),
		"StartFeishuLoginForProfile":   oneArg[string](a.StartFeishuLoginForProfile),
		"StartDingTalkLoginForProfile": oneArg[string](a.StartDingTalkLoginForProfile),
		"TestModelForProfile":          webLockedOneArg[string](a, a.TestModelForProfile),
		"GetModelProviderPresets":      noParams(func() (interface{}, error) { return a.GetModelProviderPresets(), nil }),
		"ListProfiles":                 noParams(func() (interface{}, error) { return a.ListProfiles() }),
		"GetWebStatus":                 noParams(func() (interface{}, error) { return a.webStatus(), nil }),
		"GetRecommendedResourceLimits": noParams(func() (interface{}, error) {
			return a.GetRecommendedResourceLimits()
		}),
		"SaveComposeSettings":          oneArg[ComposeSettings](a.SaveComposeSettings),
		"SaveProxySettings":            oneArg[ProxySettings](a.SaveProxySettings),
		"SaveModelConfigForProfile":    twoArgs[string, ModelConfig](a.SaveModelConfigForProfile),
		"SaveProviderConfigForProfile": twoArgs[string, ProviderConfig](a.SaveProviderConfigForProfile),
		"SaveWeComConfigForProfile":    twoArgs[string, WeComConfig](a.SaveWeComConfigForProfile),
		"SaveFeishuConfigForProfile":   twoArgs[string, FeishuConfig](a.SaveFeishuConfigForProfile),
		"SaveDingTalkConfigForProfile": twoArgs[string, DingTalkConfig](a.SaveDingTalkConfigForProfile),
		"UnbindPlatformForProfile":     twoArgs[string, string](a.UnbindPlatformForProfile),
		"CreateProfile":                oneArg[CreateProfileRequest](a.CreateProfile),
		"DeleteProfile":                oneArg[DeleteProfileRequest](a.webDeleteProfile),
		"CompleteProfileSetup":         oneArg[string](a.CompleteProfileSetup),
		"UpdateProfileName":            twoArgs[string, string](a.UpdateProfileName),
		"SetProfileEnabled":            twoArgs[string, bool](a.SetProfileEnabled),
		"MoveProfile":                  twoArgs[string, string](a.MoveProfile),
		"SelectProfile":                oneArg[string](a.SelectProfile),
		"BatchCopyProfileConfig":       oneArgValue[BatchProfileConfigRequest, BatchProfileConfigResult](a.BatchCopyProfileConfig),
		"SyncBundledContent":           oneArgValue[BundledContentSyncRequest, BundledContentSyncResult](a.SyncBundledContent),
		"FetchProviderConfigModelListForProfile": func(params []json.RawMessage) (interface{}, error) {
			profileID, err := decodeParam[string](params, 0)
			if err != nil {
				return nil, err
			}
			provider, err := decodeParam[ProviderConfigEntry](params, 1)
			if err != nil {
				return nil, err
			}
			return a.FetchProviderConfigModelListForProfile(profileID, provider)
		},
		"SetHomeChannelForProfile":       threeArgs[string, string, string](a.SetHomeChannelForProfile),
		"SendTestMessageForProfile":      fourArgs[string, string, string, string](a.SendTestMessageForProfile),
		"ListProfileSkillsForProfile":    oneArgValue[string, SkillsState](a.ListProfileSkillsForProfile),
		"GetSkillDetailForProfile":       twoArgsValue[string, string, SkillDetail](a.GetSkillDetailForProfile),
		"DeleteSkillForProfile":          oneArg[DeleteSkillRequest](a.webDeleteSkill),
		"BatchDeleteSkillsForProfile":    oneArg[BatchDeleteSkillsRequest](a.webBatchDeleteSkills),
		"SyncBundledSkillsForProfile":    oneArgValue[string, SyncBundledSkillsResult](a.SyncBundledSkillsForProfile),
		"RestoreDefaultSkillsForProfile": oneArgValue[RestoreDefaultRequest, SyncBundledSkillsResult](a.webRestoreDefaultSkills),
		"RestoreDefaultSoulForProfile":   oneArg[RestoreDefaultRequest](a.webRestoreDefaultSoul),
		"ListSkillHubSkillsForProfile":   twoArgsValue[string, SkillHubQuery, SkillHubState](a.ListSkillHubSkillsForProfile),
		"GetSkillHubDetailForProfile":    twoArgsValue[string, string, SkillHubDetail](a.GetSkillHubDetailForProfile),
		"InstallSkillHubSkillForProfile": twoArgs[string, string](a.InstallSkillHubSkillForProfile),
		"ReadWebTextFile":                twoArgsValue[string, string, string](a.webReadTextFile),
		"SaveWebTextFile":                oneArg[WebTextFileRequest](a.webSaveTextFile),
		"FactoryResetInstance":           a.webLocked(a.FactoryResetInstance),
		"ExportInstanceBackup":           oneArgValue[string, InstanceBackupManifest](a.ExportInstanceBackup),
		"InspectInstanceBackup":          oneArgValue[string, InstanceBackupManifest](a.InspectInstanceBackup),
		"ImportInstanceBackup":           oneArgValue[InstanceBackupImportRequest, InstanceBackupImportResult](a.ImportInstanceBackup),
		"OpenSkillDirectoryForProfile":   twoArgs[string, string](a.OpenSkillDirectoryForProfile),
		"OpenFileManagement": noParams(func() (interface{}, error) {
			return a.webFileManagementURL()
		}),
		"SaveWebSettings":      oneArg[WebSettingsRequest](a.SaveWebSettings),
		"ChangeWebPassword":    twoArgs[string, string](a.ChangeWebPassword),
		"ResetWebPassword":     noResult(a.ResetWebPassword),
		"CheckForUpdates":      oneArgValue[bool, UpdateInfo](a.CheckForUpdates),
		"DismissUpdate":        oneArg[string](a.DismissUpdate),
		"InstallUpdate":        oneArg[string](a.InstallUpdate),
		"SetAutoUpdateEnabled": oneArgValue[bool, UpdateStatus](a.SetAutoUpdateEnabled),
		"RetryPostUpdate":      noParams(func() (interface{}, error) { return a.RetryPostUpdate() }),
		"OpenUpdateURL":        oneArg[string](a.OpenUpdateURL),
		"OpenWebManagement":    noResult(a.OpenWebManagement),
	}
	handlers["ApplyRecommendedDingTalkSettingsForProfile"] = oneArg[string](a.ApplyRecommendedDingTalkSettingsForProfile)
	return handlers
}

func (a *App) webStartLogTail(clientID string) error {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return fmt.Errorf("缺少日志客户端 ID")
	}
	if a.web == nil {
		a.web = newWebRuntime()
	}
	a.web.mu.Lock()
	wasEmpty := len(a.web.logTailRefs) == 0
	a.web.logTailRefs[clientID] = true
	a.web.mu.Unlock()
	if wasEmpty {
		return a.TailLogs()
	}
	return nil
}

func (a *App) webStopLogTail(clientID string) error {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" || a.web == nil {
		return nil
	}
	a.web.mu.Lock()
	hadLogRef := a.web.logTailRefs[clientID]
	delete(a.web.logTailRefs, clientID)
	shouldStop := hadLogRef && len(a.web.logTailRefs) == 0
	a.web.mu.Unlock()
	if shouldStop {
		a.StopTailLogs()
	}
	return nil
}

func (a *App) webDeleteProfile(req DeleteProfileRequest) error {
	id := strings.TrimSpace(req.ID)
	if req.Confirm != id {
		return fmt.Errorf("请输入 Profile ID 确认删除")
	}
	return a.DeleteProfile(id)
}

func (a *App) webDeleteSkill(req DeleteSkillRequest) error {
	if !req.Confirm {
		return fmt.Errorf("请确认删除技能")
	}
	return a.DeleteSkillForProfile(req.ProfileID, req.Path)
}

func (a *App) webBatchDeleteSkills(req BatchDeleteSkillsRequest) error {
	if !req.Confirm {
		return fmt.Errorf("请确认批量删除技能")
	}
	return a.BatchDeleteSkillsForProfile(req.ProfileID, req.Paths)
}

func (a *App) webRestoreDefaultSkills(req RestoreDefaultRequest) (SyncBundledSkillsResult, error) {
	if !req.Confirm {
		return SyncBundledSkillsResult{}, fmt.Errorf("请确认恢复默认技能")
	}
	return a.RestoreDefaultSkillsForProfile(req.ProfileID)
}

func (a *App) webRestoreDefaultSoul(req RestoreDefaultRequest) error {
	if !req.Confirm {
		return fmt.Errorf("请确认恢复默认人格")
	}
	return a.RestoreDefaultSoulForProfile(req.ProfileID)
}

func (a *App) webReadTextFile(profileID string, kind string) (string, error) {
	path, err := a.webTextFilePath(profileID, kind)
	if err != nil {
		return "", err
	}
	return a.ReadTextFile(path)
}

func (a *App) webSaveTextFile(req WebTextFileRequest) error {
	if strings.TrimSpace(req.Kind) == "compose_override" && req.Confirm != "确认" {
		return fmt.Errorf("保存 Docker Compose 覆盖文件前请输入“确认”")
	}
	path, err := a.webTextFilePath(req.ProfileID, req.Kind)
	if err != nil {
		return err
	}
	return a.SaveTextFile(TextFileRequest{Path: path, Content: req.Content, Reason: "before-web-text-save"})
}

func (a *App) webTextFilePath(profileID string, kind string) (string, error) {
	profileID, err := a.resolveProfileID(profileID)
	if err != nil {
		return "", err
	}
	var path string
	switch strings.TrimSpace(kind) {
	case "profile_config":
		path = filepath.Join(a.profileDataDir(profileID), "config.yaml")
	case "profile_env":
		path = filepath.Join(a.profileDataDir(profileID), ".env")
	case "profile_soul":
		path = filepath.Join(a.profileDataDir(profileID), "SOUL.md")
	case "compose_override":
		path = a.overridePath()
	default:
		return "", fmt.Errorf("Web 管理不开放该文件")
	}
	rel, err := filepath.Rel(a.instanceRoot, path)
	if err != nil {
		return "", err
	}
	return rel, nil
}

func (a *App) webFileManagementURL() (string, error) {
	settings, err := a.readComposeSettings()
	if err != nil {
		return "", err
	}
	if !settings.DufsEnabled {
		return "", fmt.Errorf("Dufs 文件管理未开启")
	}
	status, err := a.dufsStatus()
	if err != nil {
		return "", err
	}
	return status.PrimaryURL, nil
}

func (a *App) SaveWebSettings(req WebSettingsRequest) error {
	previous, err := a.readWebConfig()
	if err != nil {
		return err
	}
	if req.Host != "127.0.0.1" && req.Host != "0.0.0.0" {
		return fmt.Errorf("访问范围无效")
	}
	if strings.TrimSpace(req.Port) == "" {
		return fmt.Errorf("端口不能为空")
	}
	port := strings.TrimSpace(req.Port)
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return fmt.Errorf("端口必须是 1 到 65535 之间的数字")
	}
	next := previous
	next.Enabled = req.Enabled
	next.Host = req.Host
	next.Port = port
	next.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	addressChanged := previous.Host != next.Host || previous.Port != next.Port
	if next.Enabled && addressChanged && previous.Port != next.Port {
		probe, err := net.Listen("tcp", net.JoinHostPort(next.Host, next.Port))
		if err != nil {
			return fmt.Errorf("Web 管理监听地址不可用：%w", err)
		}
		if err := probe.Close(); err != nil {
			return err
		}
	}
	if err := a.writeWebConfig(next); err != nil {
		return err
	}
	if previous.Enabled == next.Enabled && !addressChanged {
		return nil
	}
	go func() {
		time.Sleep(200 * time.Millisecond)
		a.stopWebServer(context.Background())
		if err := a.startWebServerChecked(); err != nil {
			a.webLogf("server restart failed, rolling back error=%s", err.Error())
			if writeErr := a.writeWebConfig(previous); writeErr != nil {
				a.setWebError(errors.Join(err, writeErr).Error())
				return
			}
			_ = a.startWebServerChecked()
		}
	}()
	return nil
}

func (a *App) ChangeWebPassword(oldPassword string, newPassword string) error {
	cfg, err := a.readWebConfig()
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(cfg.PasswordHash), []byte(oldPassword)); err != nil {
		return fmt.Errorf("旧访问密码错误")
	}
	return a.setWebPassword(newPassword, newPassword == defaultWebPassword)
}

func (a *App) ResetWebPassword() error {
	return a.setWebPassword(defaultWebPassword, true)
}

func (a *App) setWebPassword(password string, usingDefault bool) error {
	cfg, err := a.readWebConfig()
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	cfg.PasswordHash = string(hash)
	cfg.UsingDefaultPassword = usingDefault
	cfg.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := a.writeWebConfig(cfg); err != nil {
		return err
	}
	return a.clearWebSessions()
}

func (a *App) OpenWebManagement() error {
	url := a.webStatus().PrimaryURL
	if url == "" {
		url = "http://127.0.0.1:" + defaultWebPort
	}
	runtime.BrowserOpenURL(a.ctx, url)
	return nil
}
