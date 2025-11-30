// Package output provides console output for v2 performance testing.
package output

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/wesleyorama2/lunge/internal/performance/v2/engine"
	"github.com/wesleyorama2/lunge/internal/performance/v2/metrics"
)

// ANSI escape codes for cursor control and colors
const (
	// Cursor control
	cursorUp       = "\033[%dA"  // Move cursor up N lines
	cursorDown     = "\033[%dB"  // Move cursor down N lines
	cursorToColumn = "\033[%dG"  // Move cursor to column N
	clearLine      = "\033[2K"   // Clear entire line
	clearToEnd     = "\033[K"    // Clear from cursor to end of line
	saveCursor     = "\033[s"    // Save cursor position
	restoreCursor  = "\033[u"    // Restore cursor position
	hideCursor     = "\033[?25l" // Hide cursor
	showCursor     = "\033[?25h" // Show cursor

	// Colors
	colorReset   = "\033[0m"
	colorBold    = "\033[1m"
	colorDim     = "\033[2m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorRed     = "\033[31m"

	// Box drawing characters
	boxHorizontal  = "━"
	boxVertical    = "│"
	boxTopLeft     = "┌"
	boxTopRight    = "┐"
	boxBottomLeft  = "└"
	boxBottomRight = "┘"

	// Progress bar characters
	progressFilled = "█"
	progressEmpty  = "░"
)

// LiveStats contains real-time statistics for display.
type LiveStats struct {
	// Progress tracking
	Progress  float64       // 0.0 to 1.0
	Elapsed   time.Duration // Time elapsed since test start
	Remaining time.Duration // Estimated time remaining

	// VU stats
	ActiveVUs int // Current active virtual users
	TargetVUs int // Target virtual users

	// Request stats
	CurrentRPS    float64 // Current requests per second
	TotalRequests int64   // Total requests completed
	Errors        int64   // Total errors
	ErrorRate     float64 // Error rate (0.0 to 1.0)

	// Latency stats
	LatencyP95 time.Duration // P95 latency
	LatencyAvg time.Duration // Average latency

	// Phase info
	CurrentPhase string // Current test phase name
	CurrentStage int    // Current stage number (1-indexed)
	TotalStages  int    // Total number of stages
}

// ConsoleOutput manages live console output during test execution.
type ConsoleOutput struct {
	testName       string
	executorType   string
	totalDuration  time.Duration
	updateInterval time.Duration
	writer         io.Writer
	isTTY          bool
	useColors      bool
	quiet          bool

	// State
	mu          sync.Mutex
	lastStats   *LiveStats
	linesOutput int // Number of lines in the live display
}

// ConsoleOutputConfig contains configuration for ConsoleOutput.
type ConsoleOutputConfig struct {
	TestName       string
	ExecutorType   string
	TotalDuration  time.Duration
	UpdateInterval time.Duration
	Writer         io.Writer
	Quiet          bool
	ForceColors    bool
	ForceTTY       bool
}

// NewConsoleOutput creates a new console output handler.
func NewConsoleOutput(config ConsoleOutputConfig) *ConsoleOutput {
	if config.Writer == nil {
		config.Writer = os.Stdout
	}
	if config.UpdateInterval == 0 {
		config.UpdateInterval = time.Second
	}

	isTTY := config.ForceTTY || isTerminal(config.Writer)
	useColors := config.ForceColors || (isTTY && supportsColors())

	return &ConsoleOutput{
		testName:       config.TestName,
		executorType:   config.ExecutorType,
		totalDuration:  config.TotalDuration,
		updateInterval: config.UpdateInterval,
		writer:         config.Writer,
		isTTY:          isTTY,
		useColors:      useColors,
		quiet:          config.Quiet,
	}
}

// isTerminal checks if the writer is a terminal.
func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return isTerminalFile(f)
	}
	return false
}

// isTerminalFile checks if a file is a terminal (cross-platform).
func isTerminalFile(f *os.File) bool {
	// Check if it's stdout/stderr
	if f == os.Stdout || f == os.Stderr {
		return checkIsTerminal(f)
	}
	return false
}

// supportsColors checks if the terminal supports colors.
func supportsColors() bool {
	// Check for explicit color disable
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check for explicit color enable
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}

	// Windows: depends on Windows version and terminal
	if runtime.GOOS == "windows" {
		// Modern Windows terminals support ANSI colors
		// Check for Windows Terminal or ConEmu
		if os.Getenv("WT_SESSION") != "" || os.Getenv("ConEmuANSI") == "ON" {
			return true
		}
		// Recent Windows 10 and later support ANSI
		return true
	}

	// Unix: most terminals support colors
	term := os.Getenv("TERM")
	if term == "" || term == "dumb" {
		return false
	}
	return true
}

// PrintHeader prints the test header.
func (c *ConsoleOutput) PrintHeader() {
	if c.quiet {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	line := strings.Repeat(boxHorizontal, 56)
	status := "Running"
	executorInfo := ""
	if c.executorType != "" {
		executorInfo = fmt.Sprintf(" [%s]", c.executorType)
	}

	c.writeln(c.colorize(line, colorCyan))
	c.writeln(c.colorize(fmt.Sprintf("%s - %s%s", c.testName, status, executorInfo), colorBold))
	c.writeln(c.colorize(line, colorCyan))
	c.writeln("")
}

// Update updates the live display with new statistics.
func (c *ConsoleOutput) Update(stats *LiveStats) {
	if c.quiet || !c.isTTY {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastStats = stats

	// Clear previous output
	if c.linesOutput > 0 {
		c.write(fmt.Sprintf(cursorUp, c.linesOutput))
		for i := 0; i < c.linesOutput; i++ {
			c.write(clearLine)
			if i < c.linesOutput-1 {
				c.write("\n")
			}
		}
		c.write(fmt.Sprintf(cursorUp, c.linesOutput))
	}

	// Render progress section
	lines := c.renderLiveStats(stats)
	c.linesOutput = len(lines)

	for _, line := range lines {
		c.writeln(line)
	}
}

// renderLiveStats renders the live statistics display.
func (c *ConsoleOutput) renderLiveStats(stats *LiveStats) []string {
	var lines []string

	// Progress bar
	progressBar := c.renderProgressBar(stats.Progress, 40)
	progressPercent := fmt.Sprintf("%.0f%%", stats.Progress*100)
	timeInfo := fmt.Sprintf("%s / %s", formatDuration(stats.Elapsed), formatDuration(stats.Elapsed+stats.Remaining))

	lines = append(lines, fmt.Sprintf("Progress: %s %s | %s",
		c.colorize(progressBar, colorGreen),
		c.colorize(progressPercent, colorBold),
		c.colorize(timeInfo, colorDim)))

	// Stage info
	phaseInfo := stats.CurrentPhase
	if stats.TotalStages > 0 {
		phaseInfo = fmt.Sprintf("%s (%d/%d)", stats.CurrentPhase, stats.CurrentStage, stats.TotalStages)
	}
	lines = append(lines, fmt.Sprintf("Stage:    %s", c.colorize(phaseInfo, colorMagenta)))
	lines = append(lines, "")

	// Stats box
	boxWidth := 55

	// Top border
	lines = append(lines, c.colorize(boxTopLeft+strings.Repeat(boxHorizontal, boxWidth-2)+boxTopRight, colorDim))

	// VUs and Requests row
	vusStr := fmt.Sprintf("VUs:     %s / %s",
		c.colorize(fmt.Sprintf("%d", stats.ActiveVUs), colorCyan),
		fmt.Sprintf("%d", stats.TargetVUs))
	reqsStr := fmt.Sprintf("Requests:    %s", c.colorize(formatNumber(stats.TotalRequests), colorCyan))
	lines = append(lines, c.formatBoxRow(vusStr, reqsStr, boxWidth))

	// RPS and Errors row
	rpsStr := fmt.Sprintf("RPS:     %s", c.colorize(fmt.Sprintf("%.1f", stats.CurrentRPS), colorGreen))
	errColor := colorGreen
	if stats.ErrorRate > 0.01 {
		errColor = colorYellow
	}
	if stats.ErrorRate > 0.05 {
		errColor = colorRed
	}
	errStr := fmt.Sprintf("Errors:      %s (%s)",
		c.colorize(fmt.Sprintf("%d", stats.Errors), errColor),
		c.colorize(fmt.Sprintf("%.1f%%", stats.ErrorRate*100), errColor))
	lines = append(lines, c.formatBoxRow(rpsStr, errStr, boxWidth))

	// Latency row
	p95Str := fmt.Sprintf("P95:     %s", c.colorize(formatDurationShort(stats.LatencyP95), colorBlue))
	avgStr := fmt.Sprintf("Avg:         %s", c.colorize(formatDurationShort(stats.LatencyAvg), colorBlue))
	lines = append(lines, c.formatBoxRow(p95Str, avgStr, boxWidth))

	// Bottom border
	lines = append(lines, c.colorize(boxBottomLeft+strings.Repeat(boxHorizontal, boxWidth-2)+boxBottomRight, colorDim))

	return lines
}

// formatBoxRow formats a row inside the stats box with two columns.
func (c *ConsoleOutput) formatBoxRow(left, right string, boxWidth int) string {
	// Account for ANSI codes when calculating padding
	leftVisible := stripANSI(left)
	rightVisible := stripANSI(right)

	// Each column gets roughly half the box width
	colWidth := (boxWidth - 4) / 2 // 4 = 2 borders + 2 padding

	leftPadding := colWidth - len(leftVisible)
	if leftPadding < 0 {
		leftPadding = 0
	}

	rightPadding := colWidth - len(rightVisible)
	if rightPadding < 0 {
		rightPadding = 0
	}

	return fmt.Sprintf("%s %s%s%s %s%s %s",
		c.colorize(boxVertical, colorDim),
		left, strings.Repeat(" ", leftPadding),
		c.colorize(boxVertical, colorDim),
		right, strings.Repeat(" ", rightPadding),
		c.colorize(boxVertical, colorDim))
}

// renderProgressBar renders a progress bar.
func (c *ConsoleOutput) renderProgressBar(progress float64, width int) string {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	filled := int(progress * float64(width))
	empty := width - filled

	return "[" + strings.Repeat(progressFilled, filled) + strings.Repeat(progressEmpty, empty) + "]"
}

// PrintSummary prints the final test summary.
func (c *ConsoleOutput) PrintSummary(result *engine.TestResult) {
	if c.quiet {
		// In quiet mode, just print passed/failed status
		if result.Passed {
			c.writeln(c.colorize("PASSED", colorGreen))
		} else {
			c.writeln(c.colorize("FAILED", colorRed))
		}
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear live output if we were in TTY mode
	if c.isTTY && c.linesOutput > 0 {
		c.write(fmt.Sprintf(cursorUp, c.linesOutput))
		for i := 0; i < c.linesOutput; i++ {
			c.write(clearLine + "\n")
		}
		c.write(fmt.Sprintf(cursorUp, c.linesOutput))
		c.linesOutput = 0
	}

	line := strings.Repeat(boxHorizontal, 56)
	status := "Completed ✓"
	statusColor := colorGreen
	if !result.Passed {
		status = "Failed ✗"
		statusColor = colorRed
	}

	c.writeln("")
	c.writeln(c.colorize(line, colorCyan))
	c.writeln(fmt.Sprintf("%s - %s",
		c.colorize(result.Name, colorBold),
		c.colorize(status, statusColor)))
	c.writeln(c.colorize(line, colorCyan))
	c.writeln("")

	// Duration and request summary
	c.writeln(fmt.Sprintf("Duration:      %s", c.colorize(formatDuration(result.Duration), colorCyan)))
	if result.Metrics != nil {
		c.writeln(fmt.Sprintf("Total Reqs:    %s", c.colorize(formatNumber(result.Metrics.TotalRequests), colorCyan)))

		successRate := 1.0 - result.Metrics.ErrorRate
		successColor := colorGreen
		if successRate < 0.99 {
			successColor = colorYellow
		}
		if successRate < 0.95 {
			successColor = colorRed
		}
		c.writeln(fmt.Sprintf("Success Rate:  %s", c.colorize(fmt.Sprintf("%.1f%%", successRate*100), successColor)))
	}
	c.writeln("")

	// Latency Distribution
	if result.Metrics != nil {
		c.writeln(c.colorize("Latency Distribution:", colorBold))
		c.writeln(fmt.Sprintf("  Min:       %s", formatDurationShort(result.Metrics.Latency.Min)))
		c.writeln(fmt.Sprintf("  P50:       %s", formatDurationShort(result.Metrics.Latency.P50)))
		c.writeln(fmt.Sprintf("  P90:       %s", formatDurationShort(result.Metrics.Latency.P90)))
		c.writeln(fmt.Sprintf("  P95:       %s", formatDurationShort(result.Metrics.Latency.P95)))
		c.writeln(fmt.Sprintf("  P99:       %s", formatDurationShort(result.Metrics.Latency.P99)))
		c.writeln(fmt.Sprintf("  Max:       %s", formatDurationShort(result.Metrics.Latency.Max)))
		c.writeln("")
	}

	// Thresholds
	if len(result.Thresholds) > 0 {
		c.writeln(c.colorize("Thresholds:", colorBold))
		for _, t := range result.Thresholds {
			status := c.colorize("✓", colorGreen)
			if !t.Passed {
				status = c.colorize("✗", colorRed)
			}
			c.writeln(fmt.Sprintf("  %s %s %s (actual: %s)", status, t.Metric, t.Expression, t.Value))
		}
		c.writeln("")
	}
}

// PrintNonInteractiveUpdate prints a non-interactive status update.
// Used when output is not a TTY (e.g., piped to a file or CI/CD).
func (c *ConsoleOutput) PrintNonInteractiveUpdate(stats *LiveStats) {
	if c.quiet {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple one-line status for non-TTY
	c.writeln(fmt.Sprintf("[%s] Progress: %.0f%% | VUs: %d | Reqs: %d | RPS: %.1f | Errors: %d (%.1f%%) | P95: %s",
		formatDuration(stats.Elapsed),
		stats.Progress*100,
		stats.ActiveVUs,
		stats.TotalRequests,
		stats.CurrentRPS,
		stats.Errors,
		stats.ErrorRate*100,
		formatDurationShort(stats.LatencyP95)))
}

// IsTTY returns whether the output is a terminal.
func (c *ConsoleOutput) IsTTY() bool {
	return c.isTTY
}

// write writes to the output without a newline.
func (c *ConsoleOutput) write(s string) {
	fmt.Fprint(c.writer, s)
}

// writeln writes to the output with a newline.
func (c *ConsoleOutput) writeln(s string) {
	fmt.Fprintln(c.writer, s)
}

// colorize wraps text in color codes if colors are enabled.
func (c *ConsoleOutput) colorize(text, color string) string {
	if !c.useColors {
		return text
	}
	return color + text + colorReset
}

// Helper functions

// formatDuration formats a duration in a human-readable format.
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %02ds", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dh %02dm %02ds", h, m, s)
}

// formatDurationShort formats a duration in a short format.
func formatDurationShort(d time.Duration) string {
	if d < time.Microsecond {
		return "0ms"
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

// formatNumber formats a number with thousands separators.
func formatNumber(n int64) string {
	str := fmt.Sprintf("%d", n)
	if len(str) <= 3 {
		return str
	}

	// Add thousands separators
	var result strings.Builder
	offset := len(str) % 3
	if offset > 0 {
		result.WriteString(str[:offset])
	}
	for i := offset; i < len(str); i += 3 {
		if result.Len() > 0 {
			result.WriteString(",")
		}
		result.WriteString(str[i : i+3])
	}
	return result.String()
}

// stripANSI removes ANSI escape codes from a string.
func stripANSI(s string) string {
	// Simple state machine to strip ANSI sequences
	var result strings.Builder
	inEscape := false

	for i := 0; i < len(s); i++ {
		if s[i] == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteByte(s[i])
	}

	return result.String()
}

// StatsFromMetrics creates LiveStats from engine metrics.
func StatsFromMetrics(
	metricsSnapshot *metrics.Snapshot,
	progress float64,
	totalDuration time.Duration,
	targetVUs int,
	currentStage, totalStages int,
) *LiveStats {
	if metricsSnapshot == nil {
		return &LiveStats{
			Progress:     progress,
			TargetVUs:    targetVUs,
			CurrentStage: currentStage,
			TotalStages:  totalStages,
			CurrentPhase: "initializing",
		}
	}

	elapsed := metricsSnapshot.Elapsed
	remaining := time.Duration(0)
	if progress > 0 && progress < 1 {
		remaining = time.Duration(float64(elapsed) * (1 - progress) / progress)
	} else if totalDuration > 0 {
		remaining = totalDuration - elapsed
		if remaining < 0 {
			remaining = 0
		}
	}

	return &LiveStats{
		Progress:      progress,
		Elapsed:       elapsed,
		Remaining:     remaining,
		ActiveVUs:     metricsSnapshot.ActiveVUs,
		TargetVUs:     targetVUs,
		CurrentRPS:    metricsSnapshot.RPS,
		TotalRequests: metricsSnapshot.TotalRequests,
		Errors:        metricsSnapshot.FailedRequests,
		ErrorRate:     metricsSnapshot.ErrorRate,
		LatencyP95:    metricsSnapshot.Latency.P95,
		LatencyAvg:    metricsSnapshot.Latency.Mean,
		CurrentPhase:  string(metricsSnapshot.CurrentPhase),
		CurrentStage:  currentStage,
		TotalStages:   totalStages,
	}
}
