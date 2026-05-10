// Package ui renders a real-time terminal dashboard using ANSI escape codes.
package ui

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/Lappy000/pingboard/internal/monitor"
)

// ANSI color codes for terminal rendering.
const (
	reset      = "\033[0m"
	bold       = "\033[1m"
	dim        = "\033[2m"
	underline  = "\033[4m"

	fgBlack   = "\033[30m"
	fgRed     = "\033[31m"
	fgGreen   = "\033[32m"
	fgYellow  = "\033[33m"
	fgBlue    = "\033[34m"
	fgMagenta = "\033[35m"
	fgCyan    = "\033[36m"
	fgWhite   = "\033[37m"

	bgBlack   = "\033[40m"
	bgRed     = "\033[41m"
	bgGreen   = "\033[42m"
	bgYellow  = "\033[43m"
	bgBlue    = "\033[44m"

	fgBrightGreen  = "\033[92m"
	fgBrightRed    = "\033[91m"
	fgBrightYellow = "\033[93m"
	fgBrightCyan   = "\033[96m"
	fgGray         = "\033[90m"
)

// Box-drawing characters for the table.
const (
	topLeft     = "\u250C"
	topRight    = "\u2510"
	bottomLeft  = "\u2514"
	bottomRight = "\u2518"
	horizontal  = "\u2500"
	vertical    = "\u2502"
	teeDown     = "\u252C"
	teeUp       = "\u2534"
	teeRight    = "\u251C"
	teeLeft     = "\u2524"
	cross       = "\u253C"
)

// Dashboard renders endpoint monitoring data to the terminal.
type Dashboard struct {
	mon       *monitor.Monitor
	compact   bool
	startTime time.Time
}

// New creates a new Dashboard renderer.
func New(mon *monitor.Monitor, compact bool) *Dashboard {
	return &Dashboard{
		mon:       mon,
		compact:   compact,
		startTime: time.Now(),
	}
}

// Render clears the screen and draws the current dashboard state.
func (d *Dashboard) Render() {
	d.clearScreen()
	d.moveCursor(1, 1)

	stats := d.mon.GetStats()

	d.renderHeader()
	d.renderSummaryBar(stats)
	fmt.Println()
	d.renderTable(stats)
	fmt.Println()
	d.renderSparklines(stats)
	d.renderFooter()
}

// clearScreen sends the ANSI escape to clear the terminal.
func (d *Dashboard) clearScreen() {
	fmt.Fprint(os.Stdout, "\033[2J")
}

// moveCursor positions the cursor at row, col.
func (d *Dashboard) moveCursor(row, col int) {
	fmt.Fprintf(os.Stdout, "\033[%d;%dH", row, col)
}

// hideCursor hides the terminal cursor.
func (d *Dashboard) HideCursor() {
	fmt.Fprint(os.Stdout, "\033[?25l")
}

// showCursor restores the terminal cursor.
func (d *Dashboard) ShowCursor() {
	fmt.Fprint(os.Stdout, "\033[?25h")
}

// renderHeader displays the application title and branding.
func (d *Dashboard) renderHeader() {
	banner := []string{
		fmt.Sprintf("%s%s", bold, fgBrightCyan),
		"  ____  _             ____                      _ ",
		" |  _ \\(_)_ __   __ | __ )  ___   __ _ _ __ __| |",
		" | |_) | | '_ \\ / _`|  _ \\ / _ \\ / _` | '__/ _` |",
		" |  __/| | | | | (_| | |_) | (_) | (_| | | | (_| |",
		" |_|   |_|_| |_|\\__, |____/ \\___/ \\__,_|_|  \\__,_|",
		"                |___/                               ",
		reset,
	}

	for _, line := range banner {
		fmt.Println(line)
	}
	fmt.Printf("  %s%sReal-time HTTP Endpoint Monitor%s\n\n", dim, fgWhite, reset)
}

// renderSummaryBar shows aggregate counts at the top.
func (d *Dashboard) renderSummaryBar(stats []*monitor.EndpointStats) {
	up, down, degraded, unknown := 0, 0, 0, 0
	for _, s := range stats {
		switch s.CurrentStatus {
		case monitor.StatusUp:
			up++
		case monitor.StatusDown:
			down++
		case monitor.StatusDegraded:
			degraded++
		default:
			unknown++
		}
	}

	uptime := formatDuration(time.Since(d.startTime))

	fmt.Printf("  %s%s ENDPOINTS: %d %s", bold, fgWhite, len(stats), reset)
	fmt.Printf("  %s%s%s UP: %d %s", bold, bgGreen, fgBlack, up, reset)
	if down > 0 {
		fmt.Printf("  %s%s%s DOWN: %d %s", bold, bgRed, fgWhite, down, reset)
	} else {
		fmt.Printf("  %s%s DOWN: %d %s", dim, fgGray, down, reset)
	}
	if degraded > 0 {
		fmt.Printf("  %s%s%s SLOW: %d %s", bold, bgYellow, fgBlack, degraded, reset)
	}
	if unknown > 0 {
		fmt.Printf("  %s%s PENDING: %d %s", dim, fgGray, unknown, reset)
	}
	fmt.Printf("  %s%s SESSION: %s%s\n", dim, fgGray, uptime, reset)
}

// renderTable draws the main status table with box-drawing characters.
func (d *Dashboard) renderTable(stats []*monitor.EndpointStats) {
	// Column widths
	nameW := 22
	statusW := 10
	latencyW := 12
	avgW := 12
	p95W := 12
	uptimeW := 10
	checksW := 10
	lastW := 14

	// Find longest name
	for _, s := range stats {
		if len(s.Name) > nameW-2 {
			nameW = len(s.Name) + 2
		}
	}

	totalW := nameW + statusW + latencyW + avgW + p95W + uptimeW + checksW + lastW + 9

	// Top border
	fmt.Printf("  %s%s%s%s\n", topLeft, strings.Repeat(horizontal, totalW-2), topRight, reset)

	// Header row
	fmt.Printf("  %s %s%-*s%s", vertical, bold+fgBrightCyan, nameW, "ENDPOINT", reset)
	fmt.Printf("%s%-*s%s", bold+fgBrightCyan, statusW, "STATUS", reset)
	fmt.Printf("%s%-*s%s", bold+fgBrightCyan, latencyW, "LATENCY", reset)
	fmt.Printf("%s%-*s%s", bold+fgBrightCyan, avgW, "AVG", reset)
	fmt.Printf("%s%-*s%s", bold+fgBrightCyan, p95W, "P95", reset)
	fmt.Printf("%s%-*s%s", bold+fgBrightCyan, uptimeW, "UPTIME", reset)
	fmt.Printf("%s%-*s%s", bold+fgBrightCyan, checksW, "CHECKS", reset)
	fmt.Printf("%s%-*s%s", bold+fgBrightCyan, lastW, "LAST CHECK", reset)
	fmt.Printf(" %s\n", vertical)

	// Separator
	fmt.Printf("  %s%s%s%s\n", teeRight, strings.Repeat(horizontal, totalW-2), teeLeft, reset)

	// Data rows
	for _, s := range stats {
		d.renderTableRow(s, nameW, statusW, latencyW, avgW, p95W, uptimeW, checksW, lastW)
	}

	// Bottom border
	fmt.Printf("  %s%s%s%s\n", bottomLeft, strings.Repeat(horizontal, totalW-2), bottomRight, reset)
}

// renderTableRow draws a single data row in the table.
func (d *Dashboard) renderTableRow(s *monitor.EndpointStats, nameW, statusW, latencyW, avgW, p95W, uptimeW, checksW, lastW int) {
	// Status indicator with color
	var statusStr string
	switch s.CurrentStatus {
	case monitor.StatusUp:
		statusStr = fmt.Sprintf("%s%s● UP    %s", bold, fgBrightGreen, reset)
	case monitor.StatusDown:
		statusStr = fmt.Sprintf("%s%s● DOWN  %s", bold, fgBrightRed, reset)
	case monitor.StatusDegraded:
		statusStr = fmt.Sprintf("%s%s● SLOW  %s", bold, fgBrightYellow, reset)
	default:
		statusStr = fmt.Sprintf("%s%s○ ---   %s", dim, fgGray, reset)
	}

	// Latency with color coding
	latencyStr := formatLatency(s.LastLatency)
	avgStr := formatLatency(s.AvgLatency)
	p95Str := formatLatency(s.P95Latency)

	// Uptime percentage with color
	var uptimeStr string
	if s.TotalChecks == 0 {
		uptimeStr = fmt.Sprintf("%s%s  ---%% %s", dim, fgGray, reset)
	} else if s.UptimePercent >= 99.9 {
		uptimeStr = fmt.Sprintf("%s%s%6.1f%%%s", bold, fgBrightGreen, s.UptimePercent, reset)
	} else if s.UptimePercent >= 95.0 {
		uptimeStr = fmt.Sprintf("%s%s%6.1f%%%s", bold, fgYellow, s.UptimePercent, reset)
	} else {
		uptimeStr = fmt.Sprintf("%s%s%6.1f%%%s", bold, fgBrightRed, s.UptimePercent, reset)
	}

	// Last checked time
	var lastStr string
	if s.LastChecked.IsZero() {
		lastStr = fmt.Sprintf("%s%snever%s", dim, fgGray, reset)
	} else {
		ago := time.Since(s.LastChecked)
		lastStr = fmt.Sprintf("%s%s%s ago%s", dim, fgGray, formatDurationShort(ago), reset)
	}

	// Checks count
	checksStr := fmt.Sprintf("%d", s.TotalChecks)

	// Name with possible truncation
	name := s.Name
	if len(name) > nameW-2 {
		name = name[:nameW-5] + "..."
	}

	fmt.Printf("  %s %s%-*s%s", vertical, fgWhite+bold, nameW, name, reset)
	fmt.Printf("%-*s", statusW, statusStr)
	fmt.Printf("%-*s", latencyW, latencyStr)
	fmt.Printf("%-*s", avgW, avgStr)
	fmt.Printf("%-*s", p95W, p95Str)
	fmt.Printf("%-*s", uptimeW, uptimeStr)
	fmt.Printf("%s%-*s%s", fgGray, checksW, checksStr, reset)
	fmt.Printf("%-*s", lastW, lastStr)
	fmt.Printf(" %s\n", vertical)
}

// renderSparklines draws mini latency graphs for each endpoint.
func (d *Dashboard) renderSparklines(stats []*monitor.EndpointStats) {
	sparkChars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	fmt.Printf("  %s%sLatency Sparklines (last 30 checks):%s\n", bold, fgCyan, reset)
	fmt.Println()

	for _, s := range stats {
		if len(s.History) == 0 {
			continue
		}

		// Get last 30 latencies
		history := s.History
		if len(history) > 30 {
			history = history[len(history)-30:]
		}

		var maxLat time.Duration
		latencies := make([]time.Duration, 0, len(history))
		for _, h := range history {
			lat := h.Latency
			if lat <= 0 {
				lat = 0
			}
			latencies = append(latencies, lat)
			if lat > maxLat {
				maxLat = lat
			}
		}

		// Build sparkline
		var spark strings.Builder
		for _, lat := range latencies {
			if maxLat == 0 {
				spark.WriteRune(sparkChars[0])
				continue
			}
			normalized := float64(lat) / float64(maxLat)
			idx := int(normalized * float64(len(sparkChars)-1))
			if idx >= len(sparkChars) {
				idx = len(sparkChars) - 1
			}

			// Color based on latency relative to average
			if s.CurrentStatus == monitor.StatusDown {
				spark.WriteString(fgBrightRed)
			} else if normalized > 0.8 {
				spark.WriteString(fgBrightYellow)
			} else {
				spark.WriteString(fgBrightGreen)
			}
			spark.WriteRune(sparkChars[idx])
			spark.WriteString(reset)
		}

		nameLabel := fmt.Sprintf("%-18s", s.Name)
		if len(s.Name) > 18 {
			nameLabel = s.Name[:15] + "..."
		}

		fmt.Printf("  %s%s%s%s %s %s%s(max: %s)%s\n",
			bold, fgWhite, nameLabel, reset,
			spark.String(),
			dim, fgGray, formatLatencyRaw(maxLat), reset,
		)
	}
}

// renderFooter displays help hints and session info.
func (d *Dashboard) renderFooter() {
	fmt.Println()
	fmt.Printf("  %s%sPress Ctrl+C to exit%s", dim, fgGray, reset)
	fmt.Printf("  %s%s|%s", dim, fgGray, reset)
	fmt.Printf("  %s%sPowered by pingboard%s\n", dim, fgGray, reset)
}

// formatLatency formats a duration with color coding.
func formatLatency(d time.Duration) string {
	if d == 0 || d == time.Duration(math.MaxInt64) {
		return fmt.Sprintf("%s%s   ---%s", dim, fgGray, reset)
	}

	ms := float64(d.Microseconds()) / 1000.0

	var color string
	switch {
	case ms < 100:
		color = fgBrightGreen
	case ms < 300:
		color = fgGreen
	case ms < 500:
		color = fgYellow
	case ms < 1000:
		color = fgBrightYellow
	default:
		color = fgBrightRed
	}

	if ms < 1000 {
		return fmt.Sprintf("%s%s%6.0fms%s", bold, color, ms, reset)
	}
	return fmt.Sprintf("%s%s%5.1fs %s", bold, color, ms/1000.0, reset)
}

// formatLatencyRaw formats a duration without color.
func formatLatencyRaw(d time.Duration) string {
	if d == 0 {
		return "---"
	}
	ms := float64(d.Microseconds()) / 1000.0
	if ms < 1000 {
		return fmt.Sprintf("%.0fms", ms)
	}
	return fmt.Sprintf("%.1fs", ms/1000.0)
}

// formatDuration formats an elapsed duration as human-readable (e.g., "2h 15m").
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}

// formatDurationShort formats duration as a compact string.
func formatDurationShort(d time.Duration) string {
	if d < time.Second {
		return "<1s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
}
