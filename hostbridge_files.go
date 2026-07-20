package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	hostBridgeMaxFile        = 16 << 20
	hostBridgeMaxFileRequest = 24 << 20
)

type hostPathRequest struct {
	Path string `json:"path"`
}

type hostFileWriteRequest struct {
	Path          string `json:"path"`
	ContentBase64 string `json:"content_base64"`
	CreateParents bool   `json:"create_parents"`
	Overwrite     bool   `json:"overwrite"`
}

type hostFileMoveRequest struct {
	Source        string `json:"source"`
	Target        string `json:"target"`
	CreateParents bool   `json:"create_parents"`
	Overwrite     bool   `json:"overwrite"`
}

type hostFileInfo struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Size       int64  `json:"size"`
	Mode       string `json:"mode"`
	IsDir      bool   `json:"is_dir"`
	IsSymlink  bool   `json:"is_symlink"`
	LinkTarget string `json:"link_target,omitempty"`
	ModifiedAt string `json:"modified_at"`
}

func (a *App) handleHostFileRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req hostPathRequest
	if !decodeHostJSON(w, r, hostBridgeMaxBody, &req) {
		return
	}
	path, err := absoluteHostPath(req.Path)
	if err != nil {
		writeHostError(w, http.StatusBadRequest, err)
		return
	}
	file, err := os.Open(path)
	if err != nil {
		writeHostError(w, http.StatusBadRequest, err)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		writeHostError(w, http.StatusInternalServerError, err)
		return
	}
	if !info.Mode().IsRegular() {
		writeHostError(w, http.StatusBadRequest, errors.New("只能读取普通文件"))
		return
	}
	if info.Size() > hostBridgeMaxFile {
		writeHostError(w, http.StatusRequestEntityTooLarge, fmt.Errorf("文件超过 %d MiB 限制", hostBridgeMaxFile>>20))
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, hostBridgeMaxFile+1))
	if err != nil {
		writeHostError(w, http.StatusInternalServerError, err)
		return
	}
	writeHostJSON(w, http.StatusOK, map[string]interface{}{
		"path":           path,
		"size":           len(data),
		"modified_at":    info.ModTime().UTC().Format(time.RFC3339),
		"content_base64": base64.StdEncoding.EncodeToString(data),
	})
}

func (a *App) handleHostFileWrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	release, err := a.beginExclusiveOperation("写入宿主机文件")
	if err != nil {
		writeHostError(w, http.StatusConflict, err)
		return
	}
	defer release()
	var req hostFileWriteRequest
	if !decodeHostJSON(w, r, hostBridgeMaxFileRequest, &req) {
		return
	}
	path, err := absoluteHostPath(req.Path)
	if err != nil {
		writeHostError(w, http.StatusBadRequest, err)
		return
	}
	data, err := base64.StdEncoding.DecodeString(req.ContentBase64)
	if err != nil {
		writeHostError(w, http.StatusBadRequest, errors.New("content_base64 无效"))
		return
	}
	if len(data) > hostBridgeMaxFile {
		writeHostError(w, http.StatusRequestEntityTooLarge, fmt.Errorf("文件超过 %d MiB 限制", hostBridgeMaxFile>>20))
		return
	}
	mode := os.FileMode(0644)
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			writeHostError(w, http.StatusBadRequest, errors.New("目标路径是目录"))
			return
		}
		if !req.Overwrite {
			writeHostError(w, http.StatusConflict, errors.New("目标文件已存在"))
			return
		}
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		writeHostError(w, http.StatusBadRequest, err)
		return
	}
	if req.CreateParents {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			writeHostError(w, http.StatusBadRequest, err)
			return
		}
	}
	if err := writeHostFile(path, data, mode); err != nil {
		writeHostError(w, http.StatusBadRequest, err)
		return
	}
	writeHostJSON(w, http.StatusOK, map[string]interface{}{"path": path, "size": len(data)})
}

func (a *App) handleHostFileStat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req hostPathRequest
	if !decodeHostJSON(w, r, hostBridgeMaxBody, &req) {
		return
	}
	path, err := absoluteHostPath(req.Path)
	if err != nil {
		writeHostError(w, http.StatusBadRequest, err)
		return
	}
	info, err := os.Lstat(path)
	if err != nil {
		writeHostError(w, http.StatusBadRequest, err)
		return
	}
	result, err := hostFileInfoFor(path, info)
	if err != nil {
		writeHostError(w, http.StatusInternalServerError, err)
		return
	}
	writeHostJSON(w, http.StatusOK, result)
}

func (a *App) handleHostFileList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req hostPathRequest
	if !decodeHostJSON(w, r, hostBridgeMaxBody, &req) {
		return
	}
	path, err := absoluteHostPath(req.Path)
	if err != nil {
		writeHostError(w, http.StatusBadRequest, err)
		return
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		writeHostError(w, http.StatusBadRequest, err)
		return
	}
	if len(entries) > 10000 {
		writeHostError(w, http.StatusRequestEntityTooLarge, errors.New("目录条目超过 10000 个"))
		return
	}
	files := make([]hostFileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := os.Lstat(filepath.Join(path, entry.Name()))
		if err != nil {
			writeHostError(w, http.StatusInternalServerError, err)
			return
		}
		fileInfo, err := hostFileInfoFor(filepath.Join(path, entry.Name()), info)
		if err != nil {
			writeHostError(w, http.StatusInternalServerError, err)
			return
		}
		files = append(files, fileInfo)
	}
	writeHostJSON(w, http.StatusOK, map[string]interface{}{"path": path, "entries": files})
}

func (a *App) handleHostFileMkdir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	release, err := a.beginExclusiveOperation("创建宿主机目录")
	if err != nil {
		writeHostError(w, http.StatusConflict, err)
		return
	}
	defer release()
	var req hostPathRequest
	if !decodeHostJSON(w, r, hostBridgeMaxBody, &req) {
		return
	}
	path, err := absoluteHostPath(req.Path)
	if err == nil {
		err = os.MkdirAll(path, 0755)
	}
	if err != nil {
		writeHostError(w, http.StatusBadRequest, err)
		return
	}
	writeHostJSON(w, http.StatusOK, map[string]string{"path": path})
}

func (a *App) handleHostFileMove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	release, err := a.beginExclusiveOperation("移动宿主机文件")
	if err != nil {
		writeHostError(w, http.StatusConflict, err)
		return
	}
	defer release()
	var req hostFileMoveRequest
	if !decodeHostJSON(w, r, hostBridgeMaxBody, &req) {
		return
	}
	source, err := absoluteHostPath(req.Source)
	if err != nil {
		writeHostError(w, http.StatusBadRequest, err)
		return
	}
	target, err := absoluteHostPath(req.Target)
	if err != nil {
		writeHostError(w, http.StatusBadRequest, err)
		return
	}
	if info, err := os.Lstat(target); err == nil {
		if !req.Overwrite {
			writeHostError(w, http.StatusConflict, errors.New("目标路径已存在"))
			return
		}
		if info.IsDir() {
			writeHostError(w, http.StatusConflict, errors.New("不支持覆盖已有目录"))
			return
		}
		if runtime.GOOS == "windows" {
			if err := os.Remove(target); err != nil {
				writeHostError(w, http.StatusBadRequest, err)
				return
			}
		}
	} else if !os.IsNotExist(err) {
		writeHostError(w, http.StatusBadRequest, err)
		return
	}
	if req.CreateParents {
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			writeHostError(w, http.StatusBadRequest, err)
			return
		}
	}
	if err := os.Rename(source, target); err != nil {
		writeHostError(w, http.StatusBadRequest, err)
		return
	}
	writeHostJSON(w, http.StatusOK, map[string]string{"source": source, "target": target})
}

func absoluteHostPath(path string) (string, error) {
	if path == "" {
		return "", errors.New("路径不能为空")
	}
	if !filepath.IsAbs(path) {
		return "", errors.New("宿主机路径必须是绝对路径")
	}
	return filepath.Clean(path), nil
}

func hostFileInfoFor(path string, info os.FileInfo) (hostFileInfo, error) {
	result := hostFileInfo{
		Name:       info.Name(),
		Path:       path,
		Size:       info.Size(),
		Mode:       info.Mode().String(),
		IsDir:      info.IsDir(),
		IsSymlink:  info.Mode()&os.ModeSymlink != 0,
		ModifiedAt: info.ModTime().UTC().Format(time.RFC3339),
	}
	if result.IsSymlink {
		var err error
		result.LinkTarget, err = os.Readlink(path)
		if err != nil {
			return hostFileInfo{}, err
		}
	}
	return result, nil
}

func writeHostFile(path string, data []byte, mode os.FileMode) error {
	if runtime.GOOS == "windows" {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
		if err != nil {
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
		return file.Close()
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".hermes-host-write-*")
	if err != nil {
		return err
	}
	tmp := file.Name()
	defer os.Remove(tmp)
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
	return os.Rename(tmp, path)
}

func decodeHostJSON(w http.ResponseWriter, r *http.Request, maxBytes int64, target interface{}) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeHostError(w, http.StatusBadRequest, errors.New("请求格式错误"))
		return false
	}
	return true
}

func writeHostError(w http.ResponseWriter, status int, err error) {
	writeHostJSON(w, status, map[string]string{"error": err.Error()})
}
