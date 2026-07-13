package main

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"net/http"
	"os/exec"
	goRuntime "runtime"
	"strings"

	"github.com/google/uuid"
	"github.com/kbinani/screenshot"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	hostBridgeMaxClipboard  = 1 << 20
	hostBridgeMaxScreenshot = 25 << 20
)

type hostNotificationRequest struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

type hostClipboardRequest struct {
	Text string `json:"text"`
}

type hostScreenshotRequest struct {
	Display int `json:"display"`
}

type hostOpenRequest struct {
	Target string `json:"target"`
}

type hostLaunchRequest struct {
	Program string   `json:"program"`
	Args    []string `json:"args"`
	Cwd     string   `json:"cwd"`
}

func (a *App) handleHostNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req hostNotificationRequest
	if !decodeHostJSON(w, r, 64<<10, &req) {
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Message = strings.TrimSpace(req.Message)
	if req.Title == "" {
		req.Title = "Hermes"
	}
	if req.Message == "" {
		writeHostError(w, http.StatusBadRequest, errors.New("通知内容不能为空"))
		return
	}
	if err := a.ensureHostNotifications(); err != nil {
		writeHostError(w, http.StatusServiceUnavailable, err)
		return
	}
	err := wailsRuntime.SendNotification(a.ctx, wailsRuntime.NotificationOptions{
		ID:    uuid.NewString(),
		Title: req.Title,
		Body:  req.Message,
	})
	if err != nil {
		writeHostError(w, http.StatusInternalServerError, err)
		return
	}
	writeHostJSON(w, http.StatusOK, map[string]bool{"sent": true})
}

func (a *App) ensureHostNotifications() error {
	a.notificationMu.Lock()
	defer a.notificationMu.Unlock()
	if a.notificationReady {
		return nil
	}
	if a.ctx == nil {
		return errors.New("桌面运行时未就绪")
	}
	if err := wailsRuntime.InitializeNotifications(a.ctx); err != nil {
		return fmt.Errorf("初始化宿主机通知失败：%w", err)
	}
	authorized, err := wailsRuntime.RequestNotificationAuthorization(a.ctx)
	if err != nil {
		wailsRuntime.CleanupNotifications(a.ctx)
		return fmt.Errorf("请求宿主机通知权限失败：%w", err)
	}
	if !authorized {
		wailsRuntime.CleanupNotifications(a.ctx)
		return errors.New("宿主机通知权限未授予")
	}
	a.notificationReady = true
	return nil
}

func (a *App) cleanupHostNotifications() {
	a.notificationMu.Lock()
	defer a.notificationMu.Unlock()
	if a.notificationReady && a.ctx != nil {
		wailsRuntime.CleanupNotifications(a.ctx)
	}
	a.notificationReady = false
}

func (a *App) handleHostClipboardText(w http.ResponseWriter, r *http.Request) {
	if a.ctx == nil {
		writeHostError(w, http.StatusServiceUnavailable, errors.New("桌面运行时未就绪"))
		return
	}
	switch r.Method {
	case http.MethodGet:
		value, err := wailsRuntime.ClipboardGetText(a.ctx)
		if err != nil {
			writeHostError(w, http.StatusInternalServerError, err)
			return
		}
		if len(value) > hostBridgeMaxClipboard {
			writeHostError(w, http.StatusRequestEntityTooLarge, errors.New("剪贴板文本超过 1 MiB 限制"))
			return
		}
		writeHostJSON(w, http.StatusOK, map[string]string{"text": value})
	case http.MethodPost:
		var req hostClipboardRequest
		if !decodeHostJSON(w, r, hostBridgeMaxClipboard+(64<<10), &req) {
			return
		}
		if len(req.Text) > hostBridgeMaxClipboard {
			writeHostError(w, http.StatusRequestEntityTooLarge, errors.New("剪贴板文本超过 1 MiB 限制"))
			return
		}
		if err := wailsRuntime.ClipboardSetText(a.ctx, req.Text); err != nil {
			writeHostError(w, http.StatusInternalServerError, err)
			return
		}
		writeHostJSON(w, http.StatusOK, map[string]bool{"written": true})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *App) handleHostDisplays(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	count := screenshot.NumActiveDisplays()
	displays := make([]map[string]int, 0, count)
	for index := 0; index < count; index++ {
		bounds := screenshot.GetDisplayBounds(index)
		displays = append(displays, map[string]int{
			"index":  index,
			"x":      bounds.Min.X,
			"y":      bounds.Min.Y,
			"width":  bounds.Dx(),
			"height": bounds.Dy(),
		})
	}
	writeHostJSON(w, http.StatusOK, map[string]interface{}{"displays": displays})
}

func (a *App) handleHostScreenshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req hostScreenshotRequest
	if !decodeHostJSON(w, r, 64<<10, &req) {
		return
	}
	count := screenshot.NumActiveDisplays()
	if count == 0 {
		writeHostError(w, http.StatusServiceUnavailable, errors.New("没有可用的宿主机显示器"))
		return
	}
	if req.Display < 0 || req.Display >= count {
		writeHostError(w, http.StatusBadRequest, fmt.Errorf("显示器编号必须在 0-%d 之间", count-1))
		return
	}
	bounds := screenshot.GetDisplayBounds(req.Display)
	image, err := screenshot.CaptureRect(bounds)
	if err != nil {
		writeHostError(w, http.StatusServiceUnavailable, fmt.Errorf("宿主机截图失败，请检查屏幕录制权限：%w", err))
		return
	}
	var data bytes.Buffer
	if err := png.Encode(&data, image); err != nil {
		writeHostError(w, http.StatusInternalServerError, err)
		return
	}
	if data.Len() > hostBridgeMaxScreenshot {
		writeHostError(w, http.StatusRequestEntityTooLarge, errors.New("宿主机截图超过 25 MiB 限制"))
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("X-Screenshot-Width", fmt.Sprintf("%d", bounds.Dx()))
	w.Header().Set("X-Screenshot-Height", fmt.Sprintf("%d", bounds.Dy()))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data.Bytes())
}

func (a *App) handleHostOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req hostOpenRequest
	if !decodeHostJSON(w, r, 256<<10, &req) {
		return
	}
	req.Target = strings.TrimSpace(req.Target)
	if req.Target == "" {
		writeHostError(w, http.StatusBadRequest, errors.New("打开目标不能为空"))
		return
	}
	cmd, err := hostOpenCommand(req.Target)
	if err == nil {
		err = cmd.Start()
	}
	if err != nil {
		writeHostError(w, http.StatusBadRequest, err)
		return
	}
	writeHostJSON(w, http.StatusOK, map[string]bool{"opened": true})
}

func (a *App) handleHostLaunch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req hostLaunchRequest
	if !decodeHostJSON(w, r, 256<<10, &req) {
		return
	}
	req.Program = strings.TrimSpace(req.Program)
	if req.Program == "" {
		writeHostError(w, http.StatusBadRequest, errors.New("应用程序不能为空"))
		return
	}
	cmd := exec.Command(req.Program, req.Args...)
	if req.Cwd != "" {
		cwd, err := absoluteHostPath(req.Cwd)
		if err != nil {
			writeHostError(w, http.StatusBadRequest, err)
			return
		}
		cmd.Dir = cwd
	}
	if err := cmd.Start(); err != nil {
		writeHostError(w, http.StatusBadRequest, err)
		return
	}
	writeHostJSON(w, http.StatusOK, map[string]int{"pid": cmd.Process.Pid})
}

func hostOpenCommand(target string) (*exec.Cmd, error) {
	switch goRuntime.GOOS {
	case "windows":
		return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", target), nil
	case "darwin":
		return exec.Command("open", target), nil
	case "linux":
		return exec.Command("xdg-open", target), nil
	default:
		return nil, fmt.Errorf("unsupported host operating system: %s", goRuntime.GOOS)
	}
}
