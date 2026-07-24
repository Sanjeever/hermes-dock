//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/google/uuid"
	"golang.org/x/sys/windows"
)

const secureWindowsShare = windows.FILE_SHARE_READ | windows.FILE_SHARE_WRITE | windows.FILE_SHARE_DELETE

type secureFileRenameInformation struct {
	ReplaceIfExists uint32
	RootDirectory   windows.Handle
	FileNameLength  uint32
	FileName        [1]uint16
}

func openFileBeneath(root string, path string) (*os.File, error) {
	components, err := securePathComponents(root, path)
	if err != nil {
		return nil, err
	}
	parent, err := openWindowsDirectoryBeneath(root, components[:len(components)-1])
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(parent)
	handle, err := openWindowsRelative(parent, components[len(components)-1], windows.FILE_GENERIC_READ|windows.SYNCHRONIZE, windows.FILE_OPEN, windows.FILE_NON_DIRECTORY_FILE)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	return os.NewFile(uintptr(handle), path), nil
}

func atomicWriteFileBeneath(root string, path string, data []byte, mode os.FileMode) error {
	components, err := securePathComponents(root, path)
	if err != nil {
		return err
	}
	parent, err := openWindowsDirectoryBeneath(root, components[:len(components)-1])
	if err != nil {
		return err
	}
	defer windows.CloseHandle(parent)
	tempName := ".hermes-dock-write-" + uuid.NewString()
	handle, err := openWindowsRelative(parent, tempName, windows.FILE_GENERIC_WRITE|windows.DELETE|windows.SYNCHRONIZE, windows.FILE_CREATE, windows.FILE_NON_DIRECTORY_FILE)
	if err != nil {
		return &os.PathError{Op: "create", Path: filepath.Join(filepath.Dir(path), tempName), Err: err}
	}
	file := os.NewFile(uintptr(handle), filepath.Join(filepath.Dir(path), tempName))
	committed := false
	defer func() {
		if !committed {
			deleteOnClose := byte(1)
			_ = windows.SetFileInformationByHandle(handle, windows.FileDispositionInfo, &deleteOnClose, 1)
		}
		_ = file.Close()
	}()
	if err := file.Chmod(mode); err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := renameWindowsRelative(handle, parent, components[len(components)-1]); err != nil {
		return &os.PathError{Op: "rename", Path: path, Err: err}
	}
	committed = true
	return file.Close()
}

func openWindowsDirectoryBeneath(root string, components []string) (windows.Handle, error) {
	rootPath, err := windows.UTF16PtrFromString(filepath.Clean(root))
	if err != nil {
		return windows.InvalidHandle, err
	}
	current, err := windows.CreateFile(rootPath, windows.FILE_GENERIC_READ|windows.SYNCHRONIZE, secureWindowsShare, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return windows.InvalidHandle, &os.PathError{Op: "open", Path: root, Err: err}
	}
	if err := rejectWindowsReparsePoint(current); err != nil {
		windows.CloseHandle(current)
		return windows.InvalidHandle, err
	}
	currentPath := filepath.Clean(root)
	for _, component := range components {
		next, openErr := openWindowsRelative(current, component, windows.FILE_GENERIC_READ|windows.SYNCHRONIZE, windows.FILE_OPEN, windows.FILE_DIRECTORY_FILE)
		if openErr != nil {
			windows.CloseHandle(current)
			return windows.InvalidHandle, &os.PathError{Op: "open", Path: filepath.Join(currentPath, component), Err: openErr}
		}
		windows.CloseHandle(current)
		current = next
		currentPath = filepath.Join(currentPath, component)
	}
	return current, nil
}

func openWindowsRelative(parent windows.Handle, name string, access uint32, disposition uint32, options uint32) (windows.Handle, error) {
	objectName, err := windows.NewNTUnicodeString(name)
	if err != nil {
		return windows.InvalidHandle, err
	}
	attributes := &windows.OBJECT_ATTRIBUTES{
		RootDirectory: parent,
		ObjectName:    objectName,
		Attributes:    windows.OBJ_CASE_INSENSITIVE | windows.OBJ_DONT_REPARSE,
	}
	attributes.Length = uint32(unsafe.Sizeof(*attributes))
	var handle windows.Handle
	var status windows.IO_STATUS_BLOCK
	err = windows.NtCreateFile(&handle, access, attributes, &status, nil, windows.FILE_ATTRIBUTE_NORMAL, secureWindowsShare, disposition, options|windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT, 0, 0)
	if err != nil {
		return windows.InvalidHandle, err
	}
	if err := rejectWindowsReparsePoint(handle); err != nil {
		windows.CloseHandle(handle)
		return windows.InvalidHandle, err
	}
	return handle, nil
}

func rejectWindowsReparsePoint(handle windows.Handle) error {
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
		return err
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return fmt.Errorf("路径包含 Windows reparse point")
	}
	return nil
}

func renameWindowsRelative(file windows.Handle, parent windows.Handle, name string) error {
	utf16Name, err := windows.UTF16FromString(name)
	if err != nil {
		return err
	}
	nameLength := len(utf16Name)*2 - 2
	var layout secureFileRenameInformation
	bufferSize := int(unsafe.Offsetof(layout.FileName)) + nameLength
	buffer := make([]byte, bufferSize)
	info := (*secureFileRenameInformation)(unsafe.Pointer(&buffer[0]))
	info.ReplaceIfExists = windows.FILE_RENAME_REPLACE_IF_EXISTS
	info.RootDirectory = parent
	info.FileNameLength = uint32(nameLength)
	copy((*[windows.MAX_LONG_PATH]uint16)(unsafe.Pointer(&info.FileName[0]))[:nameLength/2:nameLength/2], utf16Name)
	var status windows.IO_STATUS_BLOCK
	return windows.NtSetInformationFile(file, &status, &buffer[0], uint32(bufferSize), windows.FileRenameInformation)
}
