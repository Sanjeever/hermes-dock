package main

import (
	"encoding/json"
	"fmt"
)

func decodeParam[T any](params []json.RawMessage, index int) (T, error) {
	var zero T
	if index >= len(params) {
		return zero, fmt.Errorf("缺少参数 %d", index+1)
	}
	if err := json.Unmarshal(params[index], &zero); err != nil {
		return zero, err
	}
	return zero, nil
}

func noResult(fn func() error) webRPCHandler {
	return func(params []json.RawMessage) (interface{}, error) { return nil, fn() }
}

func (a *App) webLocked(fn func() error) webRPCHandler {
	return func(params []json.RawMessage) (interface{}, error) {
		if a.web == nil {
			return nil, fn()
		}
		a.web.mu.Lock()
		if a.web.operationBusy {
			a.web.mu.Unlock()
			return nil, fmt.Errorf("已有操作正在执行，请稍后再试")
		}
		a.web.operationBusy = true
		a.web.mu.Unlock()
		defer func() {
			a.web.mu.Lock()
			a.web.operationBusy = false
			a.web.mu.Unlock()
		}()
		return nil, fn()
	}
}

func noParams(fn func() (interface{}, error)) webRPCHandler {
	return func(params []json.RawMessage) (interface{}, error) { return fn() }
}

func oneArg[T any](fn func(T) error) webRPCHandler {
	return func(params []json.RawMessage) (interface{}, error) {
		arg, err := decodeParam[T](params, 0)
		if err != nil {
			return nil, err
		}
		return nil, fn(arg)
	}
}

func oneArgValue[T any, R any](fn func(T) (R, error)) webRPCHandler {
	return func(params []json.RawMessage) (interface{}, error) {
		arg, err := decodeParam[T](params, 0)
		if err != nil {
			return nil, err
		}
		return fn(arg)
	}
}

func noResultValue[T any, R any](fn func(T) (R, error)) webRPCHandler { return oneArgValue(fn) }

func twoArgs[A any, B any](fn func(A, B) error) webRPCHandler {
	return func(params []json.RawMessage) (interface{}, error) {
		a1, err := decodeParam[A](params, 0)
		if err != nil {
			return nil, err
		}
		a2, err := decodeParam[B](params, 1)
		if err != nil {
			return nil, err
		}
		return nil, fn(a1, a2)
	}
}

func threeArgs[A any, B any, C any](fn func(A, B, C) error) webRPCHandler {
	return func(params []json.RawMessage) (interface{}, error) {
		a1, err := decodeParam[A](params, 0)
		if err != nil {
			return nil, err
		}
		a2, err := decodeParam[B](params, 1)
		if err != nil {
			return nil, err
		}
		a3, err := decodeParam[C](params, 2)
		if err != nil {
			return nil, err
		}
		return nil, fn(a1, a2, a3)
	}
}
