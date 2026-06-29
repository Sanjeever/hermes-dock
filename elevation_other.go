//go:build !darwin && !linux && !windows

package main

func ensureElevated() (bool, error) {
	return false, nil
}
