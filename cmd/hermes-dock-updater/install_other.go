//go:build !windows && !linux && !darwin

package main

import "errors"

func installPackage(config updateConfig) (func() error, func(), error) {
	return nil, nil, errors.New("当前系统不支持自动更新")
}
