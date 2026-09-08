package share

import (
	"encoding/json"

	"github.com/xtls/xray-core/infra/conf"
)

func convertShareLinksForTest(links string) (*conf.Config, error) {
	config, err := parseShareCandidatesForTest(links)
	if err != nil {
		return nil, err
	}
	return filterBuildableOutbounds(config)
}

func parseShareCandidatesForTest(links string) (*conf.Config, error) {
	return parseShareCandidates(links, true)
}

func convertShareLinksWithKeyForTest(links, secretKey string) (*conf.Config, error) {
	result, err := ConvertShareLinksToXrayJson(links, secretKey)
	if err != nil {
		return nil, err
	}
	var config conf.Config
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
