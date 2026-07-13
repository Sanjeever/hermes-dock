//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -x objective-c -Wno-deprecated-declarations
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices

#import <Cocoa/Cocoa.h>
#import <ApplicationServices/ApplicationServices.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

static bool dock_rpa_accessibility_trusted(bool prompt) {
	const void *keys[] = { kAXTrustedCheckOptionPrompt };
	const void *values[] = { prompt ? kCFBooleanTrue : kCFBooleanFalse };
	CFDictionaryRef options = CFDictionaryCreate(
		kCFAllocatorDefault,
		keys,
		values,
		1,
		&kCFTypeDictionaryKeyCallBacks,
		&kCFTypeDictionaryValueCallBacks
	);
	bool trusted = AXIsProcessTrustedWithOptions(options);
	CFRelease(options);
	return trusted;
}

static char *dock_rpa_windows_json(void) {
	@autoreleasepool {
		CFArrayRef windowInfo = CGWindowListCopyWindowInfo(
			kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
			kCGNullWindowID
		);
		if (windowInfo == NULL) {
			return NULL;
		}
		pid_t activePID = [[[NSWorkspace sharedWorkspace] frontmostApplication] processIdentifier];
		bool activeAssigned = false;
		NSMutableArray *result = [NSMutableArray array];
		for (NSDictionary *window in (NSArray *)windowInfo) {
			NSNumber *layer = window[(id)kCGWindowLayer];
			if (layer == nil || [layer intValue] != 0) {
				continue;
			}
			NSNumber *windowNumber = window[(id)kCGWindowNumber];
			NSNumber *ownerPID = window[(id)kCGWindowOwnerPID];
			NSDictionary *bounds = window[(id)kCGWindowBounds];
			if (windowNumber == nil || ownerPID == nil || bounds == nil) {
				continue;
			}
			NSString *title = window[(id)kCGWindowName];
			if (title == nil || [title length] == 0) {
				title = window[(id)kCGWindowOwnerName];
			}
			int width = [bounds[@"Width"] intValue];
			int height = [bounds[@"Height"] intValue];
			if (width <= 0 || height <= 0) {
				continue;
			}
			bool active = !activeAssigned && [ownerPID intValue] == activePID;
			if (active) {
				activeAssigned = true;
			}
			[result addObject:@{
				@"id": [NSString stringWithFormat:@"mac:%u", [windowNumber unsignedIntValue]],
				@"pid": ownerPID,
				@"title": title ?: @"",
				@"x": bounds[@"X"],
				@"y": bounds[@"Y"],
				@"width": @(width),
				@"height": @(height),
				@"active": @(active),
				@"minimized": @NO
			}];
		}
		CFRelease(windowInfo);
		NSData *data = [NSJSONSerialization dataWithJSONObject:result options:0 error:nil];
		if (data == nil) {
			return NULL;
		}
		NSString *json = [[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding];
		char *output = strdup([json UTF8String]);
		[json release];
		return output;
	}
}

static int dock_rpa_activate_window(uint32_t targetWindow) {
	@autoreleasepool {
		CFArrayRef windowInfo = CGWindowListCopyWindowInfo(kCGWindowListOptionAll, kCGNullWindowID);
		if (windowInfo == NULL) {
			return 1;
		}
		pid_t targetPID = 0;
		for (NSDictionary *window in (NSArray *)windowInfo) {
			NSNumber *windowNumber = window[(id)kCGWindowNumber];
			if ([windowNumber unsignedIntValue] == targetWindow) {
				targetPID = [window[(id)kCGWindowOwnerPID] intValue];
				break;
			}
		}
		CFRelease(windowInfo);
		if (targetPID == 0) {
			return 2;
		}
		NSRunningApplication *application = [NSRunningApplication runningApplicationWithProcessIdentifier:targetPID];
		if (application == nil || ![application activateWithOptions:NSApplicationActivateIgnoringOtherApps]) {
			return 3;
		}

		AXUIElementRef appElement = AXUIElementCreateApplication(targetPID);
		CFTypeRef windows = NULL;
		if (AXUIElementCopyAttributeValue(appElement, kAXWindowsAttribute, &windows) == kAXErrorSuccess) {
			for (id value in (NSArray *)windows) {
				AXUIElementRef windowElement = (AXUIElementRef)value;
				CFTypeRef number = NULL;
				if (AXUIElementCopyAttributeValue(windowElement, CFSTR("AXWindowNumber"), &number) == kAXErrorSuccess) {
					if (CFGetTypeID(number) == CFNumberGetTypeID()) {
						int windowNumber = 0;
						CFNumberGetValue((CFNumberRef)number, kCFNumberIntType, &windowNumber);
						if ((uint32_t)windowNumber == targetWindow) {
							AXUIElementPerformAction(windowElement, kAXRaiseAction);
							CFRelease(number);
							break;
						}
					}
					CFRelease(number);
				}
			}
			CFRelease(windows);
		}
		CFRelease(appElement);
		usleep(100000);
		return 0;
	}
}

static bool dock_rpa_mouse_position(double *x, double *y) {
	CGEventRef event = CGEventCreate(NULL);
	if (event == NULL) {
		return false;
	}
	CGPoint point = CGEventGetLocation(event);
	CFRelease(event);
	*x = point.x;
	*y = point.y;
	return true;
}

static bool dock_rpa_mouse_event(int type, int button, double x, double y, int clickCount) {
	CGEventRef event = CGEventCreateMouseEvent(NULL, (CGEventType)type, CGPointMake(x, y), (CGMouseButton)button);
	if (event == NULL) {
		return false;
	}
	if (clickCount > 0) {
		CGEventSetIntegerValueField(event, kCGMouseEventClickState, clickCount);
	}
	CGEventPost(kCGHIDEventTap, event);
	CFRelease(event);
	return true;
}

static bool dock_rpa_scroll(int dx, int dy) {
	CGEventRef event = CGEventCreateScrollWheelEvent(NULL, kCGScrollEventUnitLine, 2, dy, dx);
	if (event == NULL) {
		return false;
	}
	CGEventPost(kCGHIDEventTap, event);
	CFRelease(event);
	return true;
}

static bool dock_rpa_key(uint16_t keyCode, uint64_t flags) {
	CGEventRef down = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)keyCode, true);
	CGEventRef up = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)keyCode, false);
	if (down == NULL || up == NULL) {
		if (down != NULL) CFRelease(down);
		if (up != NULL) CFRelease(up);
		return false;
	}
	CGEventSetFlags(down, (CGEventFlags)flags);
	CGEventSetFlags(up, (CGEventFlags)flags);
	CGEventPost(kCGHIDEventTap, down);
	CGEventPost(kCGHIDEventTap, up);
	CFRelease(down);
	CFRelease(up);
	return true;
}

static bool dock_rpa_type_text(const char *utf8, int intervalMS) {
	@autoreleasepool {
		NSString *text = [NSString stringWithUTF8String:utf8];
		if (text == nil) {
			return false;
		}
		NSUInteger length = [text length];
		for (NSUInteger offset = 0; offset < length;) {
			NSUInteger chunkLength = MIN((NSUInteger)20, length - offset);
			if (offset + chunkLength < length) {
				unichar last = [text characterAtIndex:offset + chunkLength - 1];
				if (CFStringIsSurrogateHighCharacter(last)) {
					chunkLength--;
				}
			}
			unichar buffer[20];
			[text getCharacters:buffer range:NSMakeRange(offset, chunkLength)];
			CGEventRef down = CGEventCreateKeyboardEvent(NULL, 0, true);
			CGEventRef up = CGEventCreateKeyboardEvent(NULL, 0, false);
			if (down == NULL || up == NULL) {
				if (down != NULL) CFRelease(down);
				if (up != NULL) CFRelease(up);
				return false;
			}
			CGEventKeyboardSetUnicodeString(down, chunkLength, buffer);
			CGEventKeyboardSetUnicodeString(up, chunkLength, buffer);
			CGEventPost(kCGHIDEventTap, down);
			CGEventPost(kCGHIDEventTap, up);
			CFRelease(down);
			CFRelease(up);
			offset += chunkLength;
			if (intervalMS > 0) {
				usleep((useconds_t)intervalMS * 1000);
			}
		}
		return true;
	}
}
*/
import "C"

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

const (
	macEventLeftDown     = 1
	macEventLeftUp       = 2
	macEventRightDown    = 3
	macEventRightUp      = 4
	macEventMouseMoved   = 5
	macEventLeftDragged  = 6
	macEventRightDragged = 7
	macEventOtherDown    = 25
	macEventOtherUp      = 26
	macEventOtherDragged = 27
	macMouseLeft         = 0
	macMouseRight        = 1
	macMouseMiddle       = 2
	macFlagShift         = 1 << 17
	macFlagControl       = 1 << 18
	macFlagOption        = 1 << 19
	macFlagCommand       = 1 << 20
)

func hostRPAPlatformCapabilities() hostRPACapabilities {
	return hostRPACapabilities{
		Supported:                       true,
		Backend:                         "macos-coregraphics",
		Windows:                         true,
		Mouse:                           true,
		Keyboard:                        true,
		UnicodeText:                     true,
		RequiresAccessibilityPermission: true,
	}
}

func hostRPAListWindows() ([]hostRPAWindow, error) {
	if err := requireMacRPAAccessibility(); err != nil {
		return nil, err
	}
	value := C.dock_rpa_windows_json()
	if value == nil {
		return nil, errors.New("读取 macOS 窗口失败")
	}
	defer C.free(unsafe.Pointer(value))
	var windows []hostRPAWindow
	if err := json.Unmarshal([]byte(C.GoString(value)), &windows); err != nil {
		return nil, fmt.Errorf("解析 macOS 窗口失败：%w", err)
	}
	return windows, nil
}

func hostRPAActiveWindow() (hostRPAWindow, error) {
	windows, err := hostRPAListWindows()
	if err != nil {
		return hostRPAWindow{}, err
	}
	for _, window := range windows {
		if window.Active {
			return window, nil
		}
	}
	return hostRPAWindow{}, errors.New("没有活动窗口")
}

func hostRPAActivateWindow(id string) error {
	if err := requireMacRPAAccessibility(); err != nil {
		return err
	}
	value := strings.TrimPrefix(strings.TrimSpace(id), "mac:")
	if value == id || value == "" {
		return errors.New("macOS 窗口 ID 无效")
	}
	windowID, err := strconv.ParseUint(value, 10, 32)
	if err != nil || windowID == 0 {
		return errors.New("macOS 窗口 ID 无效")
	}
	switch result := int(C.dock_rpa_activate_window(C.uint32_t(windowID))); result {
	case 0:
		active, activeErr := hostRPAActiveWindow()
		if activeErr != nil {
			return activeErr
		}
		if active.ID != id {
			return errors.New("目标应用已激活，但指定窗口未进入前台")
		}
		return nil
	case 2:
		return errors.New("窗口已经不存在")
	default:
		return errors.New("激活 macOS 窗口失败")
	}
}

func hostRPAMousePosition() (int, int, error) {
	var x, y C.double
	if !bool(C.dock_rpa_mouse_position(&x, &y)) {
		return 0, 0, errors.New("读取鼠标位置失败")
	}
	return int(x), int(y), nil
}

func hostRPAPerformMouse(req hostRPAMouseRequest) error {
	if err := requireMacRPAAccessibility(); err != nil {
		return err
	}
	switch req.Action {
	case "move":
		return macMoveMouse(req.X, req.Y, req.DurationMS, macEventMouseMoved, macMouseLeft)
	case "click":
		if err := macMoveMouse(req.X, req.Y, req.DurationMS, macEventMouseMoved, macMouseLeft); err != nil {
			return err
		}
		down, up, button, _ := macMouseButtonEvents(req.Button)
		for count := 1; count <= req.Count; count++ {
			if !bool(C.dock_rpa_mouse_event(down, button, C.double(req.X), C.double(req.Y), C.int(count))) ||
				!bool(C.dock_rpa_mouse_event(up, button, C.double(req.X), C.double(req.Y), C.int(count))) {
				return errors.New("发送鼠标点击失败")
			}
			if count < req.Count {
				time.Sleep(80 * time.Millisecond)
			}
		}
		return nil
	case "drag":
		down, up, button, dragged := macMouseButtonEvents(req.Button)
		if err := macMoveMouse(req.FromX, req.FromY, 0, macEventMouseMoved, int(button)); err != nil {
			return err
		}
		if !bool(C.dock_rpa_mouse_event(down, button, C.double(req.FromX), C.double(req.FromY), 1)) {
			return errors.New("按下鼠标失败")
		}
		moveErr := macMoveMouse(req.ToX, req.ToY, req.DurationMS, dragged, int(button))
		upOK := bool(C.dock_rpa_mouse_event(up, button, C.double(req.ToX), C.double(req.ToY), 1))
		var upErr error
		if !upOK {
			upErr = errors.New("释放鼠标失败")
		}
		return errors.Join(moveErr, upErr)
	case "scroll":
		if err := macMoveMouse(req.X, req.Y, 0, macEventMouseMoved, macMouseLeft); err != nil {
			return err
		}
		if !bool(C.dock_rpa_scroll(C.int(req.DX), C.int(req.DY))) {
			return errors.New("发送滚动事件失败")
		}
		return nil
	default:
		return errors.New("不支持的鼠标动作")
	}
}

func hostRPAPerformKeyboard(req hostRPAKeyboardRequest) error {
	if err := requireMacRPAAccessibility(); err != nil {
		return err
	}
	switch req.Action {
	case "press":
		key, modifier, err := macKey(req.Key)
		if err != nil {
			return err
		}
		if modifier != 0 {
			return errors.New("修饰键必须与普通按键组成快捷键")
		}
		for index := 0; index < req.Count; index++ {
			if !bool(C.dock_rpa_key(C.uint16_t(key), 0)) {
				return errors.New("发送键盘事件失败")
			}
			if req.IntervalMS > 0 && index+1 < req.Count {
				time.Sleep(time.Duration(req.IntervalMS) * time.Millisecond)
			}
		}
		return nil
	case "hotkey":
		var flags uint64
		var target uint16
		targetCount := 0
		for _, name := range req.Keys {
			key, modifier, err := macKey(name)
			if err != nil {
				return err
			}
			if modifier != 0 {
				flags |= modifier
			} else {
				target = key
				targetCount++
			}
		}
		if targetCount != 1 {
			return errors.New("macOS 快捷键必须包含一个普通按键")
		}
		if !bool(C.dock_rpa_key(C.uint16_t(target), C.uint64_t(flags))) {
			return errors.New("发送快捷键失败")
		}
		return nil
	case "type":
		text := C.CString(req.Text)
		defer C.free(unsafe.Pointer(text))
		if !bool(C.dock_rpa_type_text(text, C.int(req.IntervalMS))) {
			return errors.New("输入 Unicode 文本失败")
		}
		return nil
	default:
		return errors.New("不支持的键盘动作")
	}
}

func requireMacRPAAccessibility() error {
	if !bool(C.dock_rpa_accessibility_trusted(C.bool(true))) {
		return errors.New("未授予辅助功能权限，请在“系统设置 → 隐私与安全性 → 辅助功能”中允许 Hermes Dock")
	}
	return nil
}

func macMoveMouse(x, y, durationMS, eventType, button int) error {
	if durationMS <= 0 {
		if !bool(C.dock_rpa_mouse_event(C.int(eventType), C.int(button), C.double(x), C.double(y), 0)) {
			return errors.New("移动鼠标失败")
		}
		return nil
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
		if !bool(C.dock_rpa_mouse_event(C.int(eventType), C.int(button), C.double(currentX), C.double(currentY), 0)) {
			return errors.New("移动鼠标失败")
		}
		time.Sleep(time.Duration(durationMS/steps) * time.Millisecond)
	}
	return nil
}

func macMouseButtonEvents(button string) (C.int, C.int, C.int, int) {
	switch button {
	case "right":
		return macEventRightDown, macEventRightUp, macMouseRight, macEventRightDragged
	case "middle":
		return macEventOtherDown, macEventOtherUp, macMouseMiddle, macEventOtherDragged
	default:
		return macEventLeftDown, macEventLeftUp, macMouseLeft, macEventLeftDragged
	}
}

func macKey(name string) (uint16, uint64, error) {
	keys := map[string]uint16{
		"a": 0, "s": 1, "d": 2, "f": 3, "h": 4, "g": 5, "z": 6, "x": 7,
		"c": 8, "v": 9, "b": 11, "q": 12, "w": 13, "e": 14, "r": 15,
		"y": 16, "t": 17, "1": 18, "2": 19, "3": 20, "4": 21, "6": 22,
		"5": 23, "9": 25, "7": 26, "8": 28, "0": 29, "o": 31, "u": 32,
		"i": 34, "p": 35, "enter": 36, "l": 37, "j": 38, "k": 40, "n": 45,
		"m": 46, "tab": 48, "space": 49, "backspace": 51, "escape": 53, "esc": 53,
		"f5": 96, "f6": 97, "f7": 98, "f3": 99, "f8": 100, "f9": 101,
		"f11": 103, "f10": 109, "f12": 111, "home": 115, "pageup": 116,
		"delete": 117, "f4": 118, "end": 119, "f2": 120, "pagedown": 121,
		"f1": 122, "left": 123, "right": 124, "down": 125, "up": 126,
	}
	modifiers := map[string]uint64{
		"shift": macFlagShift,
		"ctrl":  macFlagControl, "control": macFlagControl,
		"alt": macFlagOption, "option": macFlagOption,
		"cmd": macFlagCommand, "command": macFlagCommand, "super": macFlagCommand,
	}
	if modifier, ok := modifiers[name]; ok {
		return 0, modifier, nil
	}
	key, ok := keys[name]
	if !ok {
		return 0, 0, fmt.Errorf("不支持的 macOS 按键：%s", name)
	}
	return key, 0, nil
}
