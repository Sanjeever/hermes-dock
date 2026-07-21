//go:build amd64

package main

import "embed"

//go:embed runtime-deps/linux-amd64
var runtimeDependencyFS embed.FS

const runtimeDependencySourceRoot = "runtime-deps/linux-amd64"
