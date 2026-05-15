// Package alerter handles downtime notifications via stdout, webhook, or log file.
package alerter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Lappy000/pingboard/internal/config"
	"github.com/Lappy000/pingboard/internal/monitor"
)

// Event represents a state change that triggers an alert.
type Event struct {
	Endpoint  string    `json:"endpoint"`
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	Latency   string    `json:"latency,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Downtime  string    `json:"downtime,omitempty"`
	Recovered bool      `json:"recovered"`
}

// Alerter manages alert delivery based on probe results.
type Alerter struct {
	alerts     []config.Alert
	endpoints  []config.Endpoint
	failCounts map[string]int
	downSince  map[string]time.Time
	notified   map[string]bool
	mu         sync.Mutex
	httpClient *http.Client
	logFiles   map[string]*os.File
}

// New creates an Alerter instance with the given alert configurations.
func New(alerts []config.Alert, endpoints []config.Endpoint) *Alerter {
	return &Alerter{
		alerts:     alerts,
		endpoints:  endpoints,
		failCounts: make(map[string]int),
		downSince:  make(map[string]time.Time),
		notified:   make(map[string]bool),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logFiles:   make(map[string]*os.File),
	}
}

// ProcessResult evaluates a probe result and fires alerts if thresholds are met.
func (a *Alerter) ProcessResult(result monitor.ProbeResult) {
	a.mu.Lock()
	defer a.mu.Unlock()

	name := result.Endpoint

	if result.Status == monitor.StatusDown {
		a.failCounts[name]++

		// Record when downtime started
		if _, exists := a.downSince[name]; !exists {
			a.downSince[name] = result.Timestamp
		}

		// Check each alert's threshold
		for _, alert := range a.alerts {
			if a.failCounts[name] >= alert.Threshold && !a.notified[name] {
				event := Event{
					Endpoint:  name,
					URL:       a.getEndpointURL(name),
					Status:    "DOWN",
					Error:     result.Error,
					Latency:   result.Latency.String(),
					Timestamp: result.Timestamp,
					Recovered: false,
				}
				a.sendAlert(alert, event)
				a.notified[name] = true
			}
		}
	} else {
		// Endpoint recovered
		if a.notified[name] {
			downtime := ""
			if since, ok := a.downSince[name]; ok {
				downtime = time.Since(since).Round(time.Second).String()
			}

			event := Event{
				Endpoint:  name,
				URL:       a.getEndpointURL(name),
				Status:    "UP",
				Latency:   result.Latency.String(),
				Timestamp: result.Timestamp,
				Downtime:  downtime,
				Recovered: true,
			}

			for _, alert := range a.alerts {
				a.sendAlert(alert, event)
			}
		}

		// Reset tracking
		a.failCounts[name] = 0
		delete(a.downSince, name)
		a.notified[name] = false
	}
}

// Close releases any open file handles.
func (a *Alerter) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, f := range a.logFiles {
		f.Close()
	}
}

// sendAlert dispatches an event through the appropriate channel.
func (a *Alerter) sendAlert(alert config.Alert, event Event) {
	switch alert.Type {
	case "stdout":
		a.alertStdout(event)
	case "webhook":
		a.alertWebhook(alert.WebhookURL, event)
	case "log":
		a.alertLog(alert.LogFile, event)
	}
}

// alertStdout prints a formatted alert to the terminal.
func (a *Alerter) alertStdout(event Event) {
	ts := event.Timestamp.Format("15:04:05")

	if event.Recovered {
		fmt.Printf("\033[92m[%s] RECOVERED: %s is back UP (was down for %s)\033[0m\n",
			ts, event.Endpoint, event.Downtime)
	} else {
		fmt.Printf("\033[91m[%s] ALERT: %s is DOWN - %s\033[0m\n",
			ts, event.Endpoint, event.Error)
	}
}

// alertWebhook sends a JSON payload to a webhook URL.
func (a *Alerter) alertWebhook(url string, event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		fmt.Fprintf(os.Stderr, "alerter: failed to marshal event: %v\n", err)
		return
	}

	resp, err := a.httpClient.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		fmt.Fprintf(os.Stderr, "alerter: webhook request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		fmt.Fprintf(os.Stderr, "alerter: webhook returned status %d\n", resp.StatusCode)
	}
}

// alertLog writes events to a log file in JSON Lines format.
func (a *Alerter) alertLog(path string, event Event) {
	f, ok := a.logFiles[path]
	if !ok {
		var err error
		f, err = os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "alerter: failed to open log file %s: %v\n", path, err)
			return
		}
		a.logFiles[path] = f
	}

	line, err := json.Marshal(event)
	if err != nil {
		return
	}
	line = append(line, '\n')
	f.Write(line)
}

// getEndpointURL finds the URL for a named endpoint.
func (a *Alerter) getEndpointURL(name string) string {
	for _, ep := range a.endpoints {
		if ep.Name == name {
			return ep.URL
		}
	}
	return ""
}

// Summary returns a formatted summary of current alert state.
func (a *Alerter) Summary() string {
	a.mu.Lock()
	defer a.mu.Unlock()

	var down []string
	for name, notified := range a.notified {
		if notified {
			since := a.downSince[name]
			duration := time.Since(since).Round(time.Second)
			down = append(down, fmt.Sprintf("%s (down %s)", name, duration))
		}
	}

	if len(down) == 0 {
		return "All endpoints healthy"
	}
	return fmt.Sprintf("ALERTS ACTIVE: %s", strings.Join(down, ", "))
}
