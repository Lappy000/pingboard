package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadValidConfig(t *testing.T) {
	yaml := `
endpoints:
  - name: Test API
    url: https://example.com
    interval: 5s
    timeout: 3s
alerts:
  - type: stdout
    threshold: 2
dashboard:
  refresh_rate: 1s
`
	path := writeTemp(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(cfg.Endpoints))
	}
	ep := cfg.Endpoints[0]
	if ep.Name != "Test API" {
		t.Errorf("expected name 'Test API', got %q", ep.Name)
	}
	if ep.Interval != 5*time.Second {
		t.Errorf("expected interval 5s, got %v", ep.Interval)
	}
	if ep.Timeout != 3*time.Second {
		t.Errorf("expected timeout 3s, got %v", ep.Timeout)
	}
	if ep.Method != "GET" {
		t.Errorf("expected default method GET, got %q", ep.Method)
	}
	if ep.ExpectStatus != 200 {
		t.Errorf("expected default expect_status 200, got %d", ep.ExpectStatus)
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	yaml := `
defaults:
  method: POST
  timeout: 20s
  interval: 60s
  expect_status: 201
endpoints:
  - name: Endpoint A
    url: https://a.example.com
  - name: Endpoint B
    url: https://b.example.com
    method: PUT
    timeout: 5s
`
	path := writeTemp(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	a := cfg.Endpoints[0]
	if a.Method != "POST" {
		t.Errorf("endpoint A: expected method POST from defaults, got %q", a.Method)
	}
	if a.Timeout != 20*time.Second {
		t.Errorf("endpoint A: expected timeout 20s from defaults, got %v", a.Timeout)
	}
	if a.ExpectStatus != 201 {
		t.Errorf("endpoint A: expected status 201 from defaults, got %d", a.ExpectStatus)
	}

	b := cfg.Endpoints[1]
	if b.Method != "PUT" {
		t.Errorf("endpoint B: expected overridden method PUT, got %q", b.Method)
	}
	if b.Timeout != 5*time.Second {
		t.Errorf("endpoint B: expected overridden timeout 5s, got %v", b.Timeout)
	}
}

func TestLoadRejectsEmptyEndpoints(t *testing.T) {
	yaml := `endpoints: []`
	path := writeTemp(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for empty endpoints, got nil")
	}
}

func TestLoadRejectsDuplicateNames(t *testing.T) {
	yaml := `
endpoints:
  - name: Dup
    url: https://a.example.com
  - name: Dup
    url: https://b.example.com
`
	path := writeTemp(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for duplicate names, got nil")
	}
}

func TestLoadRejectsMissingURL(t *testing.T) {
	yaml := `
endpoints:
  - name: NoURL
`
	path := writeTemp(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing URL, got nil")
	}
}

func TestLoadRejectsUnknownAlertType(t *testing.T) {
	yaml := `
endpoints:
  - name: Test
    url: https://example.com
alerts:
  - type: email
    threshold: 1
`
	path := writeTemp(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for unknown alert type, got nil")
	}
}

func TestLoadRejectsWebhookWithoutURL(t *testing.T) {
	yaml := `
endpoints:
  - name: Test
    url: https://example.com
alerts:
  - type: webhook
    threshold: 1
`
	path := writeTemp(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for webhook without URL, got nil")
	}
}

func TestExampleConfigIsValid(t *testing.T) {
	path := writeTemp(t, ExampleConfig())
	_, err := Load(path)
	if err != nil {
		t.Fatalf("ExampleConfig() should produce valid config, got: %v", err)
	}
}

func TestDashboardDefaults(t *testing.T) {
	yaml := `
endpoints:
  - name: Test
    url: https://example.com
`
	path := writeTemp(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Dashboard.RefreshRate != 2*time.Second {
		t.Errorf("expected default refresh_rate 2s, got %v", cfg.Dashboard.RefreshRate)
	}
	if cfg.Dashboard.MaxHistory != 100 {
		t.Errorf("expected default max_history 100, got %d", cfg.Dashboard.MaxHistory)
	}
}

// writeTemp writes content to a temporary YAML file and returns its path.
func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}
