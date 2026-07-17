package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"
)

const maxCommandOutputLineBytes = 1024 * 1024

type StreamEvent struct {
	Line string `json:"line"`
	Done bool   `json:"done"`
	Code int    `json:"code"`
}

func (a *App) runComposeStreaming(ctx context.Context, event string, args ...string) error {
	fullArgs := append([]string{"compose"}, args...)
	return a.runStreaming(ctx, event, "docker", fullArgs...)
}

func (a *App) runComposeBlocking(ctx context.Context, args ...string) error {
	if !commandExists("docker") {
		return fmt.Errorf("未找到 docker 命令")
	}
	fullArgs := append([]string{"compose"}, args...)
	cmd := backgroundCommandContext(ctx, "docker", fullArgs...)
	cmd.Dir = a.instanceRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(redact(string(out)))
		if message != "" {
			return fmt.Errorf("%w: %s", err, message)
		}
		return err
	}
	return nil
}

func (a *App) runStreaming(ctx context.Context, event string, name string, args ...string) error {
	if !commandExists(name) {
		return fmt.Errorf("未找到 %s 命令", name)
	}
	cmd := backgroundCommandContext(ctx, name, args...)
	cmd.Dir = a.instanceRoot
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	scanErrors := make(chan error, 2)
	scan := func(reader io.ReadCloser) {
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 64*1024), maxCommandOutputLineBytes)
		for scanner.Scan() {
			a.emit(event, StreamEvent{Line: redact(scanner.Text())})
		}
		scanErrors <- scanner.Err()
	}
	go scan(stdout)
	go scan(stderr)
	var scanErr error
	for range 2 {
		if err := <-scanErrors; err != nil && scanErr == nil {
			scanErr = err
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		}
	}

	err = cmd.Wait()
	if scanErr != nil {
		return fmt.Errorf("读取 Docker 命令输出失败：%w", scanErr)
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			a.emit(event, StreamEvent{Done: true, Code: exitErr.ExitCode()})
		}
		return err
	}
	a.emit(event, StreamEvent{Done: true, Code: 0})
	return nil
}

func (a *App) TailLogs() error {
	ctx, cancel := context.WithCancel(context.Background())
	a.logMu.Lock()
	previous := a.logCancel
	a.logCancel = cancel
	a.logMu.Unlock()
	if previous != nil {
		previous()
	}
	go func() {
		err := a.runComposeStreaming(ctx, "logs:line", "logs", "-f", "hermes")
		if err != nil && ctx.Err() == nil {
			a.emit("logs:line", StreamEvent{Line: redact(err.Error()), Done: true, Code: 1})
		}
	}()
	return nil
}

func (a *App) StopTailLogs() {
	a.logMu.Lock()
	cancel := a.logCancel
	a.logCancel = nil
	a.logMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func portAvailable(port string) bool {
	if strings.TrimSpace(port) == "" {
		return false
	}
	ln, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}
