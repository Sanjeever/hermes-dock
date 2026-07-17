package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	hermesHome   = "/opt/data"
	manifestPath = "/opt/data/.dock/profiles-runtime.json"
	statusPath   = "/opt/data/.dock/profile-status.json"
)

const maxProfileLogLineBytes = 1024 * 1024

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type RuntimeManifest struct {
	SchemaVersion int                      `json:"schemaVersion"`
	Generation    string                   `json:"generation"`
	GeneratedAt   string                   `json:"generatedAt"`
	Profiles      []RuntimeManifestProfile `json:"profiles"`
}

type RuntimeManifestProfile struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	Home      string `json:"home"`
	IsDefault bool   `json:"isDefault"`
	Runnable  bool   `json:"runnable"`
	Reason    string `json:"reason"`
}

type RuntimeStatus struct {
	Generation string                          `json:"generation"`
	UpdatedAt  string                          `json:"updatedAt"`
	Profiles   map[string]RuntimeProfileStatus `json:"profiles"`
}

type RuntimeProfileStatus struct {
	Enabled      bool   `json:"enabled"`
	State        string `json:"state"`
	PID          int    `json:"pid"`
	StartedAt    string `json:"startedAt"`
	LastExitCode int    `json:"lastExitCode"`
	RestartCount int    `json:"restartCount"`
	Message      string `json:"message"`
}

type supervisor struct {
	mu       sync.Mutex
	status   RuntimeStatus
	cancel   context.CancelFunc
	profiles []RuntimeManifestProfile
}

func main() {
	manifest, err := readManifest()
	if err != nil {
		fmt.Printf("[runner] %s\n", err)
		select {}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	ctx, cancel := context.WithCancel(ctx)
	s := &supervisor{
		cancel:   cancel,
		profiles: manifest.Profiles,
		status:   initialRuntimeStatus(manifest),
	}
	s.writeStatus()

	var wg sync.WaitGroup
	runnable := 0
	for _, profile := range manifest.Profiles {
		if !profile.Enabled {
			fmt.Printf("[runner] %s disabled\n", profile.ID)
			continue
		}
		if !profile.Runnable {
			fmt.Printf("[%s] skipped: %s\n", profile.ID, firstNonEmpty(profile.Reason, "not_configured"))
			continue
		}
		runnable++
		wg.Add(1)
		go func(profile RuntimeManifestProfile) {
			defer wg.Done()
			s.runProfile(ctx, profile)
		}(profile)
	}
	if runnable == 0 {
		fmt.Println("[runner] no runnable profiles")
	}

	<-ctx.Done()
	s.cancel()
	wg.Wait()
	s.markStopped()
}

func initialRuntimeStatus(manifest RuntimeManifest) RuntimeStatus {
	status := RuntimeStatus{
		Generation: manifest.Generation,
		Profiles:   map[string]RuntimeProfileStatus{},
	}
	for _, profile := range manifest.Profiles {
		state := "disabled"
		message := ""
		if profile.Enabled {
			if profile.Runnable {
				state = "starting"
			} else {
				state = "not_configured"
				message = profile.Reason
			}
		}
		status.Profiles[profile.ID] = RuntimeProfileStatus{
			Enabled: profile.Enabled,
			State:   state,
			Message: message,
		}
	}
	return status
}

func readManifest() (RuntimeManifest, error) {
	var manifest RuntimeManifest
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return manifest, fmt.Errorf("读取运行清单失败：%w", err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, fmt.Errorf("运行清单不是有效 JSON：%w", err)
	}
	return manifest, nil
}

func (s *supervisor) runProfile(ctx context.Context, profile RuntimeManifestProfile) {
	restarts := 0
	failures := []time.Time{}
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		exitCode, err := s.runOnce(ctx, profile, restarts)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			fmt.Printf("[%s] exited: %s\n", profile.ID, err)
		}
		now := time.Now()
		failures = append(failures, now)
		cutoff := now.Add(-5 * time.Minute)
		kept := failures[:0]
		for _, item := range failures {
			if item.After(cutoff) {
				kept = append(kept, item)
			}
		}
		failures = kept
		if tooManyRecentFailures(failures, now) {
			s.update(profile.ID, RuntimeProfileStatus{
				Enabled:      true,
				State:        "failed",
				LastExitCode: exitCode,
				RestartCount: restarts,
				Message:      "连续失败次数过多，已停止自动重启",
			})
			fmt.Printf("[%s] profile stopped after repeated failures\n", profile.ID)
			return
		}
		restarts++
		time.Sleep(time.Duration(restarts) * time.Second)
	}
}

func tooManyRecentFailures(failures []time.Time, now time.Time) bool {
	cutoff := now.Add(-5 * time.Minute)
	count := 0
	for _, failure := range failures {
		if failure.After(cutoff) {
			count++
		}
	}
	return count >= 5
}

func (s *supervisor) runOnce(ctx context.Context, profile RuntimeManifestProfile, restarts int) (int, error) {
	env, err := buildEnv(profile)
	if err != nil {
		s.update(profile.ID, RuntimeProfileStatus{
			Enabled:      true,
			State:        "failed",
			RestartCount: restarts,
			Message:      err.Error(),
		})
		return 1, err
	}
	args := []string{"gateway", "run"}
	if !profile.IsDefault {
		args = []string{"-p", profile.ID, "gateway", "run"}
	}
	cmd := exec.CommandContext(ctx, "hermes", args...)
	cmd.Dir = hermesHome
	cmd.Env = env
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 1, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return 1, err
	}
	if err := cmd.Start(); err != nil {
		return 1, err
	}
	startedAt := time.Now().UTC().Format(time.RFC3339)
	s.update(profile.ID, RuntimeProfileStatus{
		Enabled:      true,
		State:        "running",
		PID:          cmd.Process.Pid,
		StartedAt:    startedAt,
		RestartCount: restarts,
	})
	fmt.Printf("[%s] started pid=%d\n", profile.ID, cmd.Process.Pid)

	logErrors := make(chan error, 2)
	go prefixLines(profile.ID, stdout, logErrors)
	go prefixLines(profile.ID, stderr, logErrors)
	var logErr error
	for range 2 {
		if err := <-logErrors; err != nil && logErr == nil {
			logErr = err
			_ = cmd.Process.Kill()
		}
	}

	err = cmd.Wait()
	if logErr != nil {
		err = fmt.Errorf("读取 profile 日志失败：%w", logErr)
	}
	exitCode := 0
	if err != nil {
		exitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}
	s.update(profile.ID, RuntimeProfileStatus{
		Enabled:      true,
		State:        "exited",
		LastExitCode: exitCode,
		RestartCount: restarts,
		Message:      fmt.Sprintf("进程退出，代码 %d", exitCode),
	})
	return exitCode, err
}

func buildEnv(profile RuntimeManifestProfile) ([]string, error) {
	env := os.Environ()
	env = setEnv(env, "HERMES_HOME", hermesHome)
	env = setEnv(env, "HERMES_DOCK_PROFILE", profile.ID)
	env = setEnv(env, "HERMES_DOCK_PROFILE_HOME", profile.Home)
	var envPath string
	if profile.IsDefault {
		envPath = filepath.Join(hermesHome, ".env")
	} else {
		envPath = filepath.Join(hermesHome, "profiles", profile.ID, ".env")
	}
	vars, err := readEnvFile(envPath)
	if err != nil {
		return nil, err
	}
	for key, value := range vars {
		env = setEnv(env, key, value)
	}
	return env, nil
}

func readEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	out := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		if !envKeyPattern.MatchString(key) {
			return nil, fmt.Errorf("环境变量名称无效：%q", key)
		}
		out[key] = unquoteEnv(strings.TrimSpace(parts[1]))
	}
	return out, scanner.Err()
}

func unquoteEnv(value string) string {
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = strings.TrimPrefix(strings.TrimSuffix(value, "\""), "\"")
		value = strings.ReplaceAll(value, "\\\"", "\"")
		value = strings.ReplaceAll(value, "\\\\", "\\")
	}
	return value
}

func setEnv(env []string, key string, value string) []string {
	prefix := key + "="
	item := prefix + value
	for i, existing := range env {
		if strings.HasPrefix(existing, prefix) {
			env[i] = item
			return env
		}
	}
	return append(env, item)
}

func prefixLines(profileID string, reader io.Reader, errors chan<- error) {
	errors <- prefixLinesTo(os.Stdout, profileID, reader)
}

func prefixLinesTo(writer io.Writer, profileID string, reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), maxProfileLogLineBytes)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fmt.Fprintf(writer, "[%s] %s\n", profileID, redact(line))
	}
	return scanner.Err()
}

func (s *supervisor) update(id string, next RuntimeProfileStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.status.Profiles[id]
	if next.Enabled {
		current.Enabled = true
	}
	if next.State != "" {
		current.State = next.State
	}
	if next.PID != 0 || next.State != "running" {
		current.PID = next.PID
	}
	if next.StartedAt != "" {
		current.StartedAt = next.StartedAt
	}
	if next.LastExitCode != 0 || next.State == "exited" || next.State == "failed" {
		current.LastExitCode = next.LastExitCode
	}
	current.RestartCount = next.RestartCount
	current.Message = next.Message
	s.status.Profiles[id] = current
	s.writeStatusLocked()
}

func (s *supervisor) writeStatus() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeStatusLocked()
}

func (s *supervisor) writeStatusLocked() {
	s.status.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	_ = os.MkdirAll(filepath.Dir(statusPath), 0755)
	data, err := json.MarshalIndent(s.status, "", "  ")
	if err != nil {
		return
	}
	_ = atomicWriteStatus(append(data, '\n'))
}

func atomicWriteStatus(data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(statusPath), ".profile-status-*")
	if err != nil {
		return err
	}
	tmp := file.Name()
	defer os.Remove(tmp)
	if err := file.Chmod(0644); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, statusPath)
}

func (s *supervisor) markStopped() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, profile := range s.profiles {
		status := s.status.Profiles[profile.ID]
		if status.State == "running" || status.State == "starting" {
			status.State = "exited"
			status.PID = 0
			status.Message = "runner stopped"
			s.status.Profiles[profile.ID] = status
		}
	}
	s.writeStatusLocked()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func redact(value string) string {
	secrets := []string{"token", "secret", "api_key", "apikey", "password", "authorization"}
	lower := strings.ToLower(value)
	for _, key := range secrets {
		idx := strings.Index(lower, key)
		if idx < 0 {
			continue
		}
		eq := strings.IndexAny(value[idx:], "=:")
		if eq < 0 {
			continue
		}
		start := idx + eq + 1
		end := start
		for end < len(value) && value[end] != ' ' && value[end] != ',' {
			end++
		}
		if end > start {
			value = value[:start] + "***" + value[end:]
			lower = strings.ToLower(value)
		}
	}
	return value
}
