package nodep

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	PingDelayTimeout int64 = 11000
	PingDelayError   int64 = 10000
)

type geoLocation struct {
	Ip string `json:"ip,omitempty"`
	Cc string `json:"cc,omitempty"`
}

// Find the delay and country code of some outbound.
// timeout means how long the http request will be cancelled if no response, in units of seconds.
// url means the website we use to test speed. "https://www.google.com/gen_204" is a good choice for most cases.
// times means how many times we should test the url.
// proxy means the local http/socks5 proxy, like "http://127.0.0.1:1080".

func MeasureDelay(timeout int, url string, times int, proxy string) string {
	httpTimeout := time.Second * time.Duration(timeout)
	c, err := coreHTTPClient(httpTimeout, proxy)
	if err != nil {
		return fmt.Sprintf("%d::%s", PingDelayError, err)
	}
	delaySum := int64(0)
	count := int64(0)
	isValid := false
	lastErr := ""
	for i := 0; i < times; i++ {
		delay, err := pingHTTPRequest(c, url)
		if delay != PingDelayTimeout {
			delaySum += delay
			count += 1
			isValid = true
		} else {
			lastErr = err.Error()
		}
	}
	if !isValid {
		return fmt.Sprintf("%d::%s", PingDelayTimeout, lastErr)
	}
	country, err := geolocationHTTPRequest(c)
	if err != nil {
		fmt.Println("geolocation error: ", err)
	}

	return fmt.Sprintf("%d:%s:%s", delaySum/count, country, lastErr)
}

func coreHTTPClient(timeout time.Duration, proxy string) (*http.Client, error) {
	tr := &http.Transport{
		DisableKeepAlives: true,
		Proxy: func(r *http.Request) (*url.URL, error) {
			return url.Parse(proxy)
		},
	}

	c := &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}

	return c, nil
}

func pingHTTPRequest(c *http.Client, url string) (int64, error) {
	start := time.Now()
	req, _ := http.NewRequest("GET", url, nil)
	_, err := c.Do(req)
	if err != nil {
		return PingDelayTimeout, err
	}
	return time.Since(start).Milliseconds(), nil
}

func geolocationHTTPRequest(c *http.Client) (string, error) {
	req, _ := http.NewRequest("GET", "https://ident.me/json", nil)
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var location geoLocation
	if err = json.Unmarshal(body, &location); err != nil {
		return "", err
	}
	return location.Cc, nil
}
