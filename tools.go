//go:build tools
// +build tools

// See https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
package tools

import (
	_ "golang.org/x/mobile/cmd/gobind"
	_ "golang.org/x/mobile/cmd/gomobile"
)
