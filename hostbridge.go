package main

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	hostBridgeAddress   = "0.0.0.0:9877"
	hostBridgeMaxBody   = 2 << 20
	hostBridgeMaxOutput = 1 << 20
)

type hostBridgeRuntime struct {
	server  *http.Server
	cancel  context.CancelFunc
	running bool
	err     string
	limit   chan struct{}
}

type hostExecRequest struct {
	Command        string   `json:"command"`
	Program        string   `json:"program"`
	Args           []string `json:"args"`
	Cwd            string   `json:"cwd"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

type hostExecResponse struct {
	ExitCode        int    `json:"exit_code"`
	Stdout          string `json:"stdout"`
	Stderr          string `json:"stderr"`
	TimedOut        bool   `json:"timed_out"`
	StdoutTruncated bool   `json:"stdout_truncated"`
	StderrTruncated bool   `json:"stderr_truncated"`
}

type cappedBuffer struct {
	mu        sync.Mutex
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *cappedBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	original := len(data)
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		b.truncated = b.truncated || original > 0
		return original, nil
	}
	if len(data) > remaining {
		data = data[:remaining]
		b.truncated = true
	}
	_, _ = b.buf.Write(data)
	return original, nil
}

func (b *cappedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func (a *App) startHostBridge() error {
	if a.readComposeSettings().HostControlEnabled != "true" {
		return nil
	}
	a.hostBridgeMu.Lock()
	defer a.hostBridgeMu.Unlock()
	if a.hostBridge != nil && a.hostBridge.running {
		return nil
	}
	listener, err := net.Listen("tcp", a.hostBridgeAddr)
	if err != nil {
		a.hostBridge = &hostBridgeRuntime{err: err.Error()}
		return fmt.Errorf("启动宿主机控制服务失败：%w", err)
	}
	runtimeState := &hostBridgeRuntime{running: true, limit: make(chan struct{}, 4)}
	bridgeCtx, cancel := context.WithCancel(context.Background())
	runtimeState.cancel = cancel
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/info", a.requireHostBridgeToken(a.handleHostInfo))
	mux.HandleFunc("/v1/exec", a.requireHostBridgeToken(a.handleHostExec(runtimeState)))
	mux.HandleFunc("/v1/files/read", a.requireHostBridgeToken(a.handleHostFileRead))
	mux.HandleFunc("/v1/files/write", a.requireHostBridgeToken(a.handleHostFileWrite))
	mux.HandleFunc("/v1/files/stat", a.requireHostBridgeToken(a.handleHostFileStat))
	mux.HandleFunc("/v1/files/list", a.requireHostBridgeToken(a.handleHostFileList))
	mux.HandleFunc("/v1/files/mkdir", a.requireHostBridgeToken(a.handleHostFileMkdir))
	mux.HandleFunc("/v1/files/move", a.requireHostBridgeToken(a.handleHostFileMove))
	mux.HandleFunc("/v1/notify", a.requireHostBridgeToken(a.handleHostNotify))
	mux.HandleFunc("/v1/clipboard/text", a.requireHostBridgeToken(a.handleHostClipboardText))
	mux.HandleFunc("/v1/processes", a.requireHostBridgeToken(a.handleHostProcesses))
	mux.HandleFunc("/v1/ports", a.requireHostBridgeToken(a.handleHostPorts))
	mux.HandleFunc("/v1/displays", a.requireHostBridgeToken(a.handleHostDisplays))
	mux.HandleFunc("/v1/screenshot", a.requireHostBridgeToken(a.handleHostScreenshot))
	mux.HandleFunc("/v1/open", a.requireHostBridgeToken(a.handleHostOpen))
	mux.HandleFunc("/v1/launch", a.requireHostBridgeToken(a.handleHostLaunch))
	runtimeState.server = &http.Server{
		Handler:           mux,
		BaseContext:       func(net.Listener) context.Context { return bridgeCtx },
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      31 * time.Minute,
		IdleTimeout:       60 * time.Second,
	}
	a.hostBridge = runtimeState
	go func() {
		err := runtimeState.server.Serve(listener)
		a.hostBridgeMu.Lock()
		defer a.hostBridgeMu.Unlock()
		if a.hostBridge != runtimeState {
			return
		}
		runtimeState.running = false
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			runtimeState.err = err.Error()
		}
	}()
	return nil
}

func (a *App) stopHostBridge(ctx context.Context) {
	a.hostBridgeMu.Lock()
	runtimeState := a.hostBridge
	a.hostBridge = nil
	a.hostBridgeMu.Unlock()
	if runtimeState != nil && runtimeState.server != nil {
		runtimeState.cancel()
		_ = runtimeState.server.Shutdown(ctx)
	}
}

func (a *App) syncHostBridge(enabled bool) error {
	if !enabled {
		a.stopHostBridge(context.Background())
		return nil
	}
	return a.startHostBridge()
}

func (a *App) hostBridgeStatus() HostBridgeStatus {
	a.hostBridgeMu.RLock()
	defer a.hostBridgeMu.RUnlock()
	status := HostBridgeStatus{
		Enabled: a.readComposeSettings().HostControlEnabled == "true",
		Address: a.hostBridgeAddr,
	}
	if a.hostBridge != nil {
		status.Running = a.hostBridge.running
		status.Error = a.hostBridge.err
	}
	return status
}

func (a *App) requireHostBridgeToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a.readComposeSettings().HostControlEnabled != "true" {
			writeHostJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "host bridge disabled"})
			return
		}
		expected, err := os.ReadFile(a.hostBridgeTokenPath())
		if err != nil {
			writeHostJSON(w, http.StatusInternalServerError, map[string]string{"error": "host bridge token unavailable"})
			return
		}
		provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(string(expected))), []byte(provided)) != 1 {
			writeHostJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next(w, r)
	}
}

func (a *App) handleHostInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	home, _ := os.UserHomeDir()
	hostname, _ := os.Hostname()
	writeHostJSON(w, http.StatusOK, map[string]string{
		"os":             runtime.GOOS,
		"arch":           runtime.GOARCH,
		"hostname":       hostname,
		"home":           home,
		"path_separator": string(os.PathSeparator),
	})
}

func (a *App) handleHostExec(runtimeState *hostBridgeRuntime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, hostBridgeMaxBody)
		var req hostExecRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			writeHostJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if (strings.TrimSpace(req.Command) == "") == (strings.TrimSpace(req.Program) == "") {
			writeHostJSON(w, http.StatusBadRequest, map[string]string{"error": "set exactly one of command or program"})
			return
		}
		select {
		case runtimeState.limit <- struct{}{}:
			defer func() { <-runtimeState.limit }()
		case <-r.Context().Done():
			writeHostJSON(w, http.StatusRequestTimeout, map[string]string{"error": "request cancelled"})
			return
		}

		timeout := req.TimeoutSeconds
		if timeout <= 0 {
			timeout = 120
		}
		if timeout > 1800 {
			timeout = 1800
		}
		ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeout)*time.Second)
		defer cancel()
		cmd, err := hostCommand(ctx, req)
		if err != nil {
			writeHostJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if req.Cwd != "" {
			cmd.Dir = req.Cwd
		}
		cmd.Env = os.Environ()
		stdout := &cappedBuffer{limit: hostBridgeMaxOutput}
		stderr := &cappedBuffer{limit: hostBridgeMaxOutput}
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		err = cmd.Run()
		response := hostExecResponse{
			ExitCode:        0,
			Stdout:          stdout.String(),
			Stderr:          stderr.String(),
			TimedOut:        errors.Is(ctx.Err(), context.DeadlineExceeded),
			StdoutTruncated: stdout.truncated,
			StderrTruncated: stderr.truncated,
		}
		if err != nil {
			response.ExitCode = -1
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				response.ExitCode = exitErr.ExitCode()
			} else if !response.TimedOut && response.Stderr == "" {
				response.Stderr = err.Error()
			}
		}
		writeHostJSON(w, http.StatusOK, response)
	}
}

func hostCommand(ctx context.Context, req hostExecRequest) (*exec.Cmd, error) {
	if strings.TrimSpace(req.Program) != "" {
		return exec.CommandContext(ctx, req.Program, req.Args...), nil
	}
	switch runtime.GOOS {
	case "windows":
		return exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", req.Command), nil
	case "darwin":
		return exec.CommandContext(ctx, "/bin/zsh", "-lc", req.Command), nil
	case "linux":
		return exec.CommandContext(ctx, "/bin/sh", "-lc", req.Command), nil
	default:
		return nil, fmt.Errorf("unsupported host operating system: %s", runtime.GOOS)
	}
}

func writeHostJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
