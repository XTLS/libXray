package nodep

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"time"
)

const (
	PingDelayTimeout int64 = 11000
	PingDelayError   int64 = 10000
)

// get the delay of some outbound.
// timeout means how long the http request will be cancelled if no response, in units of seconds.
// url means the website we use to test speed. "https://www.google.com" is a good choice for most cases.
// proxy means the local http/socks5 proxy, like "socks5://[::1]:1080". If proxy is empty, it means no proxy.
func MeasureDelay(timeout int, url string, proxy string) (int64, error) {
	httpTimeout := time.Second * time.Duration(timeout)
	c, err := CoreHTTPClient(httpTimeout, proxy)
	if err != nil {
		return PingDelayError, err
	}
	delay, err := PingHTTPRequest(c, url, timeout)
	if err != nil {
		return delay, err
	}

	return delay, nil
}

func CoreHTTPClient(timeout time.Duration, proxy string) (*http.Client, error) {
	tr := &http.Transport{
		DisableKeepAlives: true,
	}

	if len(proxy) > 0 {
		tr.Proxy = func(r *http.Request) (*url.URL, error) {
			return url.Parse(proxy)
		}
	}

	c := &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}

	return c, nil
}

func PingHTTPRequest(c *http.Client, url string, timeout int) (int64, error) {
	start := time.Now()
	req, _ := http.NewRequest("HEAD", url, nil)
	_, err := c.Do(req)
	delay := time.Since(start).Milliseconds()
	if err != nil {
		precision := delay - int64(timeout)*1000
		if math.Abs(float64(precision)) < 50 {
			return PingDelayTimeout, err
		} else {
			return PingDelayError, err
		}
	}
	return delay, nil
}

func MeasureTCPDelay(timeout int, host string, port int, proxy string) (int64, error) {
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	httpTimeout := time.Second * time.Duration(timeout)

	var dialer *net.Dialer
	var err error

	if len(proxy) > 0 {
		// 代理进行TCP
		dialer, err = createProxyDialer(httpTimeout, proxy)
		if err != nil {
			return PingDelayError, err
		}
	} else {
		// 直连TCP
		dialer = &net.Dialer{
			Timeout: httpTimeout,
		}
	}

	start := time.Now()
	conn, err := dialer.Dial("tcp", address)
	delay := time.Since(start).Milliseconds()

	if conn != nil {
		conn.Close()
	}

	if err != nil {
		precision := delay - int64(timeout)*1000
		if math.Abs(float64(precision)) < 50 {
			return PingDelayTimeout, err
		} else {
			return PingDelayError, err
		}
	}

	return delay, nil
}


func MeasureProxyConnectDelay(timeout int, targetHost string, targetPort int, proxy string) (int64, error) {
	if len(proxy) == 0 {
		return PingDelayError, fmt.Errorf("proxy address cannot be empty")
	}

	httpTimeout := time.Second * time.Duration(timeout)

	client, err := CoreHTTPClient(httpTimeout, proxy)
	if err != nil {
		return PingDelayError, err
	}

	start := time.Now()

	var testURL string
	if targetPort == 443 {
		testURL = fmt.Sprintf("https://%s", targetHost)
	} else {
		testURL = fmt.Sprintf("http://%s:%d", targetHost, targetPort)
	}

	req, err := http.NewRequest("HEAD", testURL, nil)
	if err != nil {
		return PingDelayError, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	delay := time.Since(start).Milliseconds()

	if resp != nil {
		resp.Body.Close()
	}

	if err != nil {
		precision := delay - int64(timeout)*1000
		if math.Abs(float64(precision)) < 50 {
			return PingDelayTimeout, err
		} else {
			return PingDelayError, err
		}
	}

	return delay, nil
}

func createProxyDialer(timeout time.Duration, proxy string) (*net.Dialer, error) {
	_, err := url.Parse(proxy)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{
		Timeout: timeout,
	}

	return dialer, nil
}
