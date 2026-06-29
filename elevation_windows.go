//go:build windows && !dev

package main

func ensureElevated() (bool, error) {
	return false, nil
}
