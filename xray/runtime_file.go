//go:build !windows

package xray

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func lockRuntimeFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
}

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
