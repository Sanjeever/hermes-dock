package main

import (
	"context"
	"os/exec"
)

func backgroundCommand(name string, args ...string) *exec.Cmd {
	return configureBackgroundCommand(exec.Command(name, args...))
}

func backgroundCommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return configureBackgroundCommand(exec.CommandContext(ctx, name, args...))
}
