package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestDecodeParamValidatesInput(t *testing.T) {
	value, err := decodeParam[string]([]json.RawMessage{json.RawMessage(`"sales"`)}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if value != "sales" {
		t.Fatalf("decoded value = %q, want sales", value)
	}

	if _, err := decodeParam[string](nil, 0); err == nil || !strings.Contains(err.Error(), "缺少参数 1") {
		t.Fatalf("missing parameter error = %v", err)
	}
	if _, err := decodeParam[string]([]json.RawMessage{json.RawMessage(`{`)}, 0); err == nil {
		t.Fatal("invalid JSON should be rejected")
	}
}

func TestRPCArgumentAdaptersForwardDecodedValues(t *testing.T) {
	params := []json.RawMessage{
		json.RawMessage(`"sales"`),
		json.RawMessage(`2`),
		json.RawMessage(`true`),
		json.RawMessage(`"message"`),
	}

	var one string
	if _, err := oneArg(func(value string) error { one = value; return nil })(params); err != nil {
		t.Fatal(err)
	}
	if one != "sales" {
		t.Fatalf("oneArg value = %q", one)
	}

	var twoProfile string
	var twoCount int
	if _, err := twoArgs(func(profile string, count int) error {
		twoProfile = profile
		twoCount = count
		return nil
	})(params); err != nil {
		t.Fatal(err)
	}
	if twoProfile != "sales" || twoCount != 2 {
		t.Fatalf("twoArgs values = %q, %d", twoProfile, twoCount)
	}

	var three bool
	if _, err := threeArgs(func(profile string, count int, enabled bool) error {
		three = profile == "sales" && count == 2 && enabled
		return nil
	})(params); err != nil {
		t.Fatal(err)
	}
	if !three {
		t.Fatal("threeArgs did not forward decoded values")
	}

	var four string
	if _, err := fourArgs(func(profile string, count int, enabled bool, message string) error {
		if profile == "sales" && count == 2 && enabled {
			four = message
		}
		return nil
	})(params); err != nil {
		t.Fatal(err)
	}
	if four != "message" {
		t.Fatalf("fourArgs value = %q", four)
	}
}

func TestRPCValueAdaptersReturnResults(t *testing.T) {
	params := []json.RawMessage{json.RawMessage(`"sales"`), json.RawMessage(`2`)}

	result, err := noParams(func() (interface{}, error) { return "ready", nil })(nil)
	if err != nil || result != "ready" {
		t.Fatalf("noParams result = %#v, error = %v", result, err)
	}

	result, err = oneArgValue(func(profile string) (string, error) { return profile + "-ready", nil })(params)
	if err != nil || result != "sales-ready" {
		t.Fatalf("oneArgValue result = %#v, error = %v", result, err)
	}

	result, err = noResultValue(func(profile string) (int, error) { return len(profile), nil })(params)
	if err != nil || result != 5 {
		t.Fatalf("noResultValue result = %#v, error = %v", result, err)
	}

	result, err = twoArgsValue(func(profile string, count int) (string, error) {
		return profile + strings.Repeat("!", count), nil
	})(params)
	if err != nil || result != "sales!!" {
		t.Fatalf("twoArgsValue result = %#v, error = %v", result, err)
	}
}

func TestRPCAdaptersPropagateHandlerErrors(t *testing.T) {
	wantErr := errors.New("handler failed")
	if _, err := noResult(func() error { return wantErr })(nil); !errors.Is(err, wantErr) {
		t.Fatalf("noResult error = %v", err)
	}
	if _, err := oneArg(func(string) error { return wantErr })([]json.RawMessage{json.RawMessage(`"sales"`)}); !errors.Is(err, wantErr) {
		t.Fatalf("oneArg error = %v", err)
	}
}

func TestWebLockedRejectsConcurrentOperationAndReleasesAfterFailure(t *testing.T) {
	app := NewApp()
	app.web = newWebRuntime()
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	blockingHandler := app.webLocked(func() error {
		close(entered)
		<-release
		return nil
	})
	go func() {
		_, err := blockingHandler(nil)
		done <- err
	}()
	<-entered

	if _, err := blockingHandler(nil); err == nil || !strings.Contains(err.Error(), "已有操作正在执行") {
		t.Fatalf("concurrent operation error = %v", err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("first operation error = %v", err)
	}

	failingHandler := app.webLocked(func() error {
		return errors.New("operation failed")
	})
	if _, err := failingHandler(nil); err == nil || err.Error() != "operation failed" {
		t.Fatalf("handler error = %v", err)
	}
	if app.web.operationBusy {
		t.Fatal("operation lock was not released after failure")
	}
}

func TestWebLockedOneArgDecodesBeforeTakingOperationLock(t *testing.T) {
	app := NewApp()
	app.web = newWebRuntime()
	called := false
	handler := webLockedOneArg(app, func(value string) error {
		called = value == "sales"
		return nil
	})

	app.web.operationBusy = true
	if _, err := handler(nil); err == nil || !strings.Contains(err.Error(), "缺少参数 1") {
		t.Fatalf("decode error = %v", err)
	}
	if called || !app.web.operationBusy {
		t.Fatal("invalid argument changed operation state or called the handler")
	}
	app.web.operationBusy = false

	if _, err := handler([]json.RawMessage{json.RawMessage(`"sales"`)}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("decoded argument was not forwarded")
	}
}
