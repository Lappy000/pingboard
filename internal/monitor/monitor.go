// Package monitor implements concurrent HTTP endpoint checking with latency tracking.
package monitor

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/Lappy000/pingboard/internal/config"
)

// Status represents the health state of an endpoint.
type Status int

const (
	StatusUnknown Status = iota
	StatusUp
	StatusDown
	StatusDegraded
)

// String returns a human-readable status label.
func (s Status) String() string {
	switch s {
	case StatusUp:
		return "UP"
	case StatusDown:
		return "DOWN"
	case StatusDegraded:
		return "SLOW"
	default:
		return "---"
	}
}

// ProbeResult captures the outcome of a single HTTP health check.
type ProbeResult struct {
	Endpoint    string
	Status      Status
	StatusCode  int
	Latency     time.Duration
	Error       string
	Timestamp   time.Time
}

// EndpointStats maintains rolling statistics for a monitored endpoint.
type EndpointStats struct {
	Name            string
	URL             string
	CurrentStatus   Status
	LastLatency     time.Duration
	AvgLatency      time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
	P95Latency      time.Duration
	TotalChecks     int64
	TotalSuccesses  int64
	TotalFailures   int64
	ConsecFailures  int
	UptimePercent   float64
	LastChecked     time.Time
	LastError       string
	History         []ProbeResult
}

// Monitor orchestrates concurrent endpoint health checking.
type Monitor struct {
	endpoints []config.Endpoint
	stats     map[string]*EndpointStats
	mu        sync.RWMutex
	client    *http.Client
	ctx       context.Context
	cancel    context.CancelFunc
	results   chan ProbeResult
	onResult  func(ProbeResult)
}

// Option configures the Monitor behavior.
type Option func(*Monitor)

// WithResultCallback sets a function to call when new probe results arrive.
func WithResultCallback(fn func(ProbeResult)) Option {
	return func(m *Monitor) {
		m.onResult = fn
	}
}

// New creates a new Monitor for the given endpoints.
func New(endpoints []config.Endpoint, opts ...Option) *Monitor {
	ctx, cancel := context.WithCancel(context.Background())

	m := &Monitor{
		endpoints: endpoints,
		stats:     make(map[string]*EndpointStats),
		ctx:       ctx,
		cancel:    cancel,
		results:   make(chan ProbeResult, len(endpoints)*4),
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig:     &tls.Config{InsecureSkipVerify: false},
				MaxIdleConnsPerHost: 4,
				IdleConnTimeout:     90 * time.Second,
				DisableKeepAlives:   false,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}

	for _, ep := range endpoints {
		m.stats[ep.Name] = &EndpointStats{
			Name:       ep.Name,
			URL:        ep.URL,
			MinLatency: time.Duration(math.MaxInt64),
			History:    make([]ProbeResult, 0, 100),
		}
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Start begins monitoring all endpoints concurrently.
func (m *Monitor) Start() {
	// Launch result collector
	go m.collectResults()

	// Launch a goroutine for each endpoint
	for _, ep := range m.endpoints {
		go m.probeLoop(ep)
	}
}

// Stop gracefully shuts down all monitoring goroutines.
func (m *Monitor) Stop() {
	m.cancel()
}

// GetStats returns a snapshot of all endpoint statistics.
func (m *Monitor) GetStats() []*EndpointStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*EndpointStats, 0, len(m.stats))
	for _, ep := range m.endpoints {
		if s, ok := m.stats[ep.Name]; ok {
			// Return a copy to avoid data races
			cpy := *s
			cpy.History = make([]ProbeResult, len(s.History))
			copy(cpy.History, s.History)
			result = append(result, &cpy)
		}
	}
	return result
}

// GetEndpointStats returns stats for a specific endpoint.
func (m *Monitor) GetEndpointStats(name string) (*EndpointStats, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.stats[name]
	if !ok {
		return nil, false
	}
	cpy := *s
	return &cpy, true
}

// Results returns the channel for probe results (for alerting integration).
func (m *Monitor) Results() <-chan ProbeResult {
	return m.results
}

// probeLoop runs periodic health checks for a single endpoint.
func (m *Monitor) probeLoop(ep config.Endpoint) {
	// Perform an initial probe immediately
	m.probe(ep)

	ticker := time.NewTicker(ep.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.probe(ep)
		}
	}
}

// probe performs a single HTTP health check for an endpoint.
func (m *Monitor) probe(ep config.Endpoint) {
	ctx, cancel := context.WithTimeout(m.ctx, ep.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, ep.Method, ep.URL, nil)
	if err != nil {
		m.recordResult(ProbeResult{
			Endpoint:  ep.Name,
			Status:    StatusDown,
			Error:     fmt.Sprintf("request creation failed: %v", err),
			Timestamp: time.Now(),
		})
		return
	}

	// Apply custom headers
	req.Header.Set("User-Agent", "pingboard/1.0")
	for k, v := range ep.Headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := m.client.Do(req)
	latency := time.Since(start)

	if err != nil {
		m.recordResult(ProbeResult{
			Endpoint:  ep.Name,
			Status:    StatusDown,
			Latency:   latency,
			Error:     fmt.Sprintf("request failed: %v", err),
			Timestamp: time.Now(),
		})
		return
	}
	defer resp.Body.Close()

	status := StatusUp
	if resp.StatusCode != ep.ExpectStatus {
		status = StatusDown
	} else if latency > ep.Timeout/2 {
		status = StatusDegraded
	}

	m.recordResult(ProbeResult{
		Endpoint:   ep.Name,
		Status:     status,
		StatusCode: resp.StatusCode,
		Latency:    latency,
		Timestamp:  time.Now(),
	})
}

// recordResult updates endpoint stats with a new probe result.
func (m *Monitor) recordResult(result ProbeResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.stats[result.Endpoint]
	if !ok {
		return
	}

	s.CurrentStatus = result.Status
	s.LastLatency = result.Latency
	s.LastChecked = result.Timestamp
	s.LastError = result.Error
	s.TotalChecks++

	if result.Status == StatusUp || result.Status == StatusDegraded {
		s.TotalSuccesses++
		s.ConsecFailures = 0
	} else {
		s.TotalFailures++
		s.ConsecFailures++
	}

	// Update latency stats (only for successful probes)
	if result.Latency > 0 && result.Status != StatusDown {
		if result.Latency < s.MinLatency {
			s.MinLatency = result.Latency
		}
		if result.Latency > s.MaxLatency {
			s.MaxLatency = result.Latency
		}
		// Rolling average
		if s.TotalSuccesses > 0 {
			s.AvgLatency = time.Duration(
				(int64(s.AvgLatency)*int64(s.TotalSuccesses-1) + int64(result.Latency)) / int64(s.TotalSuccesses),
			)
		}
	}

	// Calculate uptime percentage
	if s.TotalChecks > 0 {
		s.UptimePercent = float64(s.TotalSuccesses) / float64(s.TotalChecks) * 100.0
	}

	// Maintain history ring buffer
	s.History = append(s.History, result)
	if len(s.History) > 100 {
		s.History = s.History[len(s.History)-100:]
	}

	// Calculate P95 latency from history
	s.P95Latency = calculateP95(s.History)

	// Send to results channel (non-blocking)
	select {
	case m.results <- result:
	default:
	}

	// Callback if registered
	if m.onResult != nil {
		go m.onResult(result)
	}
}

// collectResults drains the results channel (prevents blocking).
func (m *Monitor) collectResults() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-m.results:
			// Results consumed by alerter or discarded
		}
	}
}

// calculateP95 computes the 95th percentile latency from probe history.
func calculateP95(history []ProbeResult) time.Duration {
	var latencies []time.Duration
	for _, r := range history {
		if r.Latency > 0 && r.Status != StatusDown {
			latencies = append(latencies, r.Latency)
		}
	}

	if len(latencies) == 0 {
		return 0
	}

	// Sort latencies for percentile calculation (O(n log n))
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	idx := int(float64(len(latencies)) * 0.95)
	if idx >= len(latencies) {
		idx = len(latencies) - 1
	}
	return latencies[idx]
}
