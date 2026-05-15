package monitor

import (
	"testing"
	"time"

	"github.com/Lappy000/pingboard/internal/config"
)

func TestNewMonitor(t *testing.T) {
	endpoints := []config.Endpoint{
		{Name: "Test1", URL: "https://example.com", Method: "GET", Timeout: 5 * time.Second, Interval: 10 * time.Second, ExpectStatus: 200},
		{Name: "Test2", URL: "https://example.org", Method: "GET", Timeout: 5 * time.Second, Interval: 10 * time.Second, ExpectStatus: 200},
	}

	m := New(endpoints)
	if m == nil {
		t.Fatal("New returned nil")
	}

	stats := m.GetStats()
	if len(stats) != 2 {
		t.Fatalf("expected 2 stats entries, got %d", len(stats))
	}
	if stats[0].Name != "Test1" {
		t.Errorf("expected first stat name 'Test1', got %q", stats[0].Name)
	}
	if stats[1].Name != "Test2" {
		t.Errorf("expected second stat name 'Test2', got %q", stats[1].Name)
	}
}

func TestCalculateP95(t *testing.T) {
	tests := []struct {
		name     string
		history  []ProbeResult
		expected time.Duration
	}{
		{
			name:     "empty history",
			history:  nil,
			expected: 0,
		},
		{
			name: "single result",
			history: []ProbeResult{
				{Latency: 100 * time.Millisecond, Status: StatusUp},
			},
			expected: 100 * time.Millisecond,
		},
		{
			name: "skips down results",
			history: []ProbeResult{
				{Latency: 100 * time.Millisecond, Status: StatusUp},
				{Latency: 500 * time.Millisecond, Status: StatusDown},
				{Latency: 200 * time.Millisecond, Status: StatusUp},
			},
			expected: 200 * time.Millisecond, // P95 of [100ms, 200ms]
		},
		{
			name:     "many results returns 95th percentile",
			history:  generateHistory(100),
			expected: 95 * time.Millisecond, // 0-99ms, P95 = 95ms
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateP95(tc.history)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestStatusString(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusUp, "UP"},
		{StatusDown, "DOWN"},
		{StatusDegraded, "SLOW"},
		{StatusUnknown, "---"},
	}
	for _, tc := range tests {
		if got := tc.status.String(); got != tc.expected {
			t.Errorf("Status(%d).String() = %q, want %q", tc.status, got, tc.expected)
		}
	}
}

func TestRecordResultUpdatesStats(t *testing.T) {
	endpoints := []config.Endpoint{
		{Name: "Test", URL: "https://example.com", Method: "GET", Timeout: 5 * time.Second, Interval: 10 * time.Second, ExpectStatus: 200},
	}

	m := New(endpoints)

	// Record a successful result
	m.recordResult(ProbeResult{
		Endpoint:  "Test",
		Status:    StatusUp,
		Latency:   50 * time.Millisecond,
		Timestamp: time.Now(),
	})

	stats, ok := m.GetEndpointStats("Test")
	if !ok {
		t.Fatal("expected to find stats for 'Test'")
	}
	if stats.TotalChecks != 1 {
		t.Errorf("expected 1 total check, got %d", stats.TotalChecks)
	}
	if stats.TotalSuccesses != 1 {
		t.Errorf("expected 1 success, got %d", stats.TotalSuccesses)
	}
	if stats.CurrentStatus != StatusUp {
		t.Errorf("expected StatusUp, got %v", stats.CurrentStatus)
	}

	// Record a failure
	m.recordResult(ProbeResult{
		Endpoint:  "Test",
		Status:    StatusDown,
		Error:     "connection refused",
		Timestamp: time.Now(),
	})

	stats, _ = m.GetEndpointStats("Test")
	if stats.TotalFailures != 1 {
		t.Errorf("expected 1 failure, got %d", stats.TotalFailures)
	}
	if stats.ConsecFailures != 1 {
		t.Errorf("expected 1 consecutive failure, got %d", stats.ConsecFailures)
	}
	if stats.UptimePercent != 50.0 {
		t.Errorf("expected 50%% uptime, got %.1f%%", stats.UptimePercent)
	}
}

func TestWithResultCallback(t *testing.T) {
	var received []ProbeResult
	endpoints := []config.Endpoint{
		{Name: "CB", URL: "https://example.com", Method: "GET", Timeout: 5 * time.Second, Interval: 10 * time.Second, ExpectStatus: 200},
	}

	m := New(endpoints, WithResultCallback(func(r ProbeResult) {
		received = append(received, r)
	}))

	m.recordResult(ProbeResult{
		Endpoint:  "CB",
		Status:    StatusUp,
		Latency:   30 * time.Millisecond,
		Timestamp: time.Now(),
	})

	// Wait briefly for goroutine callback
	time.Sleep(50 * time.Millisecond)

	if len(received) != 1 {
		t.Fatalf("expected callback to receive 1 result, got %d", len(received))
	}
	if received[0].Endpoint != "CB" {
		t.Errorf("expected endpoint 'CB', got %q", received[0].Endpoint)
	}
}

// generateHistory creates N probe results with incrementing latencies.
func generateHistory(n int) []ProbeResult {
	results := make([]ProbeResult, n)
	for i := 0; i < n; i++ {
		results[i] = ProbeResult{
			Latency: time.Duration(i) * time.Millisecond,
			Status:  StatusUp,
		}
	}
	return results
}
