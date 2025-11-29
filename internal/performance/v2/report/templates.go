// Package report provides HTML report generation for v2 performance test results.
package report

// htmlTemplate is the main HTML template for the report
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Name}} - Performance Test Report</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        :root {
            --bg-primary: #ffffff;
            --bg-secondary: #f8fafc;
            --bg-card: #ffffff;
            --text-primary: #1e293b;
            --text-secondary: #64748b;
            --text-muted: #94a3b8;
            --border-color: #e2e8f0;
            --accent-primary: #3b82f6;
            --accent-success: #22c55e;
            --accent-warning: #f59e0b;
            --accent-error: #ef4444;
            --shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
            --shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.1);
        }

        [data-theme="dark"] {
            --bg-primary: #0f172a;
            --bg-secondary: #1e293b;
            --bg-card: #1e293b;
            --text-primary: #f1f5f9;
            --text-secondary: #94a3b8;
            --text-muted: #64748b;
            --border-color: #334155;
            --shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
            --shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.3);
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background-color: var(--bg-secondary);
            color: var(--text-primary);
            line-height: 1.6;
            min-height: 100vh;
        }

        .container {
            max-width: 1400px;
            margin: 0 auto;
            padding: 2rem;
        }

        /* Header */
        .header {
            background: var(--bg-card);
            border-radius: 12px;
            padding: 2rem;
            margin-bottom: 2rem;
            box-shadow: var(--shadow);
            display: flex;
            justify-content: space-between;
            align-items: center;
            flex-wrap: wrap;
            gap: 1rem;
        }

        .header-left h1 {
            font-size: 1.75rem;
            font-weight: 700;
            margin-bottom: 0.5rem;
        }

        .header-left .description {
            color: var(--text-secondary);
            font-size: 0.95rem;
        }

        .header-left .meta {
            display: flex;
            gap: 2rem;
            margin-top: 0.75rem;
            font-size: 0.875rem;
            color: var(--text-muted);
        }

        .header-right {
            display: flex;
            align-items: center;
            gap: 1rem;
        }

        .status {
            display: inline-flex;
            align-items: center;
            gap: 0.5rem;
            padding: 0.75rem 1.5rem;
            border-radius: 8px;
            font-weight: 600;
            font-size: 1rem;
        }

        .status.pass {
            background-color: rgba(34, 197, 94, 0.1);
            color: var(--accent-success);
            border: 1px solid rgba(34, 197, 94, 0.2);
        }

        .status.fail {
            background-color: rgba(239, 68, 68, 0.1);
            color: var(--accent-error);
            border: 1px solid rgba(239, 68, 68, 0.2);
        }

        .theme-toggle {
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            padding: 0.5rem;
            cursor: pointer;
            color: var(--text-secondary);
            font-size: 1.25rem;
            transition: all 0.2s;
        }

        .theme-toggle:hover {
            background: var(--border-color);
        }

        /* Metrics Grid */
        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
            margin-bottom: 2rem;
        }

        .metric-card {
            background: var(--bg-card);
            border-radius: 12px;
            padding: 1.5rem;
            box-shadow: var(--shadow);
        }

        .metric-card .label {
            font-size: 0.75rem;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            color: var(--text-muted);
            margin-bottom: 0.5rem;
        }

        .metric-card .value {
            font-size: 1.75rem;
            font-weight: 700;
            color: var(--text-primary);
        }

        .metric-card .unit {
            font-size: 0.875rem;
            color: var(--text-secondary);
            margin-left: 0.25rem;
        }

        .metric-card .change {
            font-size: 0.75rem;
            margin-top: 0.5rem;
        }

        .metric-card .change.positive {
            color: var(--accent-success);
        }

        .metric-card .change.negative {
            color: var(--accent-error);
        }

        /* Section */
        .section {
            background: var(--bg-card);
            border-radius: 12px;
            padding: 1.5rem;
            margin-bottom: 2rem;
            box-shadow: var(--shadow);
        }

        .section-title {
            font-size: 1.125rem;
            font-weight: 600;
            margin-bottom: 1.5rem;
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .section-title::before {
            content: '';
            width: 4px;
            height: 1.25rem;
            background: var(--accent-primary);
            border-radius: 2px;
        }

        /* Latency Table */
        .latency-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
            gap: 1rem;
        }

        .latency-item {
            text-align: center;
            padding: 1rem;
            background: var(--bg-secondary);
            border-radius: 8px;
        }

        .latency-item .percentile {
            font-size: 0.75rem;
            text-transform: uppercase;
            color: var(--text-muted);
            margin-bottom: 0.25rem;
        }

        .latency-item .time {
            font-size: 1.25rem;
            font-weight: 600;
            color: var(--text-primary);
        }

        /* Charts */
        .chart-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(450px, 1fr));
            gap: 1.5rem;
        }

        .chart-container {
            background: var(--bg-card);
            border-radius: 12px;
            padding: 1.5rem;
            box-shadow: var(--shadow);
        }

        .chart-title {
            font-size: 0.875rem;
            font-weight: 600;
            color: var(--text-secondary);
            margin-bottom: 1rem;
        }

        .chart-wrapper {
            position: relative;
            height: 250px;
        }

        /* Scenarios */
        .scenario-card {
            background: var(--bg-secondary);
            border-radius: 8px;
            padding: 1.25rem;
            margin-bottom: 1rem;
        }

        .scenario-card:last-child {
            margin-bottom: 0;
        }

        .scenario-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 1rem;
        }

        .scenario-name {
            font-weight: 600;
            font-size: 1rem;
        }

        .scenario-executor {
            font-size: 0.75rem;
            color: var(--text-muted);
            background: var(--bg-card);
            padding: 0.25rem 0.75rem;
            border-radius: 4px;
        }

        .scenario-metrics {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
            gap: 1rem;
        }

        .scenario-metric {
            display: flex;
            flex-direction: column;
        }

        .scenario-metric .label {
            font-size: 0.75rem;
            color: var(--text-muted);
        }

        .scenario-metric .value {
            font-size: 1rem;
            font-weight: 600;
        }

        /* Thresholds */
        .threshold-list {
            display: flex;
            flex-direction: column;
            gap: 0.75rem;
        }

        .threshold-item {
            display: flex;
            align-items: center;
            gap: 1rem;
            padding: 1rem;
            background: var(--bg-secondary);
            border-radius: 8px;
        }

        .threshold-icon {
            font-size: 1.25rem;
        }

        .threshold-icon.pass {
            color: var(--accent-success);
        }

        .threshold-icon.fail {
            color: var(--accent-error);
        }

        .threshold-info {
            flex: 1;
        }

        .threshold-metric {
            font-weight: 600;
            font-size: 0.875rem;
        }

        .threshold-expression {
            font-size: 0.75rem;
            color: var(--text-muted);
        }

        .threshold-value {
            font-size: 0.875rem;
            color: var(--text-secondary);
            text-align: right;
        }

        /* Phase Legend */
        .phase-legend {
            display: flex;
            flex-wrap: wrap;
            gap: 1rem;
            margin-top: 1rem;
            padding-top: 1rem;
            border-top: 1px solid var(--border-color);
        }

        .phase-item {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            font-size: 0.75rem;
            color: var(--text-muted);
        }

        .phase-dot {
            width: 12px;
            height: 12px;
            border-radius: 3px;
        }

        .phase-dot.init { background: #94a3b8; }
        .phase-dot.warmup { background: #f59e0b; }
        .phase-dot.ramp-up { background: #22c55e; }
        .phase-dot.steady { background: #3b82f6; }
        .phase-dot.ramp-down { background: #8b5cf6; }
        .phase-dot.cooldown { background: #ec4899; }
        .phase-dot.done { background: #64748b; }

        /* Request Stats Table */
        .stats-table {
            width: 100%;
            border-collapse: collapse;
        }

        .stats-table th,
        .stats-table td {
            padding: 0.75rem 1rem;
            text-align: left;
            border-bottom: 1px solid var(--border-color);
        }

        .stats-table th {
            font-size: 0.75rem;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            color: var(--text-muted);
            font-weight: 600;
        }

        .stats-table td {
            font-size: 0.875rem;
        }

        .stats-table tr:last-child td {
            border-bottom: none;
        }

        .stats-table tr:hover {
            background: var(--bg-secondary);
        }

        /* Footer */
        .footer {
            text-align: center;
            padding: 2rem;
            color: var(--text-muted);
            font-size: 0.75rem;
        }

        /* Responsive */
        @media (max-width: 768px) {
            .container {
                padding: 1rem;
            }

            .header {
                flex-direction: column;
                align-items: flex-start;
            }

            .chart-grid {
                grid-template-columns: 1fr;
            }

            .metrics-grid {
                grid-template-columns: repeat(2, 1fr);
            }
        }

        /* Print styles */
        @media print {
            body {
                background: white;
            }

            .theme-toggle {
                display: none;
            }

            .container {
                max-width: none;
                padding: 0;
            }

            .section, .header, .metric-card, .chart-container {
                break-inside: avoid;
                box-shadow: none;
                border: 1px solid #e2e8f0;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <!-- Header -->
        <header class="header">
            <div class="header-left">
                <h1>{{.Name}}</h1>
                {{if .Description}}<p class="description">{{.Description}}</p>{{end}}
                <div class="meta">
                    <span>📅 {{.StartTime.Format "2006-01-02 15:04:05"}}</span>
                    <span>⏱️ {{formatDuration .Duration}}</span>
                </div>
            </div>
            <div class="header-right">
                <div class="status {{if .Passed}}pass{{else}}fail{{end}}">
                    {{if .Passed}}✓ PASSED{{else}}✗ FAILED{{end}}
                </div>
                <button class="theme-toggle" onclick="toggleTheme()" title="Toggle dark mode">🌙</button>
            </div>
        </header>

        <!-- Key Metrics -->
        <div class="metrics-grid">
            <div class="metric-card">
                <div class="label">Total Requests</div>
                <div class="value">{{formatNumber .Metrics.TotalRequests}}</div>
            </div>
            <div class="metric-card">
                <div class="label">Throughput</div>
                <div class="value">{{printf "%.1f" .Metrics.RPS}}<span class="unit">req/s</span></div>
            </div>
            <div class="metric-card">
                <div class="label">Error Rate</div>
                <div class="value">{{printf "%.2f" (mul .Metrics.ErrorRate 100)}}<span class="unit">%</span></div>
            </div>
            <div class="metric-card">
                <div class="label">P95 Latency</div>
                <div class="value">{{formatLatency .Metrics.Latency.P95}}</div>
            </div>
            <div class="metric-card">
                <div class="label">Success Rate</div>
                <div class="value">{{printf "%.2f" (successRate .Metrics)}}<span class="unit">%</span></div>
            </div>
            <div class="metric-card">
                <div class="label">Data Transferred</div>
                <div class="value">{{formatBytes .Metrics.TotalBytes}}</div>
            </div>
        </div>

        <!-- Latency Statistics -->
        <section class="section">
            <h2 class="section-title">Latency Statistics</h2>
            <div class="latency-grid">
                <div class="latency-item">
                    <div class="percentile">Min</div>
                    <div class="time">{{formatLatency .Metrics.Latency.Min}}</div>
                </div>
                <div class="latency-item">
                    <div class="percentile">P50</div>
                    <div class="time">{{formatLatency .Metrics.Latency.P50}}</div>
                </div>
                <div class="latency-item">
                    <div class="percentile">P90</div>
                    <div class="time">{{formatLatency .Metrics.Latency.P90}}</div>
                </div>
                <div class="latency-item">
                    <div class="percentile">P95</div>
                    <div class="time">{{formatLatency .Metrics.Latency.P95}}</div>
                </div>
                <div class="latency-item">
                    <div class="percentile">P99</div>
                    <div class="time">{{formatLatency .Metrics.Latency.P99}}</div>
                </div>
                <div class="latency-item">
                    <div class="percentile">Max</div>
                    <div class="time">{{formatLatency .Metrics.Latency.Max}}</div>
                </div>
                <div class="latency-item">
                    <div class="percentile">Mean</div>
                    <div class="time">{{formatLatency .Metrics.Latency.Mean}}</div>
                </div>
                <div class="latency-item">
                    <div class="percentile">Std Dev</div>
                    <div class="time">{{formatLatency .Metrics.Latency.StdDev}}</div>
                </div>
            </div>
        </section>

        <!-- Charts -->
        {{if .TimeSeries}}
        <section class="section">
            <h2 class="section-title">Time Series Analysis</h2>
            <div class="chart-grid">
                <div class="chart-container">
                    <div class="chart-title">Requests Per Second</div>
                    <div class="chart-wrapper">
                        <canvas id="rpsChart"></canvas>
                    </div>
                </div>
                <div class="chart-container">
                    <div class="chart-title">Response Latency (Percentiles)</div>
                    <div class="chart-wrapper">
                        <canvas id="latencyChart"></canvas>
                    </div>
                </div>
                <div class="chart-container">
                    <div class="chart-title">Active Virtual Users</div>
                    <div class="chart-wrapper">
                        <canvas id="vusChart"></canvas>
                    </div>
                </div>
                <div class="chart-container">
                    <div class="chart-title">Error Rate</div>
                    <div class="chart-wrapper">
                        <canvas id="errorChart"></canvas>
                    </div>
                </div>
            </div>
            <div class="phase-legend">
                <div class="phase-item"><span class="phase-dot init"></span> Init</div>
                <div class="phase-item"><span class="phase-dot warmup"></span> Warmup</div>
                <div class="phase-item"><span class="phase-dot ramp-up"></span> Ramp-Up</div>
                <div class="phase-item"><span class="phase-dot steady"></span> Steady</div>
                <div class="phase-item"><span class="phase-dot ramp-down"></span> Ramp-Down</div>
                <div class="phase-item"><span class="phase-dot cooldown"></span> Cooldown</div>
            </div>
        </section>
        {{end}}

        <!-- Scenarios -->
        {{if .Scenarios}}
        <section class="section">
            <h2 class="section-title">Scenario Results</h2>
            {{range $name, $scenario := .Scenarios}}
            <div class="scenario-card">
                <div class="scenario-header">
                    <span class="scenario-name">{{$name}}</span>
                    <span class="scenario-executor">{{$scenario.Executor}}</span>
                </div>
                <div class="scenario-metrics">
                    <div class="scenario-metric">
                        <span class="label">Duration</span>
                        <span class="value">{{formatDuration $scenario.Duration}}</span>
                    </div>
                    <div class="scenario-metric">
                        <span class="label">Iterations</span>
                        <span class="value">{{formatNumber $scenario.Iterations}}</span>
                    </div>
                    <div class="scenario-metric">
                        <span class="label">Active VUs</span>
                        <span class="value">{{$scenario.ActiveVUs}}</span>
                    </div>
                    {{if $scenario.Metrics}}
                    <div class="scenario-metric">
                        <span class="label">Avg Latency</span>
                        <span class="value">{{formatLatency $scenario.Metrics.Latency.Mean}}</span>
                    </div>
                    <div class="scenario-metric">
                        <span class="label">P95 Latency</span>
                        <span class="value">{{formatLatency $scenario.Metrics.Latency.P95}}</span>
                    </div>
                    <div class="scenario-metric">
                        <span class="label">Error Rate</span>
                        <span class="value">{{printf "%.2f%%" (mul $scenario.Metrics.ErrorRate 100)}}</span>
                    </div>
                    {{end}}
                </div>
                {{if $scenario.Error}}
                <div style="margin-top: 1rem; padding: 0.75rem; background: rgba(239, 68, 68, 0.1); border-radius: 6px; color: var(--accent-error); font-size: 0.875rem;">
                    ⚠️ Error: {{$scenario.Error}}
                </div>
                {{end}}
            </div>
            {{end}}
        </section>
        {{end}}

        <!-- Per-Request Statistics -->
        {{if hasRequestStats .Scenarios}}
        <section class="section">
            <h2 class="section-title">Request Statistics</h2>
            <table class="stats-table">
                <thead>
                    <tr>
                        <th>Request</th>
                        <th>Count</th>
                        <th>Min</th>
                        <th>Mean</th>
                        <th>P50</th>
                        <th>P95</th>
                        <th>P99</th>
                        <th>Max</th>
                    </tr>
                </thead>
                <tbody>
                    {{range $name, $scenario := .Scenarios}}
                    {{range $reqName, $stats := $scenario.RequestStats}}
                    <tr>
                        <td>{{$reqName}}</td>
                        <td>{{formatNumber $stats.Count}}</td>
                        <td>{{formatLatency $stats.Latency.Min}}</td>
                        <td>{{formatLatency $stats.Latency.Mean}}</td>
                        <td>{{formatLatency $stats.Latency.P50}}</td>
                        <td>{{formatLatency $stats.Latency.P95}}</td>
                        <td>{{formatLatency $stats.Latency.P99}}</td>
                        <td>{{formatLatency $stats.Latency.Max}}</td>
                    </tr>
                    {{end}}
                    {{end}}
                </tbody>
            </table>
        </section>
        {{end}}

        <!-- Thresholds -->
        {{if .Thresholds}}
        <section class="section">
            <h2 class="section-title">Threshold Results</h2>
            <div class="threshold-list">
                {{range .Thresholds}}
                <div class="threshold-item">
                    <span class="threshold-icon {{if .Passed}}pass{{else}}fail{{end}}">
                        {{if .Passed}}✓{{else}}✗{{end}}
                    </span>
                    <div class="threshold-info">
                        <div class="threshold-metric">{{.Metric}}</div>
                        <div class="threshold-expression">{{.Expression}}</div>
                    </div>
                    <div class="threshold-value">
                        Actual: {{.Value}}
                        {{if .Message}}<br><span style="color: var(--accent-error); font-size: 0.75rem;">{{.Message}}</span>{{end}}
                    </div>
                </div>
                {{end}}
            </div>
        </section>
        {{end}}

        <!-- Footer -->
        <footer class="footer">
            <p>Generated by Lunge Performance Testing Tool • {{.EndTime.Format "2006-01-02 15:04:05 MST"}}</p>
        </footer>
    </div>

    <script>
        // Theme toggle
        function toggleTheme() {
            const html = document.documentElement;
            const currentTheme = html.getAttribute('data-theme');
            const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
            html.setAttribute('data-theme', newTheme);
            localStorage.setItem('theme', newTheme);
            updateChartColors();
        }

        // Load saved theme
        const savedTheme = localStorage.getItem('theme') || 'light';
        document.documentElement.setAttribute('data-theme', savedTheme);

        // Chart colors based on theme
        function getChartColors() {
            const isDark = document.documentElement.getAttribute('data-theme') === 'dark';
            return {
                text: isDark ? '#f1f5f9' : '#1e293b',
                grid: isDark ? '#334155' : '#e2e8f0',
                primary: '#3b82f6',
                success: '#22c55e',
                warning: '#f59e0b',
                error: '#ef4444',
                purple: '#8b5cf6',
                pink: '#ec4899',
            };
        }

        // Phase colors for background annotations
        const phaseColors = {
            'init': 'rgba(148, 163, 184, 0.1)',
            'warmup': 'rgba(245, 158, 11, 0.1)',
            'ramp-up': 'rgba(34, 197, 94, 0.1)',
            'steady': 'rgba(59, 130, 246, 0.1)',
            'ramp-down': 'rgba(139, 92, 246, 0.1)',
            'cooldown': 'rgba(236, 72, 153, 0.1)',
            'done': 'rgba(100, 116, 139, 0.1)',
        };

        // Time series data
        const timeSeriesData = {{.TimeSeriesJSON}};

        // Prepare chart data
        const labels = timeSeriesData.map((d, i) => i + 's');
        const rpsData = timeSeriesData.map(d => d.intervalRPS);
        const p50Data = timeSeriesData.map(d => d.latencyP50 / 1000000); // Convert ns to ms
        const p95Data = timeSeriesData.map(d => d.latencyP95 / 1000000);
        const p99Data = timeSeriesData.map(d => d.latencyP99 / 1000000);
        const vusData = timeSeriesData.map(d => d.activeVUs);
        const errorData = timeSeriesData.map(d => d.intervalErrorRate * 100);
        const phases = timeSeriesData.map(d => d.phase);

        // Create phase background segments
        function createPhaseBackgrounds(phases) {
            const backgrounds = [];
            let currentPhase = phases[0];
            let startIdx = 0;
            
            for (let i = 1; i <= phases.length; i++) {
                if (i === phases.length || phases[i] !== currentPhase) {
                    backgrounds.push({
                        type: 'box',
                        xMin: startIdx,
                        xMax: i - 1,
                        backgroundColor: phaseColors[currentPhase] || 'transparent',
                        borderWidth: 0,
                    });
                    if (i < phases.length) {
                        currentPhase = phases[i];
                        startIdx = i;
                    }
                }
            }
            return backgrounds;
        }

        const phaseBackgrounds = createPhaseBackgrounds(phases);

        // Chart instances
        let rpsChart, latencyChart, vusChart, errorChart;

        function createCharts() {
            const colors = getChartColors();
            
            const commonOptions = {
                responsive: true,
                maintainAspectRatio: false,
                interaction: {
                    mode: 'index',
                    intersect: false,
                },
                plugins: {
                    legend: {
                        labels: {
                            color: colors.text,
                            usePointStyle: true,
                            pointStyle: 'circle',
                        }
                    },
                    tooltip: {
                        backgroundColor: colors.grid,
                        titleColor: colors.text,
                        bodyColor: colors.text,
                        borderColor: colors.grid,
                        borderWidth: 1,
                    }
                },
                scales: {
                    x: {
                        ticks: { color: colors.text },
                        grid: { color: colors.grid },
                    },
                    y: {
                        ticks: { color: colors.text },
                        grid: { color: colors.grid },
                        beginAtZero: true,
                    }
                }
            };

            // RPS Chart
            const rpsCtx = document.getElementById('rpsChart');
            if (rpsCtx) {
                rpsChart = new Chart(rpsCtx.getContext('2d'), {
                    type: 'line',
                    data: {
                        labels: labels,
                        datasets: [{
                            label: 'Requests/sec',
                            data: rpsData,
                            borderColor: colors.primary,
                            backgroundColor: colors.primary + '20',
                            fill: true,
                            tension: 0.3,
                            pointRadius: 0,
                            borderWidth: 2,
                        }]
                    },
                    options: commonOptions
                });
            }

            // Latency Chart
            const latencyCtx = document.getElementById('latencyChart');
            if (latencyCtx) {
                latencyChart = new Chart(latencyCtx.getContext('2d'), {
                    type: 'line',
                    data: {
                        labels: labels,
                        datasets: [
                            {
                                label: 'P50',
                                data: p50Data,
                                borderColor: colors.success,
                                backgroundColor: 'transparent',
                                tension: 0.3,
                                pointRadius: 0,
                                borderWidth: 2,
                            },
                            {
                                label: 'P95',
                                data: p95Data,
                                borderColor: colors.warning,
                                backgroundColor: 'transparent',
                                tension: 0.3,
                                pointRadius: 0,
                                borderWidth: 2,
                            },
                            {
                                label: 'P99',
                                data: p99Data,
                                borderColor: colors.error,
                                backgroundColor: 'transparent',
                                tension: 0.3,
                                pointRadius: 0,
                                borderWidth: 2,
                            }
                        ]
                    },
                    options: {
                        ...commonOptions,
                        scales: {
                            ...commonOptions.scales,
                            y: {
                                ...commonOptions.scales.y,
                                title: {
                                    display: true,
                                    text: 'Latency (ms)',
                                    color: colors.text,
                                }
                            }
                        }
                    }
                });
            }

            // VUs Chart
            const vusCtx = document.getElementById('vusChart');
            if (vusCtx) {
                vusChart = new Chart(vusCtx.getContext('2d'), {
                    type: 'line',
                    data: {
                        labels: labels,
                        datasets: [{
                            label: 'Active VUs',
                            data: vusData,
                            borderColor: colors.purple,
                            backgroundColor: colors.purple + '20',
                            fill: true,
                            tension: 0.3,
                            pointRadius: 0,
                            borderWidth: 2,
                            stepped: true,
                        }]
                    },
                    options: commonOptions
                });
            }

            // Error Chart
            const errorCtx = document.getElementById('errorChart');
            if (errorCtx) {
                errorChart = new Chart(errorCtx.getContext('2d'), {
                    type: 'line',
                    data: {
                        labels: labels,
                        datasets: [{
                            label: 'Error Rate (%)',
                            data: errorData,
                            borderColor: colors.error,
                            backgroundColor: colors.error + '20',
                            fill: true,
                            tension: 0.3,
                            pointRadius: 0,
                            borderWidth: 2,
                        }]
                    },
                    options: {
                        ...commonOptions,
                        scales: {
                            ...commonOptions.scales,
                            y: {
                                ...commonOptions.scales.y,
                                max: Math.max(10, Math.ceil(Math.max(...errorData) * 1.1)),
                                title: {
                                    display: true,
                                    text: 'Error Rate (%)',
                                    color: colors.text,
                                }
                            }
                        }
                    }
                });
            }
        }

        function updateChartColors() {
            const colors = getChartColors();
            [rpsChart, latencyChart, vusChart, errorChart].forEach(chart => {
                if (chart) {
                    chart.options.plugins.legend.labels.color = colors.text;
                    chart.options.scales.x.ticks.color = colors.text;
                    chart.options.scales.x.grid.color = colors.grid;
                    chart.options.scales.y.ticks.color = colors.text;
                    chart.options.scales.y.grid.color = colors.grid;
                    if (chart.options.scales.y.title) {
                        chart.options.scales.y.title.color = colors.text;
                    }
                    chart.update();
                }
            });
        }

        // Initialize charts when DOM is ready
        document.addEventListener('DOMContentLoaded', function() {
            if (timeSeriesData && timeSeriesData.length > 0) {
                createCharts();
            }
        });
    </script>
</body>
</html>`
