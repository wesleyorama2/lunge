// Package report provides HTML report generation for v2 performance test results.
package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"time"

	"github.com/wesleyorama2/lunge/internal/performance/v2/engine"
	"github.com/wesleyorama2/lunge/internal/performance/v2/metrics"
)

// ReportData contains all data needed to render the HTML report.
type ReportData struct {
	*engine.TestResult
	TimeSeriesJSON template.JS
}

// TimeSeriesPoint represents a single point in the time series for JSON export.
type TimeSeriesPoint struct {
	Timestamp         string  `json:"timestamp"`
	TotalRequests     int64   `json:"totalRequests"`
	TotalSuccesses    int64   `json:"totalSuccesses"`
	TotalFailures     int64   `json:"totalFailures"`
	TotalBytes        int64   `json:"totalBytes"`
	IntervalRequests  int64   `json:"intervalRequests"`
	IntervalRPS       float64 `json:"intervalRPS"`
	LatencyMin        int64   `json:"latencyMin"`
	LatencyMax        int64   `json:"latencyMax"`
	LatencyP50        int64   `json:"latencyP50"`
	LatencyP90        int64   `json:"latencyP90"`
	LatencyP95        int64   `json:"latencyP95"`
	LatencyP99        int64   `json:"latencyP99"`
	ActiveVUs         int     `json:"activeVUs"`
	Phase             string  `json:"phase"`
	IntervalErrorRate float64 `json:"intervalErrorRate"`
}

// GenerateHTML generates an HTML report from test results and writes it to a file.
func GenerateHTML(result *engine.TestResult, outputPath string) error {
	html, err := GenerateHTMLString(result)
	if err != nil {
		return fmt.Errorf("failed to generate HTML: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(html), 0644); err != nil {
		return fmt.Errorf("failed to write HTML file: %w", err)
	}

	return nil
}

// GenerateHTMLString generates an HTML report from test results and returns it as a string.
func GenerateHTMLString(result *engine.TestResult) (string, error) {
	if result == nil {
		return "", fmt.Errorf("result cannot be nil")
	}

	// Create template with helper functions
	tmpl, err := template.New("report").Funcs(templateFuncs()).Parse(htmlTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Convert time series to JSON for charts
	timeSeriesJSON, err := convertTimeSeriesJSON(result.TimeSeries)
	if err != nil {
		return "", fmt.Errorf("failed to convert time series: %w", err)
	}

	// Prepare report data
	data := ReportData{
		TestResult:     result,
		TimeSeriesJSON: template.JS(timeSeriesJSON),
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// convertTimeSeriesJSON converts the time series buckets to JSON for chart rendering.
func convertTimeSeriesJSON(timeSeries []*metrics.TimeBucket) (string, error) {
	if len(timeSeries) == 0 {
		return "[]", nil
	}

	points := make([]TimeSeriesPoint, len(timeSeries))
	for i, bucket := range timeSeries {
		points[i] = TimeSeriesPoint{
			Timestamp:         bucket.Timestamp.Format(time.RFC3339),
			TotalRequests:     bucket.TotalRequests,
			TotalSuccesses:    bucket.TotalSuccesses,
			TotalFailures:     bucket.TotalFailures,
			TotalBytes:        bucket.TotalBytes,
			IntervalRequests:  bucket.IntervalRequests,
			IntervalRPS:       bucket.IntervalRPS,
			LatencyMin:        int64(bucket.LatencyMin),
			LatencyMax:        int64(bucket.LatencyMax),
			LatencyP50:        int64(bucket.LatencyP50),
			LatencyP90:        int64(bucket.LatencyP90),
			LatencyP95:        int64(bucket.LatencyP95),
			LatencyP99:        int64(bucket.LatencyP99),
			ActiveVUs:         bucket.ActiveVUs,
			Phase:             string(bucket.Phase),
			IntervalErrorRate: bucket.IntervalErrorRate,
		}
	}

	jsonBytes, err := json.Marshal(points)
	if err != nil {
		return "[]", err
	}

	return string(jsonBytes), nil
}

// templateFuncs returns the template helper functions.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatDuration":  formatDuration,
		"formatNumber":    formatNumber,
		"formatLatency":   formatLatency,
		"formatBytes":     formatBytes,
		"mul":             mul,
		"successRate":     successRate,
		"hasRequestStats": hasRequestStats,
	}
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		if secs == 0 {
			return fmt.Sprintf("%dm", mins)
		}
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}

// formatNumber formats a large number with commas.
func formatNumber(n int64) string {
	if n < 0 {
		return "-" + formatNumber(-n)
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}

	// Format with thousands separators
	str := fmt.Sprintf("%d", n)
	result := ""
	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result
}

// formatLatency formats a latency duration in a human-readable way.
func formatLatency(d time.Duration) string {
	if d == 0 {
		return "0"
	}
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		us := float64(d.Microseconds())
		if us < 100 {
			return fmt.Sprintf("%.1fµs", us)
		}
		return fmt.Sprintf("%dµs", int(us))
	}
	if d < time.Second {
		ms := float64(d.Microseconds()) / 1000.0
		if ms < 10 {
			return fmt.Sprintf("%.2fms", ms)
		}
		if ms < 100 {
			return fmt.Sprintf("%.1fms", ms)
		}
		return fmt.Sprintf("%dms", int(ms))
	}
	s := d.Seconds()
	if s < 10 {
		return fmt.Sprintf("%.2fs", s)
	}
	return fmt.Sprintf("%.1fs", s)
}

// formatBytes formats bytes in a human-readable way.
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// mul multiplies two float64 values (for template use).
func mul(a, b float64) float64 {
	return a * b
}

// successRate calculates the success rate from a metrics snapshot.
func successRate(m *metrics.Snapshot) float64 {
	if m == nil || m.TotalRequests == 0 {
		return 0
	}
	return float64(m.SuccessRequests) / float64(m.TotalRequests) * 100
}

// hasRequestStats checks if any scenario has request stats.
func hasRequestStats(scenarios map[string]*engine.ScenarioResult) bool {
	for _, s := range scenarios {
		if len(s.RequestStats) > 0 {
			return true
		}
	}
	return false
}
