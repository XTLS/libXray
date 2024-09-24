package xray

import (
	"io"
	"net/http"
)

// query inbound and outbound stats.
// server means The metrics server address, like "http://[::1]:49227/debug/vars".
func QueryStats(server string) (string, error) {
	resp, err := http.Get(server)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
