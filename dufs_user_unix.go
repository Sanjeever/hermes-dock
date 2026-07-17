//go:build !windows

package main

import (
	"fmt"
	"os"
)

func dufsContainerUser() string {
	return fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid())
}
