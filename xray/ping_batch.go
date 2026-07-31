package xray

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/xtls/libxray/nodep"
	xrayNet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/session"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf"
	confJSON "github.com/xtls/xray-core/infra/conf/json"
)

const (
	maxPingBatchConfigs = 5
)

type PingBatchItem struct {
	XrayJSON    string
	OutboundTag string
}

type PingBatchResult struct {
	Success bool
	Delay   int64
	Error   string
}

type pingOutboundConfig struct {
	Outbounds []conf.OutboundDetourConfig `json:"outbounds"`
}

type preparedPingItem struct {
	resultIndex int
	outboundTag string
}

func PingBatch(
	items []PingBatchItem,
	timeout int,
	targetURL string,
) ([]PingBatchResult, error) {
	if err := validatePingBatchRequest(items, timeout, targetURL); err != nil {
		return nil, err
	}

	results := make([]PingBatchResult, len(items))
	prepared := make([]preparedPingItem, 0, len(items))
	mergedOutbounds := make([]conf.OutboundDetourConfig, 0, len(items))

	for index, item := range items {
		outbounds, err := readPingOutbounds(item.XrayJSON)
		if err != nil {
			results[index] = failedPingBatchResult(nodep.PingDelayError, err)
			continue
		}

		namespaced, outboundTag, err := preparePingOutbounds(
			outbounds,
			item.OutboundTag,
			index,
		)
		if err != nil {
			results[index] = failedPingBatchResult(nodep.PingDelayError, err)
			continue
		}
		prepared = append(prepared, preparedPingItem{
			resultIndex: index,
			outboundTag: outboundTag,
		})
		mergedOutbounds = append(mergedOutbounds, namespaced...)
	}

	if len(prepared) == 0 {
		return results, nil
	}

	server, err := startPingBatchServer(mergedOutbounds)
	if err != nil {
		return nil, err
	}
	defer server.Close()

	workerCount := len(prepared)
	jobs := make(chan preparedPingItem)
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for item := range jobs {
				delay, err := measureOutboundDelay(
					server,
					item.outboundTag,
					timeout,
					targetURL,
				)
				if err != nil {
					results[item.resultIndex] = failedPingBatchResult(
						delay,
						err,
					)
					continue
				}
				results[item.resultIndex] = PingBatchResult{
					Success: true,
					Delay:   delay,
				}
			}
		}()
	}

	for _, item := range prepared {
		jobs <- item
	}
	close(jobs)
	workers.Wait()

	return results, nil
}

func validatePingBatchRequest(
	items []PingBatchItem,
	timeout int,
	targetURL string,
) error {
	if len(items) == 0 {
		return errors.New("ping batch configs are empty")
	}
	if len(items) > maxPingBatchConfigs {
		return fmt.Errorf(
			"ping batch contains more than %d configs",
			maxPingBatchConfigs,
		)
	}
	if timeout <= 0 {
		return errors.New("ping batch timeout must be greater than zero")
	}

	parsedURL, err := url.ParseRequestURI(targetURL)
	if err != nil || parsedURL.Host == "" ||
		(parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return errors.New("ping batch URL must be an absolute HTTP or HTTPS URL")
	}
	return nil
}

func readPingOutbounds(xrayJSON string) ([]conf.OutboundDetourConfig, error) {
	if xrayJSON == "" {
		return nil, errors.New("ping Xray JSON is empty")
	}

	var config pingOutboundConfig
	decoder := json.NewDecoder(&confJSON.Reader{Reader: strings.NewReader(xrayJSON)})
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, err
	}
	if len(config.Outbounds) == 0 {
		return nil, errors.New("ping config has no outbounds")
	}
	return config.Outbounds, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return errors.New("ping config contains multiple JSON values")
	} else if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func preparePingOutbounds(
	outbounds []conf.OutboundDetourConfig,
	requestedTag string,
	itemIndex int,
) ([]conf.OutboundDetourConfig, string, error) {
	tagIndexes := make(map[string]int, len(outbounds))
	duplicateTags := make(map[string]struct{})
	for index, outbound := range outbounds {
		if outbound.Tag == "" {
			continue
		}
		if _, found := tagIndexes[outbound.Tag]; found {
			duplicateTags[outbound.Tag] = struct{}{}
			continue
		}
		tagIndexes[outbound.Tag] = index
	}

	targetIndex := 0
	if requestedTag != "" {
		if _, duplicate := duplicateTags[requestedTag]; duplicate {
			return nil, "", fmt.Errorf("duplicate outbound tag %q", requestedTag)
		}
		var found bool
		targetIndex, found = tagIndexes[requestedTag]
		if !found {
			return nil, "", fmt.Errorf("outbound tag %q not found", requestedTag)
		}
	} else if proxyIndex, found := tagIndexes["proxy"]; found {
		if _, duplicate := duplicateTags["proxy"]; duplicate {
			return nil, "", errors.New(`duplicate outbound tag "proxy"`)
		}
		targetIndex = proxyIndex
	}

	selectedIndexes, err := collectPingOutboundDependencies(
		outbounds,
		tagIndexes,
		duplicateTags,
		targetIndex,
	)
	if err != nil {
		return nil, "", err
	}

	namespacedTags := make(map[int]string, len(selectedIndexes))
	for order, index := range selectedIndexes {
		namespacedTags[index] = fmt.Sprintf(
			"__libxray_ping_%d_%d",
			itemIndex,
			order,
		)
	}

	prepared := make([]conf.OutboundDetourConfig, 0, len(selectedIndexes))
	for _, index := range selectedIndexes {
		outbound := outbounds[index]
		outbound.Tag = namespacedTags[index]

		if outbound.StreamSetting != nil &&
			outbound.StreamSetting.SocketSettings != nil {
			dialerProxy := outbound.StreamSetting.SocketSettings.DialerProxy
			if dialerProxy != "" {
				dependencyIndex := tagIndexes[dialerProxy]
				outbound.StreamSetting.SocketSettings.DialerProxy =
					namespacedTags[dependencyIndex]
			}
		}
		if outbound.ProxySettings != nil && outbound.ProxySettings.Tag != "" {
			dependencyIndex := tagIndexes[outbound.ProxySettings.Tag]
			outbound.ProxySettings.Tag = namespacedTags[dependencyIndex]
		}

		if _, err := outbound.Build(); err != nil {
			return nil, "", fmt.Errorf(
				"failed to build outbound %q: %w",
				outbounds[index].Tag,
				err,
			)
		}
		prepared = append(prepared, outbound)
	}
	return prepared, namespacedTags[targetIndex], nil
}

func collectPingOutboundDependencies(
	outbounds []conf.OutboundDetourConfig,
	tagIndexes map[string]int,
	duplicateTags map[string]struct{},
	targetIndex int,
) ([]int, error) {
	const (
		unvisited = iota
		visiting
		visited
	)
	states := make([]int, len(outbounds))
	selected := make([]int, 0, len(outbounds))

	var visit func(int) error
	visit = func(index int) error {
		switch states[index] {
		case visiting:
			return fmt.Errorf(
				"outbound dependency cycle detected at tag %q",
				outbounds[index].Tag,
			)
		case visited:
			return nil
		}
		states[index] = visiting
		selected = append(selected, index)

		for _, dependencyTag := range pingOutboundDependencyTags(outbounds[index]) {
			if _, duplicate := duplicateTags[dependencyTag]; duplicate {
				return fmt.Errorf(
					"outbound %q depends on duplicate outbound tag %q",
					outbounds[index].Tag,
					dependencyTag,
				)
			}
			dependencyIndex, found := tagIndexes[dependencyTag]
			if !found {
				return fmt.Errorf(
					"outbound %q depends on missing outbound %q",
					outbounds[index].Tag,
					dependencyTag,
				)
			}
			if err := visit(dependencyIndex); err != nil {
				return err
			}
		}
		states[index] = visited
		return nil
	}

	if err := visit(targetIndex); err != nil {
		return nil, err
	}
	return selected, nil
}

func pingOutboundDependencyTags(outbound conf.OutboundDetourConfig) []string {
	dependencies := make([]string, 0, 2)
	if outbound.StreamSetting != nil &&
		outbound.StreamSetting.SocketSettings != nil &&
		outbound.StreamSetting.SocketSettings.DialerProxy != "" {
		dependencies = append(
			dependencies,
			outbound.StreamSetting.SocketSettings.DialerProxy,
		)
	}
	if outbound.ProxySettings != nil && outbound.ProxySettings.Tag != "" {
		dependencies = append(dependencies, outbound.ProxySettings.Tag)
	}
	return dependencies
}

func startPingBatchServer(
	outbounds []conf.OutboundDetourConfig,
) (*core.Instance, error) {
	config, err := (&conf.Config{OutboundConfigs: outbounds}).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build ping batch config: %w", err)
	}
	server, err := core.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create ping batch instance: %w", err)
	}
	if err := server.Start(); err != nil {
		_ = server.Close()
		return nil, fmt.Errorf("failed to start ping batch instance: %w", err)
	}
	return server, nil
}

func measureOutboundDelay(
	server *core.Instance,
	outboundTag string,
	timeout int,
	targetURL string,
) (int64, error) {
	httpTimeout := time.Second * time.Duration(timeout)
	transport := &http.Transport{
		DisableKeepAlives: true,
		DialContext: func(
			ctx context.Context,
			network string,
			address string,
		) (net.Conn, error) {
			if network != "tcp" && network != "tcp4" && network != "tcp6" {
				return nil, fmt.Errorf("unsupported ping network %q", network)
			}
			destination, err := xrayNet.ParseDestination("tcp:" + address)
			if err != nil {
				return nil, err
			}
			ctx = session.SetForcedOutboundTagToContext(ctx, outboundTag)
			return core.Dial(ctx, server, destination)
		},
	}
	defer transport.CloseIdleConnections()

	client := &http.Client{
		Transport: transport,
		Timeout:   httpTimeout,
	}
	return nodep.PingHTTPRequest(client, targetURL, timeout)
}

func failedPingBatchResult(delay int64, err error) PingBatchResult {
	if delay != nodep.PingDelayTimeout {
		delay = nodep.PingDelayError
	}
	return PingBatchResult{
		Success: false,
		Delay:   delay,
		Error:   err.Error(),
	}
}
