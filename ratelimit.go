package gocronometer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	// ThrottleConfigURL returns Cronometer's published per-API rate limits (rps).
	ThrottleConfigURL = "https://cronometer.com/api/v3/throttle-config"

	// DefaultGWTRateLimit is the requests/second cap used before the live
	// throttle-config is fetched. Deliberately well under the observed gwt_rpc
	// limit of 40 rps so we never approach it.
	DefaultGWTRateLimit = 20.0

	// ThrottleSafetyFactor is the fraction of Cronometer's published gwt_rpc rps
	// we actually use, leaving headroom so concurrent web/app sessions and clock
	// skew can't push us over the real limit.
	ThrottleSafetyFactor = 0.5
)

// rateLimiter caps the request rate by reserving a minimum interval between
// requests. Safe for concurrent use; each caller reserves the next departure
// slot under lock, so sequential and parallel callers both honor the rate.
type rateLimiter struct {
	mu       sync.Mutex
	interval time.Duration
	next     time.Time
}

func newRateLimiter(rps float64) *rateLimiter {
	return &rateLimiter{interval: intervalForRPS(rps)}
}

func intervalForRPS(rps float64) time.Duration {
	if rps <= 0 {
		return 0
	}
	return time.Duration(float64(time.Second) / rps)
}

// SetRate updates the allowed requests/second (<=0 disables limiting).
func (l *rateLimiter) SetRate(rps float64) {
	l.mu.Lock()
	l.interval = intervalForRPS(rps)
	l.mu.Unlock()
}

func (l *rateLimiter) wait(ctx context.Context) error {
	l.mu.Lock()
	if l.interval <= 0 {
		l.mu.Unlock()
		return nil
	}
	now := time.Now()
	if l.next.Before(now) {
		l.next = now
	}
	delay := l.next.Sub(now)
	l.next = l.next.Add(l.interval)
	l.mu.Unlock()

	if delay <= 0 {
		return nil
	}
	t := time.NewTimer(delay)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// rateLimitedTransport throttles every request through the client to honor
// Cronometer's published rate limits.
type rateLimitedTransport struct {
	base    http.RoundTripper
	limiter *rateLimiter
}

func (t *rateLimitedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.limiter.wait(req.Context()); err != nil {
		return nil, err
	}
	return t.base.RoundTrip(req)
}

// ThrottleConfig mirrors Cronometer's /api/v3/throttle-config response.
type ThrottleConfig struct {
	APIs map[string]struct {
		Global struct {
			RPS float64 `json:"rps"`
		} `json:"global"`
	} `json:"apis"`
}

// GetThrottleConfig fetches Cronometer's current published rate-limit config.
func (c *Client) GetThrottleConfig(ctx context.Context) (*ThrottleConfig, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", ThrottleConfigURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("throttle-config request: %w", err)
	}
	defer closeAndExhaustReader(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("throttle-config returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var tc ThrottleConfig
	if err := json.Unmarshal(body, &tc); err != nil {
		return nil, fmt.Errorf("throttle-config decode: %w", err)
	}
	return &tc, nil
}

// applyThrottleConfig fetches the live config and sets the client's rate limiter
// to ThrottleSafetyFactor of the published gwt_rpc limit. Best-effort: on any
// failure the conservative default rate stays in effect.
func (c *Client) applyThrottleConfig(ctx context.Context) {
	if c.limiter == nil {
		return
	}
	tc, err := c.GetThrottleConfig(ctx)
	if err != nil || tc == nil {
		return
	}
	if api, ok := tc.APIs["gwt_rpc"]; ok && api.Global.RPS > 0 {
		c.limiter.SetRate(api.Global.RPS * ThrottleSafetyFactor)
	}
}
