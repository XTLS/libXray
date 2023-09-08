package nodep

import (
	"fmt"
	"io"
	"net"
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
// times means how many times we should test the url.
// proxy means the local http/socks5 proxy, like "socks5://[::1]:1080".

func MeasureDelay(timeout int, url string, times int, proxy string) (int64, string, error) {
	httpTimeout := time.Second * time.Duration(timeout)
	c, err := coreHTTPClient(httpTimeout, proxy)
	if err != nil {
		return PingDelayError, "", err
	}
	delaySum := int64(0)
	count := int64(0)
	isValid := false
	var lastErr error
	for i := 0; i < times; i++ {
		delay, err := pingHTTPRequest(c, url)
		if delay != PingDelayTimeout {
			delaySum += delay
			count += 1
			isValid = true
		} else {
			lastErr = err
		}
	}
	if !isValid {
		return PingDelayTimeout, "", lastErr
	}
	ip, err := ipHTTPRequest(c)
	if err != nil {
		fmt.Println("get ip error: ", err)
	}

	return delaySum / count, ip, lastErr
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

// Find the delay of some outbound.
// timeout means how long the tcp connection will be cancelled if no response, in units of seconds.
// server means the destination we use to test speed, like "8.8.8.8:853".
// times means how many times we should test the server.
func TcpPing(timeout int, server string, times int) string {
	tcpTimeout := time.Second * time.Duration(timeout)
	delaySum := int64(0)
	count := int64(0)
	isValid := false
	lastErr := ""
	for i := 0; i < times; i++ {
		delay, err := tcpPing(tcpTimeout, server)
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

	return fmt.Sprintf("%d::%s", delaySum/count, lastErr)
}

func tcpPing(timeout time.Duration, server string) (int64, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", server, timeout)
	if err != nil {
		return PingDelayTimeout, err
	}
	defer conn.Close()

	rtt := time.Since(start).Milliseconds()
	return rtt, nil
}
