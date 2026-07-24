//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"golang.org/x/sys/unix"
)

func openFileBeneath(root string, path string) (*os.File, error) {
	components, err := securePathComponents(root, path)
	if err != nil {
		return nil, err
	}
	parentFD, err := openDirectoryBeneath(root, components[:len(components)-1])
	if err != nil {
		return nil, err
	}
	defer unix.Close(parentFD)
	fd, err := unix.Openat(parentFD, components[len(components)-1], unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	return os.NewFile(uintptr(fd), path), nil
}

func atomicWriteFileBeneath(root string, path string, data []byte, mode os.FileMode) error {
	components, err := securePathComponents(root, path)
	if err != nil {
		return err
	}
	parentFD, err := openDirectoryBeneath(root, components[:len(components)-1])
	if err != nil {
		return err
	}
	defer unix.Close(parentFD)
	tempName := ".hermes-dock-write-" + uuid.NewString()
	fd, err := unix.Openat(parentFD, tempName, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, uint32(mode.Perm()))
	if err != nil {
		return &os.PathError{Op: "create", Path: filepath.Join(filepath.Dir(path), tempName), Err: err}
	}
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = unix.Unlinkat(parentFD, tempName, 0)
		}
	}()
	file := os.NewFile(uintptr(fd), filepath.Join(filepath.Dir(path), tempName))
	if err := file.Chmod(mode); err != nil {
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
	if err := unix.Renameat(parentFD, tempName, parentFD, components[len(components)-1]); err != nil {
		return &os.PathError{Op: "rename", Path: path, Err: err}
	}
	removeTemp = false
	return nil
}

func openDirectoryBeneath(root string, components []string) (int, error) {
	currentFD, err := unix.Open(filepath.Clean(root), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return -1, &os.PathError{Op: "open", Path: root, Err: err}
	}
	currentPath := filepath.Clean(root)
	for _, component := range components {
		nextFD, openErr := unix.Openat(currentFD, component, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		if openErr != nil {
			unix.Close(currentFD)
			return -1, &os.PathError{Op: "open", Path: filepath.Join(currentPath, component), Err: openErr}
		}
		if err := unix.Close(currentFD); err != nil {
			unix.Close(nextFD)
			return -1, fmt.Errorf("关闭安全路径目录失败：%w", err)
		}
		currentFD = nextFD
		currentPath = filepath.Join(currentPath, component)
	}
	return currentFD, nil
}
