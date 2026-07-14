//go:build !windows

package main

import "os/exec"

func configureBackgroundCommand(cmd *exec.Cmd) *exec.Cmd {
	return cmd
}
