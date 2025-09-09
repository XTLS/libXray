//go:build !linux

package main

func initIpRoute(_ string, _ int) error {
	return nil
}
