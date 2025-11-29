package main

import (
	"fmt"
	"os"
	"time"

	"github.com/wesleyorama2/lunge/internal/performance/v2/engine"
	"github.com/wesleyorama2/lunge/internal/performance/v2/metrics"
	"github.com/wesleyorama2/lunge/internal/performance/v2/report"
)

func main() {
	result := createSampleTestResult()

	outputPath := "sample-v2-report.html"
	if len(os.Args) > 1 {
		outputPath = os.Args[1]
	}

	err := report.GenerateHTML(result, outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Sample report generated: %s\n", outputPath)
}

func createSampleTestResult() *engine.TestResult {
	now := time.Now()

	return &engine.TestResult{
		Name:        "API Load Test - v2 Engine",
		Description: "Load testing the user API endpoints with ramping VUs",
		StartTime:   now.Add(-2 * time.Minute),
		EndTime:     now,
		Duration:    2 * time.Minute,
		Passed:      true,
		Metrics: &metrics.Snapshot{
			TotalRequests:   5847,
			SuccessRequests: 5789,
			FailedRequests:  58,
			TotalBytes:      12582912,
			RPS:             48.73,
			SteadyStateRPS:  52.1,
			ErrorRate:       0.0099,
			ActiveVUs:       10,
			Latency: metrics.LatencyStats{
				Min:    8 * time.Millisecond,
				Max:    892 * time.Millisecond,
				Mean:   47 * time.Millisecond,
				StdDev: 38 * time.Millisecond,
				P50:    39 * time.Millisecond,
				P90:    89 * time.Millisecond,
				P95:    124 * time.Millisecond,
				P99:    287 * time.Millisecond,
				Count:  5847,
			},
		},
		TimeSeries: createSampleTimeSeries(120),
		Scenarios: map[string]*engine.ScenarioResult{
			"browse-users": {
				Name:       "browse-users",
				Executor:   "ramping-vus",
				Duration:   2 * time.Minute,
				Iterations: 3521,
				ActiveVUs:  10,
				Metrics: &metrics.Snapshot{
					TotalRequests:   3521,
					SuccessRequests: 3487,
					FailedRequests:  34,
					RPS:             29.34,
					ErrorRate:       0.0097,
					Latency: metrics.LatencyStats{
						Mean: 42 * time.Millisecond,
						P95:  115 * time.Millisecond,
					},
				},
				RequestStats: map[string]engine.RequestStats{
					"GET /api/users": {
						Name:  "GET /api/users",
						Count: 2341,
						Latency: metrics.LatencyStats{
							Min:  10 * time.Millisecond,
							Max:  456 * time.Millisecond,
							Mean: 38 * time.Millisecond,
							P50:  32 * time.Millisecond,
							P95:  98 * time.Millisecond,
							P99:  234 * time.Millisecond,
						},
					},
					"GET /api/users/:id": {
						Name:  "GET /api/users/:id",
						Count: 1180,
						Latency: metrics.LatencyStats{
							Min:  8 * time.Millisecond,
							Max:  678 * time.Millisecond,
							Mean: 51 * time.Millisecond,
							P50:  42 * time.Millisecond,
							P95:  142 * time.Millisecond,
							P99:  312 * time.Millisecond,
						},
					},
				},
			},
			"create-users": {
				Name:       "create-users",
				Executor:   "constant-arrival-rate",
				Duration:   2 * time.Minute,
				Iterations: 2326,
				ActiveVUs:  5,
				Metrics: &metrics.Snapshot{
					TotalRequests:   2326,
					SuccessRequests: 2302,
					FailedRequests:  24,
					RPS:             19.38,
					ErrorRate:       0.0103,
					Latency: metrics.LatencyStats{
						Mean: 54 * time.Millisecond,
						P95:  138 * time.Millisecond,
					},
				},
				RequestStats: map[string]engine.RequestStats{
					"POST /api/users": {
						Name:  "POST /api/users",
						Count: 2326,
						Latency: metrics.LatencyStats{
							Min:  15 * time.Millisecond,
							Max:  892 * time.Millisecond,
							Mean: 54 * time.Millisecond,
							P50:  45 * time.Millisecond,
							P95:  138 * time.Millisecond,
							P99:  287 * time.Millisecond,
						},
					},
				},
			},
		},
		Thresholds: []engine.ThresholdResult{
			{
				Metric:     "http_req_duration",
				Expression: "p95 < 200ms",
				Passed:     true,
				Value:      "124ms",
			},
			{
				Metric:     "http_req_duration",
				Expression: "p99 < 500ms",
				Passed:     true,
				Value:      "287ms",
			},
			{
				Metric:     "http_req_failed",
				Expression: "rate < 0.02",
				Passed:     true,
				Value:      "0.0099",
			},
			{
				Metric:     "http_reqs",
				Expression: "rate > 40",
				Passed:     true,
				Value:      "48.73",
			},
		},
	}
}

func createSampleTimeSeries(seconds int) []*metrics.TimeBucket {
	buckets := make([]*metrics.TimeBucket, seconds)
	baseTime := time.Now().Add(-time.Duration(seconds) * time.Second)

	rampUpEnd := 20
	steadyEnd := seconds - 20

	for i := 0; i < seconds; i++ {
		var phase metrics.Phase
		var vus int
		var rps float64

		if i < rampUpEnd {
			phase = metrics.PhaseRampUp
			progress := float64(i) / float64(rampUpEnd)
			vus = int(progress * 10)
			rps = progress * 50
		} else if i < steadyEnd {
			phase = metrics.PhaseSteady
			vus = 10
			rps = 48 + float64(i%5) - 2
		} else {
			phase = metrics.PhaseRampDown
			progress := float64(seconds-i) / float64(rampUpEnd)
			vus = int(progress * 10)
			rps = progress * 50
		}

		if vus < 1 {
			vus = 1
		}
		if rps < 1 {
			rps = 1
		}

		buckets[i] = &metrics.TimeBucket{
			Timestamp:         baseTime.Add(time.Duration(i) * time.Second),
			TotalRequests:     int64(float64(i) * 48.7),
			TotalSuccesses:    int64(float64(i) * 48.2),
			TotalFailures:     int64(float64(i) * 0.5),
			TotalBytes:        int64(float64(i) * 104857),
			IntervalRequests:  int64(rps),
			IntervalRPS:       rps,
			LatencyMin:        8 * time.Millisecond,
			LatencyMax:        time.Duration(100+i*3) * time.Millisecond,
			LatencyP50:        time.Duration(35+i%10) * time.Millisecond,
			LatencyP90:        time.Duration(80+i%20) * time.Millisecond,
			LatencyP95:        time.Duration(110+i%30) * time.Millisecond,
			LatencyP99:        time.Duration(250+i%50) * time.Millisecond,
			ActiveVUs:         vus,
			Phase:             phase,
			IntervalErrorRate: 0.01 + float64(i%3)*0.002,
		}
	}

	return buckets
}
