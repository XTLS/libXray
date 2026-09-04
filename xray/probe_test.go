package xray

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestProbeXrayUsesDraftDNSAndRoutingWithoutListening(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Error("probe must use HEAD")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	config := fmt.Sprintf(`{
		"inbounds":[{"listen":"127.0.0.1","port":%s,"protocol":"socks"}],
		"dns":{"hosts":{"probe.test":"127.0.0.1"}},
		"outbounds":[{"tag":"blocked","protocol":"blackhole"},{"tag":"ok","protocol":"freedom","settings":{"domainStrategy":"UseIP"}}],
		"routing":{"rules":[{"inboundTag":["tunIn"],"domain":["full:probe.test"],"outboundTag":"ok"}]}
	}`, u.Port())
	// The configured listener port is already occupied. Only the routed request
	// should run; accidentally starting the raw listeners would fail this test.
	target := "http://probe.test:" + u.Port() + "/"
	if delay, err := ProbeXray(config, target, 2, "tunIn"); err != nil || delay < 0 {
		t.Fatalf("routed probe: delay=%d err=%v", delay, err)
	}
	if _, err := ProbeXray(config, target, 1, "other"); err == nil || !strings.HasPrefix(err.Error(), "configuration probe ") {
		t.Fatalf("ignoring the draft routing: %v", err)
	}
	if GetXrayState() {
		t.Fatal("probe published a managed instance")
	}
}

func TestProbeXrayRejectsUnsafeRequestWithoutLeakingURL(t *testing.T) {
	for _, target := range []string{"file:///secret", "https://user:secret@example.com/"} {
		_, err := ProbeXray(`{}`, target, 1, "")
		if err == nil || !strings.HasPrefix(err.Error(), "configuration probe ") || strings.Contains(err.Error(), "secret") {
			t.Fatalf("unsafe error: %v", err)
		}
	}
}
