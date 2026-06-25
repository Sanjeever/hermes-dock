package main

import "testing"

func TestWeixinHelperNoiseFiltersSupervisorShutdown(t *testing.T) {
	line := "WARNING gateway.run: Shutdown context: signal=SIGTERM under_systemd=no parent_pid=126 parent_name=s6-supervise"
	if !isWeixinHelperNoise(line) {
		t.Fatalf("expected shutdown warning to be filtered")
	}
}

func TestWeixinHelperNoiseKeepsUsefulErrors(t *testing.T) {
	line := "failed to connect to Docker daemon"
	if isWeixinHelperNoise(line) {
		t.Fatalf("expected useful error to pass through")
	}
}
