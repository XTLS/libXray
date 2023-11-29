package nodep

import (
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

// Find the delay and ip of some outbound.
// timeout means how long the http request will be cancelled if no response, in units of seconds.
// url means the website we use to test speed. "https://www.google.com" is a good choice for most cases.
// proxy means the local http/socks5 proxy, like "socks5://[::1]:1080".

func MeasureDelay(timeout int, url string, proxy string) (int64, string, error) {
	httpTimeout := time.Second * time.Duration(timeout)
	c, err := coreHTTPClient(httpTimeout, proxy)
	if err != nil {
		return PingDelayError, "", err
	}
	delay, err := pingHTTPRequest(c, url)
	if err != nil {
		return delay, "", err
	}

	ip, err := ipHTTPRequest(c)
	if err != nil {
		fmt.Println("get ip error: ", err)
	}

	return delay, ip, nil
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
	req, _ := http.NewRequest("HEAD", url, nil)
	_, err := c.Do(req)
	if err != nil {
		return PingDelayTimeout, err
	}
	return time.Since(start).Milliseconds(), nil
}

func ipHTTPRequest(c *http.Client) (string, error) {
	req, _ := http.NewRequest("GET", "https://api.seeip.org/", nil)
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	ip := string(body)
	return ip, nil
}
