package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kbinani/screenshot"
)

const (
	hostRPALeaseDuration = 30 * time.Second
	hostRPAMaxText       = 64 << 10
)

type hostRPACapabilities struct {
	Supported                       bool   `json:"supported"`
	Backend                         string `json:"backend"`
	Reason                          string `json:"reason,omitempty"`
	Windows                         bool   `json:"windows"`
	Mouse                           bool   `json:"mouse"`
	Keyboard                        bool   `json:"keyboard"`
	UnicodeText                     bool   `json:"unicode_text"`
	RequiresAccessibilityPermission bool   `json:"requires_accessibility_permission"`
}

type hostRPAWindow struct {
	ID        string `json:"id"`
	PID       int    `json:"pid"`
	Title     string `json:"title"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Active    bool   `json:"active"`
	Minimized bool   `json:"minimized"`
}

type hostRPAWindowRequest struct {
	ID string `json:"id"`
}

type hostRPAMouseRequest struct {
	Action           string `json:"action"`
	Display          int    `json:"display"`
	X                int    `json:"x"`
	Y                int    `json:"y"`
	FromX            int    `json:"from_x"`
	FromY            int    `json:"from_y"`
	ToX              int    `json:"to_x"`
	ToY              int    `json:"to_y"`
	Button           string `json:"button"`
	Count            int    `json:"count"`
	DX               int    `json:"dx"`
	DY               int    `json:"dy"`
	DurationMS       int    `json:"duration_ms"`
	ExpectedWindowID string `json:"expected_window_id"`
}

type hostRPAKeyboardRequest struct {
	Action           string   `json:"action"`
	Key              string   `json:"key"`
	Keys             []string `json:"keys"`
	Text             string   `json:"text"`
	Count            int      `json:"count"`
	IntervalMS       int      `json:"interval_ms"`
	ExpectedWindowID string   `json:"expected_window_id"`
}

type hostRPAConflictError struct {
	message string
}

func (e hostRPAConflictError) Error() string {
	return e.message
}

func (a *App) handleHostRPAInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	capabilities := hostRPAPlatformCapabilities()
	profile, err := hostRPAProfile(r)
	if err != nil {
		writeHostError(w, http.StatusBadRequest, err)
		return
	}
	a.hostRPAMu.Lock()
	capabilitiesBusy := a.hostRPAOwner != "" && a.hostRPAOwner != profile && time.Now().Before(a.hostRPAExpiresAt)
	leaseOwned := a.hostRPAOwner == profile && time.Now().Before(a.hostRPAExpiresAt)
	a.hostRPAMu.Unlock()
	writeHostJSON(w, http.StatusOK, map[string]interface{}{
		"capabilities":  capabilities,
		"busy":          capabilitiesBusy,
		"lease_owned":   leaseOwned,
		"lease_seconds": int(hostRPALeaseDuration.Seconds()),
	})
}

func (a *App) handleHostRPARelease(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	profile, err := hostRPAProfile(r)
	if err != nil {
		writeHostError(w, http.StatusBadRequest, err)
		return
	}
	a.hostRPAMu.Lock()
	defer a.hostRPAMu.Unlock()
	if a.hostRPAOwner != "" && time.Now().Before(a.hostRPAExpiresAt) && a.hostRPAOwner != profile {
		writeHostError(w, http.StatusConflict, errors.New("桌面自动化正被其他 profile 使用"))
		return
	}
	a.hostRPAOwner = ""
	a.hostRPAExpiresAt = time.Time{}
	writeHostJSON(w, http.StatusOK, map[string]bool{"released": true})
}

func (a *App) handleHostRPAWindows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := requireHostRPASupported(); err != nil {
		writeHostError(w, http.StatusServiceUnavailable, err)
		return
	}
	windows, err := hostRPAListWindows()
	if err != nil {
		writeHostError(w, http.StatusServiceUnavailable, err)
		return
	}
	writeHostJSON(w, http.StatusOK, map[string]interface{}{"windows": windows})
}

func (a *App) handleHostRPAActiveWindow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := requireHostRPASupported(); err != nil {
		writeHostError(w, http.StatusServiceUnavailable, err)
		return
	}
	window, err := hostRPAActiveWindow()
	if err != nil {
		writeHostError(w, http.StatusServiceUnavailable, err)
		return
	}
	writeHostJSON(w, http.StatusOK, window)
}

func (a *App) handleHostRPAActivateWindow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req hostRPAWindowRequest
	if !decodeHostJSON(w, r, 64<<10, &req) {
		return
	}
	req.ID = strings.TrimSpace(req.ID)
	if req.ID == "" {
		writeHostError(w, http.StatusBadRequest, errors.New("窗口 ID 不能为空"))
		return
	}
	a.withHostRPAAction(w, r, func() error {
		return hostRPAActivateWindow(req.ID)
	})
}

func (a *App) handleHostRPAMouse(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if err := requireHostRPASupported(); err != nil {
			writeHostError(w, http.StatusServiceUnavailable, err)
			return
		}
		x, y, err := hostRPAMousePosition()
		if err != nil {
			writeHostError(w, http.StatusServiceUnavailable, err)
			return
		}
		display, localX, localY := hostRPALocalPoint(x, y)
		writeHostJSON(w, http.StatusOK, map[string]int{
			"display": display,
			"x":       localX,
			"y":       localY,
		})
	case http.MethodPost:
		var req hostRPAMouseRequest
		if !decodeHostJSON(w, r, 64<<10, &req) {
			return
		}
		if err := validateHostRPAMouseRequest(&req); err != nil {
			writeHostError(w, http.StatusBadRequest, err)
			return
		}
		if err := translateHostRPAMouseCoordinates(&req); err != nil {
			writeHostError(w, http.StatusBadRequest, err)
			return
		}
		a.withHostRPAAction(w, r, func() error {
			if req.ExpectedWindowID != "" {
				if err := checkHostRPAExpectedWindow(req.ExpectedWindowID); err != nil {
					return err
				}
			}
			return hostRPAPerformMouse(req)
		})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *App) handleHostRPAKeyboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req hostRPAKeyboardRequest
	if !decodeHostJSON(w, r, hostRPAMaxText+(64<<10), &req) {
		return
	}
	if err := validateHostRPAKeyboardRequest(&req); err != nil {
		writeHostError(w, http.StatusBadRequest, err)
		return
	}
	a.withHostRPAAction(w, r, func() error {
		if err := checkHostRPAExpectedWindow(req.ExpectedWindowID); err != nil {
			return err
		}
		return hostRPAPerformKeyboard(req)
	})
}

func (a *App) withHostRPAAction(w http.ResponseWriter, r *http.Request, action func() error) {
	if err := requireHostRPASupported(); err != nil {
		writeHostError(w, http.StatusServiceUnavailable, err)
		return
	}
	profile, err := hostRPAProfile(r)
	if err != nil {
		writeHostError(w, http.StatusBadRequest, err)
		return
	}
	a.hostRPAMu.Lock()
	defer a.hostRPAMu.Unlock()
	now := time.Now()
	if a.hostRPAOwner != "" && now.Before(a.hostRPAExpiresAt) && a.hostRPAOwner != profile {
		writeHostError(w, http.StatusConflict, errors.New("桌面自动化正被其他 profile 使用"))
		return
	}
	a.hostRPAOwner = profile
	a.hostRPAExpiresAt = now.Add(hostRPALeaseDuration)
	if err := action(); err != nil {
		status := http.StatusServiceUnavailable
		var conflict hostRPAConflictError
		if errors.As(err, &conflict) {
			status = http.StatusConflict
		}
		writeHostError(w, status, err)
		return
	}
	a.hostRPAExpiresAt = time.Now().Add(hostRPALeaseDuration)
	writeHostJSON(w, http.StatusOK, map[string]bool{"performed": true})
}

func requireHostRPASupported() error {
	capabilities := hostRPAPlatformCapabilities()
	if capabilities.Supported {
		return nil
	}
	if capabilities.Reason != "" {
		return errors.New(capabilities.Reason)
	}
	return errors.New("当前宿主机不支持桌面自动化")
}

func hostRPAProfile(r *http.Request) (string, error) {
	profile := strings.TrimSpace(r.Header.Get("X-Hermes-Profile"))
	if profile == "" {
		return "default", nil
	}
	if len(profile) > 64 {
		return "", errors.New("profile ID 无效")
	}
	for _, char := range profile {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
			return "", errors.New("profile ID 无效")
		}
	}
	return profile, nil
}

func checkHostRPAExpectedWindow(expected string) error {
	if expected == "" {
		return errors.New("必须提供预期前台窗口 ID")
	}
	active, err := hostRPAActiveWindow()
	if err != nil {
		return err
	}
	if active.ID != expected {
		return hostRPAConflictError{message: fmt.Sprintf("前台窗口已经变化：当前是 %s", active.ID)}
	}
	return nil
}

func validateHostRPAMouseRequest(req *hostRPAMouseRequest) error {
	req.Action = strings.ToLower(strings.TrimSpace(req.Action))
	req.Button = strings.ToLower(strings.TrimSpace(req.Button))
	req.ExpectedWindowID = strings.TrimSpace(req.ExpectedWindowID)
	if req.DurationMS < 0 || req.DurationMS > 10000 {
		return errors.New("duration_ms 必须在 0-10000 之间")
	}
	switch req.Action {
	case "move":
		return nil
	case "click":
		if req.ExpectedWindowID == "" {
			return errors.New("必须提供预期前台窗口 ID")
		}
		if req.Button == "" {
			req.Button = "left"
		}
		if req.Button != "left" && req.Button != "right" && req.Button != "middle" {
			return errors.New("鼠标按键只支持 left、right 或 middle")
		}
		if req.Count == 0 {
			req.Count = 1
		}
		if req.Count < 1 || req.Count > 3 {
			return errors.New("点击次数必须在 1-3 之间")
		}
	case "drag":
		if req.ExpectedWindowID == "" {
			return errors.New("必须提供预期前台窗口 ID")
		}
		if req.Button == "" {
			req.Button = "left"
		}
		if req.Button != "left" && req.Button != "right" && req.Button != "middle" {
			return errors.New("鼠标按键只支持 left、right 或 middle")
		}
	case "scroll":
		if req.ExpectedWindowID == "" {
			return errors.New("必须提供预期前台窗口 ID")
		}
		if req.DX == 0 && req.DY == 0 {
			return errors.New("滚动距离不能为空")
		}
		if req.DX < -100 || req.DX > 100 || req.DY < -100 || req.DY > 100 {
			return errors.New("单次滚动距离必须在 -100 到 100 之间")
		}
	default:
		return errors.New("鼠标动作只支持 move、click、drag 或 scroll")
	}
	return nil
}

func validateHostRPAKeyboardRequest(req *hostRPAKeyboardRequest) error {
	req.Action = strings.ToLower(strings.TrimSpace(req.Action))
	req.Key = strings.ToLower(strings.TrimSpace(req.Key))
	req.ExpectedWindowID = strings.TrimSpace(req.ExpectedWindowID)
	if req.ExpectedWindowID == "" {
		return errors.New("必须提供预期前台窗口 ID")
	}
	if req.IntervalMS < 0 || req.IntervalMS > 1000 {
		return errors.New("interval_ms 必须在 0-1000 之间")
	}
	switch req.Action {
	case "press":
		if req.Key == "" {
			return errors.New("按键不能为空")
		}
		if req.Count == 0 {
			req.Count = 1
		}
		if req.Count < 1 || req.Count > 10 {
			return errors.New("按键次数必须在 1-10 之间")
		}
	case "hotkey":
		if len(req.Keys) < 2 || len(req.Keys) > 4 {
			return errors.New("快捷键必须包含 2-4 个按键")
		}
		for index := range req.Keys {
			req.Keys[index] = strings.ToLower(strings.TrimSpace(req.Keys[index]))
			if req.Keys[index] == "" {
				return errors.New("快捷键包含空按键")
			}
		}
	case "type":
		if req.Text == "" {
			return errors.New("输入文本不能为空")
		}
		if len(req.Text) > hostRPAMaxText {
			return errors.New("输入文本超过 64 KiB 限制")
		}
		if !utf8.ValidString(req.Text) {
			return errors.New("输入文本必须是有效 UTF-8")
		}
	default:
		return errors.New("键盘动作只支持 press、hotkey 或 type")
	}
	return nil
}

func translateHostRPAMouseCoordinates(req *hostRPAMouseRequest) error {
	count := screenshot.NumActiveDisplays()
	if count == 0 {
		return errors.New("没有可用的宿主机显示器")
	}
	if req.Display < 0 || req.Display >= count {
		return fmt.Errorf("显示器编号必须在 0-%d 之间", count-1)
	}
	bounds := screenshot.GetDisplayBounds(req.Display)
	translate := func(x, y int) (int, int, error) {
		if x < 0 || x >= bounds.Dx() || y < 0 || y >= bounds.Dy() {
			return 0, 0, fmt.Errorf("坐标超出显示器范围 %dx%d", bounds.Dx(), bounds.Dy())
		}
		return bounds.Min.X + x, bounds.Min.Y + y, nil
	}
	var err error
	switch req.Action {
	case "drag":
		req.FromX, req.FromY, err = translate(req.FromX, req.FromY)
		if err != nil {
			return err
		}
		req.ToX, req.ToY, err = translate(req.ToX, req.ToY)
	default:
		req.X, req.Y, err = translate(req.X, req.Y)
	}
	return err
}

func hostRPALocalPoint(globalX, globalY int) (int, int, int) {
	for display := 0; display < screenshot.NumActiveDisplays(); display++ {
		bounds := screenshot.GetDisplayBounds(display)
		if globalX >= bounds.Min.X && globalX < bounds.Max.X && globalY >= bounds.Min.Y && globalY < bounds.Max.Y {
			return display, globalX - bounds.Min.X, globalY - bounds.Min.Y
		}
	}
	return -1, globalX, globalY
}
