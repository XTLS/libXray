//go:build !windows

package xray

import (
	"os"
	"path/filepath"
)

func replaceRuntimeState(source, target string) error {
	directory, err := os.Open(filepath.Dir(target))
	if err != nil {
		return err
	}
	defer directory.Close()
	if err := os.Rename(source, target); err != nil {
		return err
	}
	return directory.Sync()
}
