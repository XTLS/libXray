//go:build !linux && !windows

package main

func initIpRoute(_ string, _ int) error {
	return nil
}
