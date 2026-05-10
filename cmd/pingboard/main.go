// pingboard - Real-time terminal dashboard for HTTP endpoint monitoring.
//
// Usage:
//
//	pingboard [flags]
//	pingboard -config config.yaml
//	pingboard -example > config.yaml
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Lappy000/pingboard/internal/alerter"
	"github.com/Lappy000/pingboard/internal/config"
	"github.com/Lappy000/pingboard/internal/monitor"
	"github.com/Lappy000/pingboard/internal/ui"
)

var (
	version = "1.0.0"
	commit  = "dev"
	date    = "unknown"
)

func main() {
	// Command-line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	showExample := flag.Bool("example", false, "Print example configuration to stdout")
	compact := flag.Bool("compact", false, "Use compact display mode")
	oneShot := flag.Bool("once", false, "Run a single check and exit (useful for CI)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("pingboard %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	if *showExample {
		fmt.Print(config.ExampleConfig())
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nRun 'pingboard -example' to generate a sample config file.\n")
		os.Exit(1)
	}

	// Initialize the alerter
	alert := alerter.New(cfg.Alerts, cfg.Endpoints)
	defer alert.Close()

	// Initialize the monitor with alert callback
	mon := monitor.New(cfg.Endpoints, monitor.WithResultCallback(func(r monitor.ProbeResult) {
		alert.ProcessResult(r)
	}))

	// Start monitoring
	mon.Start()
	defer mon.Stop()

	// Handle single-shot mode (useful for CI/scripting)
	if *oneShot {
		runOneShotMode(mon, cfg)
		return
	}

	// Initialize the dashboard UI
	useCompact := *compact || cfg.Dashboard.Compact
	dashboard := ui.New(mon, useCompact)
	dashboard.HideCursor()
	defer dashboard.ShowCursor()

	// Graceful shutdown on signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for initial probes to complete
	time.Sleep(2 * time.Second)

	// Main render loop
	ticker := time.NewTicker(cfg.Dashboard.RefreshRate)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			dashboard.ShowCursor()
			fmt.Println("\n\n  Shutting down gracefully...")
			printFinalReport(mon)
			return
		case <-ticker.C:
			dashboard.Render()
		}
	}
}

// runOneShotMode performs a single check cycle and reports results.
func runOneShotMode(mon *monitor.Monitor, cfg *config.Config) {
	fmt.Println("pingboard - one-shot mode")
	fmt.Println(strings.Repeat("-", 60))

	// Wait for all endpoints to complete one check
	time.Sleep(time.Duration(len(cfg.Endpoints)) * time.Second)

	stats := mon.GetStats()
	exitCode := 0

	for _, s := range stats {
		icon := "\033[92m\u2713\033[0m"
		if s.CurrentStatus == monitor.StatusDown {
			icon = "\033[91m\u2717\033[0m"
			exitCode = 1
		} else if s.CurrentStatus == monitor.StatusDegraded {
			icon = "\033[93m!\033[0m"
		}

		latStr := "---"
		if s.LastLatency > 0 {
			latStr = s.LastLatency.Round(time.Millisecond).String()
		}

		fmt.Printf("  %s %-30s %s (%s)\n", icon, s.Name, s.CurrentStatus, latStr)
		if s.LastError != "" {
			fmt.Printf("    \033[91m\u2514\u2500 %s\033[0m\n", s.LastError)
		}
	}

	fmt.Println(strings.Repeat("-", 60))
	os.Exit(exitCode)
}

// printFinalReport displays a summary when the program exits.
func printFinalReport(mon *monitor.Monitor) {
	stats := mon.GetStats()

	fmt.Println("\n  Final Report:")
	fmt.Println("  " + strings.Repeat("\u2500", 50))

	for _, s := range stats {
		uptimeStr := "N/A"
		if s.TotalChecks > 0 {
			uptimeStr = fmt.Sprintf("%.1f%%", s.UptimePercent)
		}

		avgStr := "---"
		if s.AvgLatency > 0 {
			avgStr = s.AvgLatency.Round(time.Millisecond).String()
		}

		fmt.Printf("  %-25s Uptime: %s  Avg: %s  Checks: %d\n",
			s.Name, uptimeStr, avgStr, s.TotalChecks)
	}
	fmt.Println()
}
