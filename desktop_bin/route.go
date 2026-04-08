package main

import (
	"fmt"
	"time"
)

const (
	routeInitRetryAttempts = 3
	routeInitRetryInterval = 2 * time.Second
)

func retryRouteInitStep(step string, fn func() error) error {
	var err error
	for i := 1; i <= routeInitRetryAttempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		if i < routeInitRetryAttempts {
			time.Sleep(routeInitRetryInterval)
		}
	}
	return fmt.Errorf("%s failed after %d attempts: %w", step, routeInitRetryAttempts, err)
}
