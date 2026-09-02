package xray

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xtls/libxray/nodep"
	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/session"
	"github.com/xtls/xray-core/core"
)

// ProbeXray dispatches one HTTP request through the draft's real DNS, routing
// and outbounds. It does not start listeners or startup-only integrations, and
// is not proof that extra inbounds or all destinations work. Like CheckRoute,
// the caller must isolate it from unmanaged instances in the same process.
func ProbeXray(xrayJSON, targetURL string, timeout int, inboundTag string) (int64, error) {
	uri, err := url.ParseRequestURI(targetURL)
	if err != nil || uri.Host == "" || uri.User != nil ||
		(uri.Scheme != "http" && uri.Scheme != "https") || timeout < 1 || timeout > 60 {
		return 0, errors.New("testXray requires an HTTP(S) URL and a timeout of 1–60 seconds")
	}
	coreServerMu.Lock()
	defer coreServerMu.Unlock()
	if coreServer != nil {
		return 0, errors.New("testXray requires an isolated process without a managed Xray instance")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	config, err := core.LoadConfig("json", strings.NewReader(xrayJSON))
	if err != nil {
		return 0, errors.New("testXray configuration could not be built")
	}
	if _, _, err = prepareRouteCheck(config); err != nil {
		return 0, err
	}
	server, err := core.NewWithContext(ctx, config)
	if err != nil {
		return 0, errors.New("testXray configuration could not be constructed")
	}
	defer server.Close()
	transport := &http.Transport{
		DisableKeepAlives: true,
		DialContext: func(call context.Context, network, address string) (net.Conn, error) {
			destination, err := xnet.ParseDestination("tcp:" + address)
			if err != nil {
				return nil, err
			}
			call = session.ContextWithInbound(call, &session.Inbound{Tag: inboundTag})
			return core.Dial(call, server, destination)
		},
	}
	defer transport.CloseIdleConnections()
	delay, err := nodep.PingHTTPRequest(&http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeout) * time.Second,
	}, targetURL, timeout)
	if err != nil {
		// HTTP errors may include a credential-bearing URL. Keep them local.
		return 0, errors.New("testXray URL request failed")
	}
	return delay, nil
}
