//go:build !linux

package main

func initIpRoute(_ string, _ string) error {
	return nil
}
