// Package config handles loading and validating pingboard configuration from YAML files.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Endpoint represents a single HTTP endpoint to monitor.
type Endpoint struct {
	Name         string            `yaml:"name"`
	URL          string            `yaml:"url"`
	Method       string            `yaml:"method,omitempty"`
	Headers      map[string]string `yaml:"headers,omitempty"`
	Timeout      time.Duration     `yaml:"timeout,omitempty"`
	Interval     time.Duration     `yaml:"interval,omitempty"`
	ExpectStatus int               `yaml:"expect_status,omitempty"`
}

// Alert configures how downtime notifications are delivered.
type Alert struct {
	Type       string `yaml:"type"` // "stdout", "webhook", "log"
	WebhookURL string `yaml:"webhook_url,omitempty"`
	LogFile    string `yaml:"log_file,omitempty"`
	Threshold  int    `yaml:"threshold,omitempty"` // consecutive failures before alerting
}

// Dashboard configures the terminal UI rendering.
type Dashboard struct {
	RefreshRate time.Duration `yaml:"refresh_rate,omitempty"`
	ShowHeaders bool          `yaml:"show_headers,omitempty"`
	Compact     bool          `yaml:"compact,omitempty"`
	MaxHistory  int           `yaml:"max_history,omitempty"`
}

// Config is the top-level configuration structure loaded from a YAML file.
type Config struct {
	Endpoints []Endpoint `yaml:"endpoints"`
	Alerts    []Alert    `yaml:"alerts,omitempty"`
	Dashboard Dashboard  `yaml:"dashboard,omitempty"`
	Defaults  Defaults   `yaml:"defaults,omitempty"`
}

// Defaults provides fallback values for endpoint settings.
type Defaults struct {
	Method       string        `yaml:"method,omitempty"`
	Timeout      time.Duration `yaml:"timeout,omitempty"`
	Interval     time.Duration `yaml:"interval,omitempty"`
	ExpectStatus int           `yaml:"expect_status,omitempty"`
}

// Load reads a YAML config file from disk and returns a validated Config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	applyDefaults(cfg)

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return cfg, nil
}

// applyDefaults fills in missing endpoint values from the defaults section.
func applyDefaults(cfg *Config) {
	if cfg.Defaults.Method == "" {
		cfg.Defaults.Method = "GET"
	}
	if cfg.Defaults.Timeout == 0 {
		cfg.Defaults.Timeout = 10 * time.Second
	}
	if cfg.Defaults.Interval == 0 {
		cfg.Defaults.Interval = 30 * time.Second
	}
	if cfg.Defaults.ExpectStatus == 0 {
		cfg.Defaults.ExpectStatus = 200
	}

	if cfg.Dashboard.RefreshRate == 0 {
		cfg.Dashboard.RefreshRate = 2 * time.Second
	}
	if cfg.Dashboard.MaxHistory == 0 {
		cfg.Dashboard.MaxHistory = 100
	}
	cfg.Dashboard.ShowHeaders = true

	for i := range cfg.Endpoints {
		ep := &cfg.Endpoints[i]
		if ep.Method == "" {
			ep.Method = cfg.Defaults.Method
		}
		if ep.Timeout == 0 {
			ep.Timeout = cfg.Defaults.Timeout
		}
		if ep.Interval == 0 {
			ep.Interval = cfg.Defaults.Interval
		}
		if ep.ExpectStatus == 0 {
			ep.ExpectStatus = cfg.Defaults.ExpectStatus
		}
	}

	for i := range cfg.Alerts {
		if cfg.Alerts[i].Threshold == 0 {
			cfg.Alerts[i].Threshold = 3
		}
	}
}

// validate checks the config for logical errors.
func validate(cfg *Config) error {
	if len(cfg.Endpoints) == 0 {
		return fmt.Errorf("at least one endpoint is required")
	}

	seen := make(map[string]bool)
	for i, ep := range cfg.Endpoints {
		if ep.Name == "" {
			return fmt.Errorf("endpoint[%d]: name is required", i)
		}
		if ep.URL == "" {
			return fmt.Errorf("endpoint[%d] (%s): url is required", i, ep.Name)
		}
		if seen[ep.Name] {
			return fmt.Errorf("endpoint[%d]: duplicate name %q", i, ep.Name)
		}
		seen[ep.Name] = true

		if ep.Timeout < 0 {
			return fmt.Errorf("endpoint[%d] (%s): timeout must be positive", i, ep.Name)
		}
		if ep.Interval < time.Second {
			return fmt.Errorf("endpoint[%d] (%s): interval must be >= 1s", i, ep.Name)
		}
	}

	for i, alert := range cfg.Alerts {
		switch alert.Type {
		case "stdout", "log", "webhook":
		default:
			return fmt.Errorf("alert[%d]: unknown type %q (use stdout, log, or webhook)", i, alert.Type)
		}
		if alert.Type == "webhook" && alert.WebhookURL == "" {
			return fmt.Errorf("alert[%d]: webhook type requires webhook_url", i)
		}
		if alert.Type == "log" && alert.LogFile == "" {
			return fmt.Errorf("alert[%d]: log type requires log_file", i)
		}
	}

	return nil
}

// ExampleConfig returns a sample YAML config string for reference.
func ExampleConfig() string {
	return `# pingboard configuration
defaults:
  method: GET
  timeout: 10s
  interval: 30s
  expect_status: 200

endpoints:
  - name: GitHub API
    url: https://api.github.com
    interval: 15s

  - name: Google
    url: https://www.google.com
    timeout: 5s

alerts:
  - type: stdout
    threshold: 3

dashboard:
  refresh_rate: 2s
  compact: false
  max_history: 100
`
}
