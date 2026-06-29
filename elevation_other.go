//go:build !darwin && !linux && !windows && !dev && !bindings

package main

func ensureElevated() (bool, error) {
	return false, nil
}
