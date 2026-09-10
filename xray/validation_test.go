package xray

import (
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/xtls/xray-core/core"
)

func TestTestXrayRejectsConstructionErrors(t *testing.T) {
	for name, rule := range map[string]string{
		"missing balancer": `{"domain":["example.com"],"balancerTag":"missing"}`,
		"invalid regexp":   `{"domain":["regexp:["],"outboundTag":"direct"}`,
	} {
		t.Run(name, func(t *testing.T) {
			config := `{"log":{"loglevel":"none"},"outbounds":[{"protocol":"freedom","tag":"direct"}],"routing":{"rules":[` + rule + `]}}`
			if _, err := core.LoadConfig("json", strings.NewReader(config)); err != nil {
				t.Fatalf("fixture must pass the old build-only check: %v", err)
			}
			if err := TestXray(config); err == nil {
				t.Fatal("construction error was accepted")
			}
			if GetXrayState() {
				t.Fatal("validation published a managed instance")
			}
		})
	}
}

func TestTestXrayDoesNotBindPorts(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	port := listener.Addr().(*net.TCPAddr).Port
	config := fmt.Sprintf(`{
		"log":{"loglevel":"none"},
		"stats":{},"metrics":{"listen":"127.0.0.1:%d"},
		"inbounds":[{"listen":"127.0.0.1","port":%d,"protocol":"socks"}],
		"outbounds":[{"protocol":"freedom","tag":"direct"}]
	}`, port, port)
	for range 2 {
		if err := TestXray(config); err != nil {
			t.Fatalf("validation must not start listeners: %v", err)
		}
		if GetXrayState() {
			t.Fatal("validation left a managed instance")
		}
	}
}
