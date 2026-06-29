//go:build !linux

package main

func initIpRoute(_ string, _ int, _ bool) error {
	return nil
}
