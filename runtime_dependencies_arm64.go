//go:build arm64

package main

import "embed"

//go:embed runtime-deps/linux-arm64
var runtimeDependencyFS embed.FS

const runtimeDependencySourceRoot = "runtime-deps/linux-arm64"
