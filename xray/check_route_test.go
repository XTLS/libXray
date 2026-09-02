package xray

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const routeCheckConfig = `{
  "log": {"loglevel":"none"},
  "dns": {"hosts":{"ip-rule.test":"192.0.2.2", "unknown.test":"198.51.100.2"}},
  "observatory": {"subjectSelector":[]},
  "outbounds": [
    {"tag":"default-loop","protocol":"loopback","settings":{"inboundTag":"default-vpn"}},
    {"tag":"direct","protocol":"freedom"},
    {"tag":"block","protocol":"blackhole"},
    {"tag":"entry-1","protocol":"freedom"}
  ],
  "routing": {
    "domainStrategy":"IPIfNonMatch",
    "balancers":[{"tag":"proxy","selector":["entry-1"],"strategy":{"type":"roundRobin"},"fallbackTag":"block"}],
    "rules":[
      {"ruleTag":"default-vpn","inboundTag":["default-vpn"],"balancerTag":"proxy"},
      {"ruleTag":"duplicate","domain":["full:domain-rule.test"],"port":"443","network":"tcp","outboundTag":"direct"},
      {"ruleTag":"duplicate","ip":["192.0.2.0/24"],"outboundTag":"block"},
      {"ruleTag":"selected-vpn","domain":["full:vpn.test"],"balancerTag":"proxy"},
      {"domain":["full:unnamed.test"],"outboundTag":"direct"}
    ]
  }
}`

func routeInput(config string) RouteCheckInput {
	return RouteCheckInput{XrayJSON: config, Domain: "unknown.test", Port: 443, Network: "tcp", InboundTag: "tunIn", Timeout: 5000}
}

func TestCheckRouteCoreEvidence(t *testing.T) {
	for _, sample := range []struct {
		name, domain, ip, network string
		port                      int
		want                      RouteCheckResult
	}{
		{"domain", "domain-rule.test", "", "tcp", 443, RouteCheckResult{Matched: true, RuleTag: "duplicate", OutboundTag: "direct"}},
		{"resolved IP", "ip-rule.test", "", "tcp", 443, RouteCheckResult{Matched: true, RuleTag: "duplicate", OutboundTag: "block"}},
		{"IP literal", "", "192.0.2.3", "udp", 53, RouteCheckResult{Matched: true, RuleTag: "duplicate", OutboundTag: "block"}},
		{"explicit balancer", "vpn.test", "", "tcp", 443, RouteCheckResult{Matched: true, RuleTag: "selected-vpn", OutboundTag: "entry-1", BalancerTag: "proxy"}},
		{"unnamed", "unnamed.test", "", "tcp", 443, RouteCheckResult{Matched: true, OutboundTag: "direct"}},
		{"default VPN", "unknown.test", "", "tcp", 443, RouteCheckResult{Defaulted: true, OutboundTag: "entry-1", BalancerTag: "proxy"}},
		{"AND network", "domain-rule.test", "", "udp", 443, RouteCheckResult{Defaulted: true, OutboundTag: "entry-1", BalancerTag: "proxy"}},
		{"AND port", "domain-rule.test", "", "tcp", 80, RouteCheckResult{Defaulted: true, OutboundTag: "entry-1", BalancerTag: "proxy"}},
	} {
		t.Run(sample.name, func(t *testing.T) {
			input := routeInput(routeCheckConfig)
			input.Domain, input.IP, input.Network, input.Port = sample.domain, sample.ip, sample.network, sample.port
			// Keep negative domain cases local, too: no external DNS in tests.
			input.XrayJSON = strings.Replace(input.XrayJSON, `"unknown.test":"198.51.100.2"`, `"unknown.test":"198.51.100.2","domain-rule.test":"198.51.100.3"`, 1)
			got, err := CheckRoute(input)
			if err != nil || got != sample.want {
				t.Fatalf("got %+v, %v; want %+v", got, err, sample.want)
			}
		})
	}

	t.Run("Raw default is not assumed to be proxy", func(t *testing.T) {
		got, err := CheckRoute(routeInput(minimalConfig))
		want := RouteCheckResult{Defaulted: true, OutboundTag: "direct"}
		if err != nil || got != want {
			t.Fatalf("got %+v, %v; want %+v", got, err, want)
		}
	})

	t.Run("ordinary VLESS is supported without connecting", func(t *testing.T) {
		input := routeInput(`{"outbounds":[{"tag":"entry","protocol":"vless","settings":{"address":"127.0.0.1","port":9,"id":"00000000-0000-0000-0000-000000000000","encryption":"none"}}]}`)
		got, err := CheckRoute(input)
		if err != nil || got != (RouteCheckResult{Defaulted: true, OutboundTag: "entry"}) {
			t.Fatalf("ordinary VLESS: %+v %v", got, err)
		}
	})
}

func TestCheckRouteDoesNotStartListenPublishOrDialTarget(t *testing.T) {
	var requests atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	defer target.Close()
	address := target.Listener.Addr().(*net.TCPAddr)
	logPath := filepath.Join(t.TempDir(), "must-not-exist.log")
	config := fmt.Sprintf(`{
      "log":{"access":%q,"error":%q,"loglevel":"debug"},
      "inbounds":[{"listen":"127.0.0.1","port":%d,"protocol":"socks"}],
      "outbounds":[{"tag":"direct","protocol":"freedom"}],
      "observatory":{"subjectSelector":["direct"],"probeUrl":%q,"probeInterval":"1ms"},
      "routing":{"rules":[{"ruleTag":"test","network":"tcp","outboundTag":"direct","webhook":{"url":%q}}]}
    }`, logPath, logPath, address.Port, target.URL, target.URL)
	input := routeInput(config)
	input.Domain, input.IP, input.Port = "", "127.0.0.1", address.Port
	got, err := CheckRoute(input)
	if err != nil || !got.Matched || got.OutboundTag != "direct" {
		t.Fatalf("route failed: %+v %v", got, err)
	}
	if requests.Load() != 0 {
		t.Fatal("route checking must not dispatch target, webhook, or probes")
	}
	if _, err := os.Stat(logPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("draft log was opened: %v", err)
	}
}

func TestCheckRouteRejectsManagedOverlapBeforeLoadingEnv(t *testing.T) {
	if err := RunXray(minimalConfig); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = StopXray() })
	const key = "XRAY_LIBXRAY_CHECK_ROUTE_TEST"
	t.Setenv(key, "original")
	input := routeInput(`{"env":{"` + key + `":"changed"},"outbounds":[{"protocol":"freedom"}]}`)
	if _, err := CheckRoute(input); err == nil || !strings.Contains(err.Error(), "isolated process") {
		t.Fatalf("expected managed-overlap error, got %v", err)
	}
	if os.Getenv(key) != "original" || !GetXrayState() {
		t.Fatal("route check modified managed runtime or process environment")
	}
}

func TestCheckRouteDNSDeadline(t *testing.T) {
	blackhole, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer blackhole.Close()
	address := blackhole.LocalAddr().(*net.UDPAddr)
	config := fmt.Sprintf(`{
      "dns":{"servers":[{"address":"127.0.0.1","port":%d}]},
      "outbounds":[{"tag":"direct","protocol":"freedom"}],
      "routing":{"domainStrategy":"IPIfNonMatch","rules":[{"ip":["192.0.2.0/24"],"outboundTag":"direct"}]}
    }`, address.Port)
	input := routeInput(config)
	input.Timeout = 100
	started := time.Now()
	if _, err := CheckRoute(input); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline, got %v", err)
	}
	if time.Since(started) > 3*time.Second {
		t.Fatal("core DNS did not honor the operation context")
	}
	// The timed-out operation is fully closed before the managed instance starts.
	if err := RunXray(minimalConfig); err != nil {
		t.Fatal(err)
	}
	if err := StopXray(); err != nil {
		t.Fatal(err)
	}
}

func TestCheckRouteRejectsInvalidInputsAndUnresolvedPaths(t *testing.T) {
	for _, sample := range []struct {
		name string
		edit func(*RouteCheckInput)
	}{
		{"empty config", func(i *RouteCheckInput) { i.XrayJSON = "" }},
		{"path is not JSON", func(i *RouteCheckInput) { i.XrayJSON = "/xray.json" }},
		{"malformed JSON", func(i *RouteCheckInput) { i.XrayJSON = "{" }},
		{"missing target", func(i *RouteCheckInput) { i.Domain = "" }},
		{"two targets", func(i *RouteCheckInput) { i.IP = "192.0.2.1" }},
		{"URL is not domain", func(i *RouteCheckInput) { i.Domain = "https://example.com" }},
		{"empty label", func(i *RouteCheckInput) { i.Domain = "example..com" }},
		{"IP in domain", func(i *RouteCheckInput) { i.Domain = "192.0.2.1" }},
		{"invalid IP", func(i *RouteCheckInput) { i.Domain, i.IP = "", "invalid" }},
		{"scoped IP", func(i *RouteCheckInput) { i.Domain, i.IP = "", "fe80::1%en0" }},
		{"port zero", func(i *RouteCheckInput) { i.Port = 0 }},
		{"port overflow", func(i *RouteCheckInput) { i.Port = 65536 }},
		{"unsupported network", func(i *RouteCheckInput) { i.Network = "icmp" }},
		{"missing timeout", func(i *RouteCheckInput) { i.Timeout = 0 }},
		{"timeout overflow", func(i *RouteCheckInput) { i.Timeout = 60001 }},
		{"no default", func(i *RouteCheckInput) { i.XrayJSON = "{}" }},
		{"loop cycle", func(i *RouteCheckInput) {
			i.XrayJSON = `{"outbounds":[{"protocol":"loopback","settings":{"inboundTag":"repeat"}}]}`
		}},
		{"loop sniffing", func(i *RouteCheckInput) {
			i.XrayJSON = `{"outbounds":[{"protocol":"loopback","settings":{"inboundTag":"repeat","sniffing":{"enabled":true,"destOverride":["tls"]}}}]}`
		}},
		{"missing selected handler", func(i *RouteCheckInput) {
			i.XrayJSON = `{"outbounds":[{"protocol":"freedom"}],"routing":{"rules":[{"network":"tcp","outboundTag":"missing"}]}}`
		}},
	} {
		t.Run(sample.name, func(t *testing.T) {
			input := routeInput(minimalConfig)
			sample.edit(&input)
			if _, err := CheckRoute(input); err == nil {
				t.Fatal("invalid input/path accepted")
			}
		})
	}
}

func TestCheckRouteRejectsConstructionSideEffects(t *testing.T) {
	for _, sample := range []struct{ config, message string }{
		{`{"outbounds":[{"protocol":"wireguard","settings":{"secretKey":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","address":["10.0.0.2/32"],"peers":[{"publicKey":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","endpoint":"127.0.0.1:9"}]}}]}`, "without creating a TUN device"},
		{`{"outbounds":[{"protocol":"vless","settings":{"address":"127.0.0.1","port":9,"id":"00000000-0000-0000-0000-000000000000","encryption":"none","reverse":{"tag":"reverse"}}}]}`, "without starting background connections"},
	} {
		if _, err := CheckRoute(routeInput(sample.config)); err == nil || !strings.Contains(err.Error(), sample.message) {
			t.Fatalf("expected explicit construction guard %q, got %v", sample.message, err)
		}
	}
}
