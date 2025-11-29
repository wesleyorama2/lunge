package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	v2config "github.com/wesleyorama2/lunge/internal/performance/v2/config"
	"github.com/wesleyorama2/lunge/internal/performance/v2/engine"
	"github.com/wesleyorama2/lunge/internal/performance/v2/executor"
	"github.com/wesleyorama2/lunge/internal/performance/v2/output"
	"github.com/wesleyorama2/lunge/internal/performance/v2/report"
)

var perfCmd = &cobra.Command{
	Use:   "perf",
	Short: "Run performance tests from a configuration file",
	Long: `Execute performance and load tests with configurable concurrency, duration, and rate limiting.
Supports various load patterns, real-time monitoring, and comprehensive reporting.

Config file mode:
  lunge perf --config test.yaml

Quick CLI mode (single scenario):
  lunge perf --url https://api.example.com/health \
    --executor ramping-vus \
    --stages "30s:10,2m:10,30s:0" \
    --duration 3m

Arrival rate mode:
  lunge perf --url https://api.example.com/health \
    --executor constant-arrival-rate \
    --rate 100 \
    --duration 5m \
    --max-vus 200`,
	Run: func(cmd *cobra.Command, args []string) {
		runPerfTest(cmd, args)
	},
}

// runPerfTest runs performance tests using the performance engine
func runPerfTest(cmd *cobra.Command, args []string) {
	configFile, _ := cmd.Flags().GetString("config")
	url, _ := cmd.Flags().GetString("url")
	verbose, _ := cmd.Flags().GetBool("verbose")
	outputPath, _ := cmd.Flags().GetString("output")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	htmlOutput, _ := cmd.Flags().GetBool("html")
	quiet, _ := cmd.Flags().GetBool("quiet")

	// Performance flags
	executorType, _ := cmd.Flags().GetString("executor")
	duration, _ := cmd.Flags().GetString("duration")
	vus, _ := cmd.Flags().GetInt("vus")
	stages, _ := cmd.Flags().GetString("stages")
	rate, _ := cmd.Flags().GetFloat64("rate")
	maxVUs, _ := cmd.Flags().GetInt("max-vus")
	preAllocatedVUs, _ := cmd.Flags().GetInt("pre-allocated-vus")

	var testConfig *v2config.TestConfig
	var err error

	if configFile != "" {
		// Load config from file
		testConfig, err = v2config.LoadConfig(configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
	} else if url != "" {
		// Build config from CLI flags
		testConfig, err = buildConfigFromCLI(url, executorType, duration, vus, stages, rate, maxVUs, preAllocatedVUs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building config: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("Error: either --config or --url is required")
		cmd.Help()
		return
	}

	// Calculate total duration from config
	totalDuration := calculateTotalDuration(testConfig)

	// Determine executor type for display
	displayExecutor := executorType
	if displayExecutor == "" {
		for _, scenario := range testConfig.Scenarios {
			displayExecutor = scenario.Executor
			break
		}
	}

	// Create console output handler
	consoleOutput := output.NewConsoleOutput(output.ConsoleOutputConfig{
		TestName:       testConfig.Name,
		ExecutorType:   displayExecutor,
		TotalDuration:  totalDuration,
		UpdateInterval: time.Second,
		Quiet:          quiet,
	})

	// Create and run the engine
	eng, err := engine.NewEngine(testConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating engine: %v\n", err)
		os.Exit(1)
	}

	if verbose && !quiet {
		fmt.Printf("Starting performance test: %s\n", testConfig.Name)
		for name, scenario := range testConfig.Scenarios {
			fmt.Printf("  Scenario: %s (executor: %s)\n", name, scenario.Executor)
		}
		fmt.Println()
	}

	// Print header
	consoleOutput.PrintHeader()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start progress update goroutine
	var wg sync.WaitGroup
	var result *engine.TestResult
	var runErr error

	// Run engine in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, runErr = eng.Run(ctx)
	}()

	// Update progress while engine is running
	updateTicker := time.NewTicker(time.Second)
	defer updateTicker.Stop()

	// Get target VUs from config
	targetVUs := getTargetVUs(testConfig)

progressLoop:
	for {
		select {
		case <-updateTicker.C:
			if eng.IsRunning() {
				metrics := eng.GetMetrics()
				progress := eng.GetProgress()
				scenarioStats := eng.GetScenarioStats()

				// Get current stage info
				currentStage, totalStages := getStageInfo(scenarioStats)

				stats := output.StatsFromMetrics(
					metrics,
					progress,
					totalDuration,
					targetVUs,
					currentStage,
					totalStages,
				)

				if consoleOutput.IsTTY() {
					consoleOutput.Update(stats)
				} else if !quiet {
					consoleOutput.PrintNonInteractiveUpdate(stats)
				}
			} else {
				break progressLoop
			}
		default:
			// Check if the engine has stopped
			if !eng.IsRunning() && result != nil {
				break progressLoop
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Wait for engine to complete
	wg.Wait()

	if runErr != nil {
		fmt.Fprintf(os.Stderr, "Error running test: %v\n", runErr)
		// Continue to output results even on error
	}

	// Print final summary
	consoleOutput.PrintSummary(result)

	// Determine output type based on flags and extension
	outputIsHTML := htmlOutput || (outputPath != "" && strings.HasSuffix(strings.ToLower(outputPath), ".html"))
	outputIsJSON := jsonOutput || (outputPath != "" && strings.HasSuffix(strings.ToLower(outputPath), ".json"))

	// Generate reports based on format
	if outputIsJSON {
		// JSON output only
		outputJSONResult(result, outputPath)
	} else if outputIsHTML {
		// HTML report
		if outputPath != "" {
			if err := outputHTMLReport(result, outputPath, verbose); err != nil {
				fmt.Fprintf(os.Stderr, "Error generating HTML report: %v\n", err)
			}
		} else {
			// Generate default HTML report with timestamp
			defaultOutput := generateDefaultHTMLPath(testConfig.Name)
			if err := outputHTMLReport(result, defaultOutput, verbose); err != nil {
				fmt.Fprintf(os.Stderr, "Error generating HTML report: %v\n", err)
			}
		}
	} else if outputPath != "" {
		// If output path specified without extension, generate both HTML and JSON
		htmlPath := outputPath + ".html"
		jsonPath := outputPath + ".json"
		if err := outputHTMLReport(result, htmlPath, verbose); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating HTML report: %v\n", err)
		}
		outputJSONResult(result, jsonPath)
	}

	// Exit with error code if test failed
	if result != nil && !result.Passed {
		os.Exit(1)
	}
	if runErr != nil {
		os.Exit(1)
	}
}

// calculateTotalDuration calculates the total test duration from config.
func calculateTotalDuration(cfg *v2config.TestConfig) time.Duration {
	var maxDuration time.Duration

	for _, scenario := range cfg.Scenarios {
		var scenarioDuration time.Duration

		if len(scenario.Stages) > 0 {
			// Sum up stage durations
			for _, stage := range scenario.Stages {
				if d, err := time.ParseDuration(stage.Duration); err == nil {
					scenarioDuration += d
				}
			}
		} else if scenario.Duration != "" {
			if d, err := time.ParseDuration(scenario.Duration); err == nil {
				scenarioDuration = d
			}
		}

		if scenarioDuration > maxDuration {
			maxDuration = scenarioDuration
		}
	}

	return maxDuration
}

// getTargetVUs gets the target VU count from config.
func getTargetVUs(cfg *v2config.TestConfig) int {
	maxVUs := 0

	for _, scenario := range cfg.Scenarios {
		if scenario.VUs > maxVUs {
			maxVUs = scenario.VUs
		}
		if scenario.MaxVUs > maxVUs {
			maxVUs = scenario.MaxVUs
		}
		// Check stage targets
		for _, stage := range scenario.Stages {
			if stage.Target > maxVUs {
				maxVUs = stage.Target
			}
		}
	}

	if maxVUs == 0 {
		maxVUs = 10 // Default
	}

	return maxVUs
}

// getStageInfo extracts current stage info from executor stats.
func getStageInfo(stats map[string]*executor.Stats) (current, total int) {
	// For simplicity, aggregate across all scenarios
	// In practice with multiple scenarios, you'd want more sophisticated handling
	for _, s := range stats {
		if s != nil {
			if s.CurrentStage > current {
				current = s.CurrentStage
			}
			if s.TotalStages > total {
				total = s.TotalStages
			}
		}
	}
	return current, total
}

// generateDefaultHTMLPath creates a default HTML report path based on test name
func generateDefaultHTMLPath(testName string) string {
	// Sanitize test name for use in filename
	safeName := strings.ReplaceAll(testName, " ", "-")
	safeName = strings.ReplaceAll(safeName, "/", "-")
	safeName = strings.ToLower(safeName)

	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("perf-report-%s-%s.html", safeName, timestamp)
}

// outputHTMLReport generates and saves an HTML report
func outputHTMLReport(result *engine.TestResult, outputPath string, verbose bool) error {
	if result == nil {
		return fmt.Errorf("no results to report")
	}

	// Ensure output path has .html extension
	if !strings.HasSuffix(strings.ToLower(outputPath), ".html") {
		outputPath = outputPath + ".html"
	}

	// Create output directory if needed
	dir := filepath.Dir(outputPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Generate the HTML report
	if err := report.GenerateHTML(result, outputPath); err != nil {
		return fmt.Errorf("failed to generate HTML report: %w", err)
	}

	if verbose {
		fmt.Printf("HTML report generated: %s\n", outputPath)
	} else {
		fmt.Printf("Report: %s\n", outputPath)
	}

	return nil
}

// buildConfigFromCLI builds a TestConfig from CLI flags
func buildConfigFromCLI(url, executorType, duration string, vus int, stages string, rate float64, maxVUs, preAllocatedVUs int) (*v2config.TestConfig, error) {
	// Default executor type
	if executorType == "" {
		executorType = "constant-vus"
	}

	// Default VUs
	if vus == 0 && executorType == "constant-vus" {
		vus = 10
	}

	// Default duration for non-stage executors
	if duration == "" && stages == "" {
		duration = "30s"
	}

	scenario := &v2config.ScenarioConfig{
		Executor:        executorType,
		VUs:             vus,
		Duration:        duration,
		Rate:            rate,
		MaxVUs:          maxVUs,
		PreAllocatedVUs: preAllocatedVUs,
		Requests: []v2config.RequestConfig{
			{
				Name:   "cli-request",
				Method: "GET",
				URL:    url,
			},
		},
	}

	// Parse stages if provided
	if stages != "" {
		parsedStages, err := parseStages(stages)
		if err != nil {
			return nil, fmt.Errorf("invalid stages format: %w", err)
		}
		scenario.Stages = parsedStages
	}

	config := &v2config.TestConfig{
		Name:        "CLI Test",
		Description: fmt.Sprintf("Test generated from CLI flags for %s", url),
		Scenarios: map[string]*v2config.ScenarioConfig{
			"cli-test": scenario,
		},
	}

	return config, nil
}

// parseStages parses stages from CLI format "30s:10,2m:10,30s:0"
func parseStages(stagesStr string) ([]v2config.StageConfig, error) {
	var stages []v2config.StageConfig

	parts := strings.Split(stagesStr, ",")
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Parse "duration:target" format
		colonIdx := strings.LastIndex(part, ":")
		if colonIdx == -1 {
			return nil, fmt.Errorf("stage %d: expected 'duration:target' format, got '%s'", i+1, part)
		}

		durationStr := part[:colonIdx]
		targetStr := part[colonIdx+1:]

		// Validate duration
		if _, err := time.ParseDuration(durationStr); err != nil {
			return nil, fmt.Errorf("stage %d: invalid duration '%s': %w", i+1, durationStr, err)
		}

		// Parse target
		target, err := strconv.Atoi(targetStr)
		if err != nil {
			return nil, fmt.Errorf("stage %d: invalid target '%s': %w", i+1, targetStr, err)
		}

		stages = append(stages, v2config.StageConfig{
			Duration: durationStr,
			Target:   target,
			Name:     fmt.Sprintf("stage-%d", i+1),
		})
	}

	if len(stages) == 0 {
		return nil, fmt.Errorf("at least one stage is required")
	}

	return stages, nil
}

// outputConsoleResult outputs the test result to console
func outputConsoleResult(result *engine.TestResult, verbose bool) {
	if result == nil {
		fmt.Println("No results available")
		return
	}

	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 59))
	fmt.Printf(" Performance Test Results: %s\n", result.Name)
	fmt.Println("=" + strings.Repeat("=", 59))
	fmt.Println()

	// Test summary
	passStatus := "✓ PASSED"
	if !result.Passed {
		passStatus = "✗ FAILED"
	}
	fmt.Printf("Status:    %s\n", passStatus)
	fmt.Printf("Duration:  %s\n", result.Duration.Round(time.Millisecond))
	fmt.Printf("Start:     %s\n", result.StartTime.Format(time.RFC3339))
	fmt.Printf("End:       %s\n", result.EndTime.Format(time.RFC3339))
	fmt.Println()

	// Overall metrics
	if result.Metrics != nil {
		m := result.Metrics
		fmt.Println("─── Overall Metrics " + strings.Repeat("─", 40))
		fmt.Printf("  Total Requests:    %d\n", m.TotalRequests)
		fmt.Printf("  Successful:        %d\n", m.SuccessRequests)
		fmt.Printf("  Failed:            %d\n", m.FailedRequests)
		fmt.Printf("  Error Rate:        %.2f%%\n", m.ErrorRate*100)
		fmt.Printf("  Throughput:        %.2f req/s\n", m.RPS)
		fmt.Printf("  Data Transferred:  %s\n", formatBytes(m.TotalBytes))
		fmt.Println()

		// Latency stats
		fmt.Println("─── Latency " + strings.Repeat("─", 48))
		fmt.Printf("  Min:    %s\n", m.Latency.Min.Round(time.Microsecond))
		fmt.Printf("  Max:    %s\n", m.Latency.Max.Round(time.Microsecond))
		fmt.Printf("  Mean:   %s\n", m.Latency.Mean.Round(time.Microsecond))
		fmt.Printf("  P50:    %s\n", m.Latency.P50.Round(time.Microsecond))
		fmt.Printf("  P90:    %s\n", m.Latency.P90.Round(time.Microsecond))
		fmt.Printf("  P95:    %s\n", m.Latency.P95.Round(time.Microsecond))
		fmt.Printf("  P99:    %s\n", m.Latency.P99.Round(time.Microsecond))
		fmt.Println()
	}

	// Scenario results
	if len(result.Scenarios) > 0 && verbose {
		fmt.Println("─── Scenarios " + strings.Repeat("─", 46))
		for name, scenario := range result.Scenarios {
			fmt.Printf("  %s:\n", name)
			fmt.Printf("    Executor:    %s\n", scenario.Executor)
			fmt.Printf("    Duration:    %s\n", scenario.Duration.Round(time.Millisecond))
			fmt.Printf("    Iterations:  %d\n", scenario.Iterations)
			if scenario.Error != nil {
				fmt.Printf("    Error:       %v\n", scenario.Error)
			}
		}
		fmt.Println()
	}

	// Threshold results
	if len(result.Thresholds) > 0 {
		fmt.Println("─── Thresholds " + strings.Repeat("─", 45))
		for _, t := range result.Thresholds {
			status := "✓"
			if !t.Passed {
				status = "✗"
			}
			fmt.Printf("  %s %s: %s (actual: %s)\n", status, t.Metric, t.Expression, t.Value)
			if t.Message != "" && !t.Passed {
				fmt.Printf("      %s\n", t.Message)
			}
		}
		fmt.Println()
	}

	fmt.Println("=" + strings.Repeat("=", 59))
}

// outputJSONResult outputs the test result as JSON
func outputJSONResult(result *engine.TestResult, outputPath string) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling result: %v\n", err)
		return
	}

	if outputPath != "" {
		if err := os.WriteFile(outputPath, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing result to file: %v\n", err)
			return
		}
		fmt.Printf("Results written to: %s\n", outputPath)
	} else {
		fmt.Println(string(data))
	}
}

// formatBytes formats bytes to human-readable string
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
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

func init() {
	// Performance engine flags
	perfCmd.Flags().String("url", "", "URL to test (alternative to --config)")
	perfCmd.Flags().String("executor", "", "Executor type: constant-vus, ramping-vus, constant-arrival-rate, ramping-arrival-rate")
	perfCmd.Flags().Int("vus", 0, "Number of virtual users")
	perfCmd.Flags().String("stages", "", "Stages in format 'duration:target,duration:target,...' for ramping executors")
	perfCmd.Flags().Float64("rate", 0, "Iterations per second for arrival-rate executors")
	perfCmd.Flags().Int("max-vus", 0, "Maximum VUs for arrival-rate executors")
	perfCmd.Flags().Int("pre-allocated-vus", 0, "Pre-allocated VUs for arrival-rate executors")
	perfCmd.Flags().Bool("json", false, "Output results as JSON")
	perfCmd.Flags().Bool("html", false, "Generate HTML report")
	perfCmd.Flags().BoolP("quiet", "q", false, "Disable live progress output, show only final summary")

	// Basic flags
	perfCmd.Flags().StringP("config", "c", "", "Configuration file")
	perfCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	perfCmd.Flags().DurationP("timeout", "t", 30*time.Second, "Request timeout")
	perfCmd.Flags().String("duration", "", "Test duration (e.g., 5m, 30s)")

	// Reporting flags
	perfCmd.Flags().String("format", "", "Report format (text, json, html)")
	perfCmd.Flags().String("output", "", "Output file for report (default: stdout)")
}
