//go:build windows

package main

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	windowsInputMouse           = 0
	windowsInputKeyboard        = 1
	windowsKeyEventExtended     = 0x0001
	windowsKeyEventKeyUp        = 0x0002
	windowsKeyEventUnicode      = 0x0004
	windowsMouseEventMove       = 0x0001
	windowsMouseEventLeftDown   = 0x0002
	windowsMouseEventLeftUp     = 0x0004
	windowsMouseEventRightDown  = 0x0008
	windowsMouseEventRightUp    = 0x0010
	windowsMouseEventMiddleDown = 0x0020
	windowsMouseEventMiddleUp   = 0x0040
	windowsMouseEventWheel      = 0x0800
	windowsMouseEventHorizontal = 0x1000
	windowsShowRestore          = 9
	windowsWheelDelta           = 120
)

var (
	windowsUser32                = windows.NewLazySystemDLL("user32.dll")
	windowsProcEnumWindows       = windowsUser32.NewProc("EnumWindows")
	windowsProcGetWindowTextW    = windowsUser32.NewProc("GetWindowTextW")
	windowsProcGetWindowTextLenW = windowsUser32.NewProc("GetWindowTextLengthW")
	windowsProcGetWindowRect     = windowsUser32.NewProc("GetWindowRect")
	windowsProcGetWindowPID      = windowsUser32.NewProc("GetWindowThreadProcessId")
	windowsProcGetForeground     = windowsUser32.NewProc("GetForegroundWindow")
	windowsProcIsWindow          = windowsUser32.NewProc("IsWindow")
	windowsProcIsWindowVisible   = windowsUser32.NewProc("IsWindowVisible")
	windowsProcIsIconic          = windowsUser32.NewProc("IsIconic")
	windowsProcShowWindow        = windowsUser32.NewProc("ShowWindow")
	windowsProcSetForeground     = windowsUser32.NewProc("SetForegroundWindow")
	windowsProcSetCursorPos      = windowsUser32.NewProc("SetCursorPos")
	windowsProcGetCursorPos      = windowsUser32.NewProc("GetCursorPos")
	windowsProcSendInput         = windowsUser32.NewProc("SendInput")
)

type windowsRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type windowsPoint struct {
	X int32
	Y int32
}

type windowsInput struct {
	Type uint32
	_    uint32
	Data [32]byte
}

type windowsMouseInput struct {
	DX        int32
	DY        int32
	MouseData uint32
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

type windowsKeyboardInput struct {
	VirtualKey uint16
	ScanCode   uint16
	Flags      uint32
	Time       uint32
	ExtraInfo  uintptr
}

type windowsEnumContext struct {
	active  uintptr
	windows []hostRPAWindow
}

var windowsEnumWindowCallbackAddress = syscall.NewCallback(windowsEnumWindowCallback)

func hostRPAPlatformCapabilities() hostRPACapabilities {
	return hostRPACapabilities{
		Supported:   true,
		Backend:     "windows-sendinput",
		Windows:     true,
		Mouse:       true,
		Keyboard:    true,
		UnicodeText: true,
	}
}

func hostRPAListWindows() ([]hostRPAWindow, error) {
	active, _, _ := windowsProcGetForeground.Call()
	context := &windowsEnumContext{active: active, windows: make([]hostRPAWindow, 0)}
	ok, _, callErr := windowsProcEnumWindows.Call(windowsEnumWindowCallbackAddress, uintptr(unsafe.Pointer(context)))
	runtime.KeepAlive(context)
	if ok == 0 {
		return nil, fmt.Errorf("枚举 Windows 窗口失败：%w", callErr)
	}
	return context.windows, nil
}

func windowsEnumWindowCallback(hwnd uintptr, contextPointer uintptr) uintptr {
	context := (*windowsEnumContext)(unsafe.Pointer(contextPointer))
	visible, _, _ := windowsProcIsWindowVisible.Call(hwnd)
	if visible == 0 {
		return 1
	}
	title := windowsWindowTitle(hwnd)
	if title == "" {
		return 1
	}
	var rect windowsRect
	ok, _, _ := windowsProcGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if ok == 0 || rect.Right <= rect.Left || rect.Bottom <= rect.Top {
		return 1
	}
	var pid uint32
	windowsProcGetWindowPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	minimized, _, _ := windowsProcIsIconic.Call(hwnd)
	context.windows = append(context.windows, hostRPAWindow{
		ID:        windowsWindowID(hwnd),
		PID:       int(pid),
		Title:     title,
		X:         int(rect.Left),
		Y:         int(rect.Top),
		Width:     int(rect.Right - rect.Left),
		Height:    int(rect.Bottom - rect.Top),
		Active:    hwnd == context.active,
		Minimized: minimized != 0,
	})
	return 1
}

func hostRPAActiveWindow() (hostRPAWindow, error) {
	hwnd, _, _ := windowsProcGetForeground.Call()
	if hwnd == 0 {
		return hostRPAWindow{}, errors.New("没有活动窗口")
	}
	return windowsWindowInfo(hwnd)
}

func hostRPAActivateWindow(id string) error {
	hwnd, err := parseWindowsWindowID(id)
	if err != nil {
		return err
	}
	valid, _, _ := windowsProcIsWindow.Call(hwnd)
	if valid == 0 {
		return errors.New("窗口已经不存在")
	}
	windowsProcShowWindow.Call(hwnd, windowsShowRestore)
	ok, _, _ := windowsProcSetForeground.Call(hwnd)
	if ok == 0 {
		return errors.New("激活窗口失败，目标程序可能以管理员身份运行")
	}
	time.Sleep(100 * time.Millisecond)
	active, _, _ := windowsProcGetForeground.Call()
	if active != hwnd {
		return errors.New("窗口未进入前台")
	}
	return nil
}

func hostRPAMousePosition() (int, int, error) {
	var point windowsPoint
	ok, _, callErr := windowsProcGetCursorPos.Call(uintptr(unsafe.Pointer(&point)))
	if ok == 0 {
		return 0, 0, fmt.Errorf("读取鼠标位置失败：%w", callErr)
	}
	return int(point.X), int(point.Y), nil
}

func hostRPAPerformMouse(req hostRPAMouseRequest) error {
	switch req.Action {
	case "move":
		return windowsMoveMouse(req.X, req.Y, req.DurationMS)
	case "click":
		if err := windowsMoveMouse(req.X, req.Y, req.DurationMS); err != nil {
			return err
		}
		down, up := windowsMouseButtonFlags(req.Button)
		for index := 0; index < req.Count; index++ {
			if err := windowsSendMouseButton(down, up); err != nil {
				return err
			}
			if index+1 < req.Count {
				time.Sleep(80 * time.Millisecond)
			}
		}
		return nil
	case "drag":
		if err := windowsMoveMouse(req.FromX, req.FromY, 0); err != nil {
			return err
		}
		down, up := windowsMouseButtonFlags(req.Button)
		if err := windowsSendMouseFlag(down, 0); err != nil {
			return err
		}
		moveErr := windowsMoveMouse(req.ToX, req.ToY, req.DurationMS)
		upErr := windowsSendMouseFlag(up, 0)
		return errors.Join(moveErr, upErr)
	case "scroll":
		if err := windowsMoveMouse(req.X, req.Y, 0); err != nil {
			return err
		}
		if req.DY != 0 {
			if err := windowsSendMouseFlag(windowsMouseEventWheel, uint32(int32(req.DY*windowsWheelDelta))); err != nil {
				return err
			}
		}
		if req.DX != 0 {
			return windowsSendMouseFlag(windowsMouseEventHorizontal, uint32(int32(req.DX*windowsWheelDelta)))
		}
		return nil
	default:
		return errors.New("不支持的鼠标动作")
	}
}

func hostRPAPerformKeyboard(req hostRPAKeyboardRequest) error {
	switch req.Action {
	case "press":
		key, err := windowsVirtualKey(req.Key)
		if err != nil {
			return err
		}
		for index := 0; index < req.Count; index++ {
			if err := windowsSendVirtualKeys([]uint16{key}); err != nil {
				return err
			}
			if req.IntervalMS > 0 && index+1 < req.Count {
				time.Sleep(time.Duration(req.IntervalMS) * time.Millisecond)
			}
		}
		return nil
	case "hotkey":
		keys := make([]uint16, 0, len(req.Keys))
		for _, name := range req.Keys {
			key, err := windowsVirtualKey(name)
			if err != nil {
				return err
			}
			keys = append(keys, key)
		}
		return windowsSendVirtualKeys(keys)
	case "type":
		return windowsTypeText(req.Text, req.IntervalMS)
	default:
		return errors.New("不支持的键盘动作")
	}
}

func windowsWindowInfo(hwnd uintptr) (hostRPAWindow, error) {
	valid, _, _ := windowsProcIsWindow.Call(hwnd)
	if valid == 0 {
		return hostRPAWindow{}, errors.New("窗口已经不存在")
	}
	var rect windowsRect
	ok, _, callErr := windowsProcGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if ok == 0 {
		return hostRPAWindow{}, fmt.Errorf("读取窗口位置失败：%w", callErr)
	}
	var pid uint32
	windowsProcGetWindowPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	minimized, _, _ := windowsProcIsIconic.Call(hwnd)
	return hostRPAWindow{
		ID:        windowsWindowID(hwnd),
		PID:       int(pid),
		Title:     windowsWindowTitle(hwnd),
		X:         int(rect.Left),
		Y:         int(rect.Top),
		Width:     int(rect.Right - rect.Left),
		Height:    int(rect.Bottom - rect.Top),
		Active:    true,
		Minimized: minimized != 0,
	}, nil
}

func windowsWindowTitle(hwnd uintptr) string {
	length, _, _ := windowsProcGetWindowTextLenW.Call(hwnd)
	if length == 0 {
		return ""
	}
	buffer := make([]uint16, int(length)+1)
	windowsProcGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	return strings.TrimSpace(windows.UTF16ToString(buffer))
}

func windowsWindowID(hwnd uintptr) string {
	return fmt.Sprintf("win:%x", hwnd)
}

func parseWindowsWindowID(id string) (uintptr, error) {
	value := strings.TrimPrefix(strings.TrimSpace(id), "win:")
	if value == id || value == "" {
		return 0, errors.New("Windows 窗口 ID 无效")
	}
	hwnd, err := strconv.ParseUint(value, 16, 64)
	if err != nil || hwnd == 0 {
		return 0, errors.New("Windows 窗口 ID 无效")
	}
	return uintptr(hwnd), nil
}

func windowsMoveMouse(x, y, durationMS int) error {
	if durationMS <= 0 {
		return windowsSetCursor(x, y)
	}
	startX, startY, err := hostRPAMousePosition()
	if err != nil {
		return err
	}
	steps := durationMS / 16
	if steps < 1 {
		steps = 1
	}
	for step := 1; step <= steps; step++ {
		currentX := startX + (x-startX)*step/steps
		currentY := startY + (y-startY)*step/steps
		if err := windowsSetCursor(currentX, currentY); err != nil {
			return err
		}
		time.Sleep(time.Duration(durationMS/steps) * time.Millisecond)
	}
	return nil
}

func windowsSetCursor(x, y int) error {
	ok, _, callErr := windowsProcSetCursorPos.Call(uintptr(int32(x)), uintptr(int32(y)))
	if ok == 0 {
		return fmt.Errorf("移动鼠标失败：%w", callErr)
	}
	return nil
}

func windowsMouseButtonFlags(button string) (uint32, uint32) {
	switch button {
	case "right":
		return windowsMouseEventRightDown, windowsMouseEventRightUp
	case "middle":
		return windowsMouseEventMiddleDown, windowsMouseEventMiddleUp
	default:
		return windowsMouseEventLeftDown, windowsMouseEventLeftUp
	}
}

func windowsSendMouseButton(down, up uint32) error {
	inputs := []windowsInput{windowsMouseEvent(down, 0), windowsMouseEvent(up, 0)}
	return windowsSendInputs(inputs)
}

func windowsSendMouseFlag(flags, data uint32) error {
	return windowsSendInputs([]windowsInput{windowsMouseEvent(flags, data)})
}

func windowsMouseEvent(flags, data uint32) windowsInput {
	input := windowsInput{Type: windowsInputMouse}
	mouse := (*windowsMouseInput)(unsafe.Pointer(&input.Data[0]))
	mouse.Flags = flags
	mouse.MouseData = data
	return input
}

func windowsKeyboardEvent(key, scan uint16, flags uint32) windowsInput {
	input := windowsInput{Type: windowsInputKeyboard}
	keyboard := (*windowsKeyboardInput)(unsafe.Pointer(&input.Data[0]))
	keyboard.VirtualKey = key
	keyboard.ScanCode = scan
	keyboard.Flags = flags
	return input
}

func windowsSendInputs(inputs []windowsInput) error {
	if len(inputs) == 0 {
		return nil
	}
	sent, _, callErr := windowsProcSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(windowsInput{}),
	)
	if int(sent) != len(inputs) {
		return fmt.Errorf("发送宿主机输入失败：%w", callErr)
	}
	return nil
}

func windowsSendVirtualKeys(keys []uint16) error {
	inputs := make([]windowsInput, 0, len(keys)*2)
	for _, key := range keys {
		inputs = append(inputs, windowsKeyboardEvent(key, 0, windowsVirtualKeyFlags(key)))
	}
	for index := len(keys) - 1; index >= 0; index-- {
		key := keys[index]
		inputs = append(inputs, windowsKeyboardEvent(key, 0, windowsVirtualKeyFlags(key)|windowsKeyEventKeyUp))
	}
	return windowsSendInputs(inputs)
}

func windowsVirtualKeyFlags(key uint16) uint32 {
	if (key >= 0x21 && key <= 0x28) || key == 0x2E || key == 0x5B {
		return windowsKeyEventExtended
	}
	return 0
}

func windowsTypeText(text string, intervalMS int) error {
	for _, code := range utf16.Encode([]rune(text)) {
		inputs := []windowsInput{
			windowsKeyboardEvent(0, code, windowsKeyEventUnicode),
			windowsKeyboardEvent(0, code, windowsKeyEventUnicode|windowsKeyEventKeyUp),
		}
		if err := windowsSendInputs(inputs); err != nil {
			return err
		}
		if intervalMS > 0 {
			time.Sleep(time.Duration(intervalMS) * time.Millisecond)
		}
	}
	return nil
}

func windowsVirtualKey(name string) (uint16, error) {
	if len(name) == 1 {
		char := name[0]
		if char >= 'a' && char <= 'z' {
			return uint16(char - 'a' + 'A'), nil
		}
		if char >= '0' && char <= '9' {
			return uint16(char), nil
		}
	}
	keys := map[string]uint16{
		"backspace": 0x08,
		"tab":       0x09,
		"enter":     0x0D,
		"shift":     0x10,
		"ctrl":      0x11,
		"control":   0x11,
		"alt":       0x12,
		"option":    0x12,
		"escape":    0x1B,
		"esc":       0x1B,
		"space":     0x20,
		"pageup":    0x21,
		"pagedown":  0x22,
		"end":       0x23,
		"home":      0x24,
		"left":      0x25,
		"up":        0x26,
		"right":     0x27,
		"down":      0x28,
		"delete":    0x2E,
		"win":       0x5B,
		"cmd":       0x5B,
		"super":     0x5B,
		"f1":        0x70,
		"f2":        0x71,
		"f3":        0x72,
		"f4":        0x73,
		"f5":        0x74,
		"f6":        0x75,
		"f7":        0x76,
		"f8":        0x77,
		"f9":        0x78,
		"f10":       0x79,
		"f11":       0x7A,
		"f12":       0x7B,
	}
	key, ok := keys[name]
	if !ok {
		return 0, fmt.Errorf("不支持的 Windows 按键：%s", name)
	}
	return key, nil
}
