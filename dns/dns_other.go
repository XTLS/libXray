//go:build !android && !linux && !windows

package dns

func InitDns(_ string, _ string) {}
