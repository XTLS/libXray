package xray

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	xlog "github.com/xtls/xray-core/app/log"
	"github.com/xtls/xray-core/app/router"
	"github.com/xtls/xray-core/common"
	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/common/session"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features/outbound"
	"github.com/xtls/xray-core/features/routing"
	rsession "github.com/xtls/xray-core/features/routing/session"
	"github.com/xtls/xray-core/proxy/loopback"
	"github.com/xtls/xray-core/proxy/vless"
	vlessoutbound "github.com/xtls/xray-core/proxy/vless/outbound"
	"github.com/xtls/xray-core/proxy/wireguard"
	"golang.org/x/net/idna"
)

type RouteCheckInput struct {
	XrayJSON   string
	Domain     string
	IP         string
	Port       int
	Network    string
	InboundTag string
	Timeout    int // milliseconds
}

type RouteCheckResult struct {
	Matched     bool
	RuleTag     string
	OutboundTag string
	BalancerTag string
	Defaulted   bool
}

type routeRuleEvidence struct {
	ruleTag     string
	balancerTag string
}

// CheckRoute checks a draft with the real Router without starting the instance
// or dispatching the target. DNS lookup can still use the draft's outbounds.
// Like TestXray, construction changes core process globals: callers must isolate
// this operation from other non-managed instances. Managed overlap is rejected.
func CheckRoute(input RouteCheckInput) (result RouteCheckResult, err error) {
	target, err := routeCheckTarget(input)
	if err != nil {
		return result, err
	}
	coreServerMu.Lock()
	defer coreServerMu.Unlock()
	if coreServer != nil {
		return result, errors.New("checkRoute requires an isolated process without a managed Xray instance")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(input.Timeout)*time.Millisecond)
	defer cancel()
	config, err := core.LoadConfig("json", strings.NewReader(input.XrayJSON))
	if err != nil {
		return result, err
	}
	rules, loops, err := prepareRouteCheck(config)
	if err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	server, err := core.NewWithContext(ctx, config)
	if err != nil {
		return result, err
	}
	defer func() {
		// Never close an instance while PickRoute/DNS is still using it.
		cancel()
		err = errors.Join(err, server.Close())
	}()
	result, err = checkRoute(ctx, server, rules, loops, &rsession.Context{
		Inbound:  &session.Inbound{Tag: input.InboundTag},
		Outbound: &session.Outbound{Target: target},
	})
	if deadlineErr := ctx.Err(); deadlineErr != nil {
		return RouteCheckResult{}, deadlineErr
	}
	return result, err
}

func routeCheckTarget(input RouteCheckInput) (xnet.Destination, error) {
	invalid := xnet.Destination{}
	if len(input.XrayJSON) == 0 || len(input.XrayJSON) > 16*1024*1024 {
		return invalid, errors.New("checkRoute xrayJson must be nonempty and no larger than 16 MiB")
	}
	if (input.Domain == "") == (input.IP == "") {
		return invalid, errors.New("checkRoute requires exactly one of domain or ip")
	}
	if input.Port < 1 || input.Port > 65535 {
		return invalid, errors.New("checkRoute port must be between 1 and 65535")
	}
	if input.Timeout < 1 || input.Timeout > 60000 {
		return invalid, errors.New("checkRoute timeout must be between 1 and 60000 milliseconds")
	}
	var network xnet.Network
	switch input.Network {
	case "tcp":
		network = xnet.Network_TCP
	case "udp":
		network = xnet.Network_UDP
	default:
		return invalid, errors.New("checkRoute network must be tcp or udp")
	}
	var address xnet.Address
	if input.IP != "" {
		ip, err := netip.ParseAddr(input.IP)
		if err != nil || ip.Zone() != "" {
			return invalid, errors.New("checkRoute ip must be an IPv4 or IPv6 address without a zone")
		}
		address = xnet.IPAddress(ip.AsSlice())
	} else {
		domain, err := idna.Lookup.ToASCII(input.Domain)
		domain = strings.TrimSuffix(domain, ".")
		if err != nil || len(domain) == 0 || len(domain) > 253 {
			return invalid, errors.New("checkRoute domain must be a valid hostname")
		}
		if _, err := netip.ParseAddr(domain); err == nil {
			return invalid, errors.New("checkRoute IP literals must use the ip field")
		}
		for _, label := range strings.Split(domain, ".") {
			if len(label) == 0 || len(label) > 63 {
				return invalid, errors.New("checkRoute domain contains an invalid label")
			}
		}
		address = xnet.DomainAddress(domain)
	}
	return xnet.Destination{Network: network, Address: address, Port: xnet.Port(input.Port)}, nil
}

func prepareRouteCheck(config *core.Config) (map[string]routeRuleEvidence, map[string]*loopback.Config, error) {
	// A TUN inbound can allocate its device during construction, before Start.
	config.Inbound = nil
	rules := make(map[string]routeRuleEvidence)
	for index, app := range config.App {
		settings, err := app.GetInstance()
		if err != nil {
			return nil, nil, err
		}
		switch settings := settings.(type) {
		case *xlog.Config:
			// Logger construction opens files; draft checking must not write them.
			config.App[index] = serial.ToTypedMessage(&xlog.Config{})
		case *router.Config:
			for index, rule := range settings.Rule {
				tag := fmt.Sprintf("__libxray_check_rule_%d", index)
				rules[tag] = routeRuleEvidence{rule.GetRuleTag(), rule.GetBalancingTag()}
				rule.RuleTag = tag
				// PickRoute fires webhooks even without dispatcher publication.
				rule.Webhook = nil
			}
			config.App[index] = serial.ToTypedMessage(settings)
		}
	}
	loops := make(map[string]*loopback.Config)
	for index, handler := range config.Outbound {
		settings, err := handler.ProxySettings.GetInstance()
		if err != nil {
			return nil, nil, err
		}
		switch settings := settings.(type) {
		case *wireguard.DeviceConfig:
			return nil, nil, errors.New("checkRoute cannot construct WireGuard outbounds without creating a TUN device")
		case *vlessoutbound.Config:
			account, err := settings.GetVnext().GetUser().GetAccount().GetInstance()
			if err != nil {
				return nil, nil, err
			}
			if account.(*vless.Account).Reverse != nil {
				return nil, nil, errors.New("checkRoute cannot construct VLESS reverse outbounds without starting background connections")
			}
		case *loopback.Config:
			// Only the first untagged handler can be the default; later ones
			// cannot be addressed by a routing rule.
			if handler.Tag != "" || index == 0 {
				loops[handler.Tag] = settings
			}
		}
	}
	return rules, loops, nil
}

func checkRoute(ctx context.Context, server *core.Instance, rules map[string]routeRuleEvidence, loops map[string]*loopback.Config, input *rsession.Context) (RouteCheckResult, error) {
	var result RouteCheckResult
	router := server.GetFeature(routing.RouterType()).(routing.Router)
	manager := server.GetFeature(outbound.ManagerType()).(outbound.Manager)
	visited := make(map[string]bool)
	for hop := 0; ; hop++ {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		picked, err := router.PickRoute(input)
		if err == nil {
			evidence := rules[picked.GetRuleTag()]
			if hop == 0 {
				result.Matched = true
				result.RuleTag = evidence.ruleTag
			}
			result.OutboundTag = picked.GetOutboundTag()
			if evidence.balancerTag != "" {
				result.BalancerTag = evidence.balancerTag
			}
			if manager.GetHandler(result.OutboundTag) == nil {
				return result, errors.New("checkRoute matched an outbound that does not exist")
			}
		} else if errors.Is(err, common.ErrNoClue) {
			if hop == 0 {
				result.Defaulted = true
			}
			fallback := manager.GetDefaultHandler()
			if fallback == nil {
				return result, errors.New("checkRoute has no default outbound")
			}
			result.OutboundTag = fallback.Tag()
		} else {
			return result, err
		}
		loop := loops[result.OutboundTag]
		if loop == nil {
			return result, nil
		}
		if loop.Sniffing.GetEnabled() {
			return result, errors.New("checkRoute cannot determine a loopback path that requires traffic sniffing")
		}
		if visited[result.OutboundTag] {
			return result, errors.New("checkRoute encountered a loopback routing cycle")
		}
		visited[result.OutboundTag] = true
		// Follow only the real loopback metadata transition, never DispatchLink.
		input.Inbound = &session.Inbound{Tag: loop.InboundTag}
		input.Content = &session.Content{SkipDNSResolve: true}
	}
}
