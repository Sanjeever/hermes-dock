package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strings"
)

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

	done := make(chan struct{})
	scan := func(scanner *bufio.Scanner) {
		for scanner.Scan() {
			a.emit(event, StreamEvent{Line: redact(scanner.Text())})
		}
		done <- struct{}{}
	}
	go scan(bufio.NewScanner(stdout))
	go scan(bufio.NewScanner(stderr))
	<-done
	<-done

	err = cmd.Wait()
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
	a.StopTailLogs()
	ctx, cancel := context.WithCancel(context.Background())
	a.logCancel = cancel
	go func() {
		err := a.runComposeStreaming(ctx, "logs:line", "logs", "-f", "hermes")
		if err != nil && ctx.Err() == nil {
			a.emit("logs:line", StreamEvent{Line: redact(err.Error()), Done: true, Code: 1})
		}
	}()
	return nil
}

func (a *App) StopTailLogs() {
	if a.logCancel != nil {
		a.logCancel()
		a.logCancel = nil
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
