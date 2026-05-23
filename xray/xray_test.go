package xray

import (
	"strings"
	"testing"
)

func TestBuildMphCacheUnsupportedByXrayCoreVersion(t *testing.T) {
	err := BuildMphCache("/tmp", "/tmp/mph.cache", "/tmp/config.json")
	if err == nil {
		t.Fatal("expected BuildMphCache to return an unsupported error")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("expected unsupported error, got %v", err)
	}
}
