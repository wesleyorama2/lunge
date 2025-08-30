package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/wesleyorama2/lunge/internal/config"
	"github.com/wesleyorama2/lunge/internal/http"
	"github.com/wesleyorama2/lunge/internal/output"
	"github.com/wesleyorama2/lunge/pkg/jsonpath"
	"github.com/wesleyorama2/lunge/pkg/jsonschema"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run tests from a configuration file",
	Run: func(cmd *cobra.Command, args []string) {
		configFile, _ := cmd.Flags().GetString("config")
		environment, _ := cmd.Flags().GetString("environment")
		suite, _ := cmd.Flags().GetString("suite")
		testName, _ := cmd.Flags().GetString("test")
		verbose, _ := cmd.Flags().GetBool("verbose")
		timeout, _ := cmd.Flags().GetDuration("timeout")
		noColor, _ := cmd.Flags().GetBool("no-color")
		formatStr, _ := cmd.Flags().GetString("format")

		if configFile == "" {
			fmt.Println("Error: config file is required")
			cmd.Help()
			return
		}

		if environment == "" {
			fmt.Println("Error: environment is required")
			cmd.Help()
			return
		}

		// Load configuration
		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		// Validate configuration
		errors := config.ValidateConfig(cfg)
		if len(errors) > 0 {
			fmt.Fprintln(os.Stderr, "Configuration validation errors:")
			for _, err := range errors {
				fmt.Fprintf(os.Stderr, "  - %s\n", err.Error())
			}
			os.Exit(1)
		}

		// Validate environment
		if err := config.ValidateEnvironment(cfg, environment); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Create formatter with specified format
		format := output.FormatText
		if formatStr != "" {
			format = output.OutputFormat(formatStr)
		}

		var formatter output.FormatProvider
		var junitFormatter *output.JUnitFormatter
		var junitTestData []output.JUnitTestCaseData

		if format == output.FormatJUnit {
			junitFormatter = &output.JUnitFormatter{
				Verbose:   verbose,
				SuiteName: suite,
			}
			formatter = junitFormatter
			junitTestData = []output.JUnitTestCaseData{}
		} else {
			formatter = output.NewFormatterWithFormat(format, verbose, noColor)
		}

		// Create HTTP client
		client := http.NewClient(
			http.WithTimeout(timeout),
		)

		// Get environment
		env := cfg.Environments[environment]
		envVars := env.Vars

		// Initialize test results
		totalTests := 0
		passedTests := 0
		failedTests := 0
		totalAssertions := 0
		passedAssertions := 0
		failedAssertions := 0
		startTime := time.Now()

		if suite != "" {
			// Validate suite
			if err := config.ValidateSuite(cfg, suite); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Get suite
			suiteConfig := cfg.Suites[suite]

			// Merge suite variables with environment variables
			if suiteConfig.Vars != nil {
				for key, value := range suiteConfig.Vars {
					envVars[key] = config.ProcessEnvironment(value, envVars)
				}
			}

			// Only print status messages for text format
			if format == output.FormatText {
				fmt.Printf("▶ RUNNING TEST SUITE: %s (%d tests)\n\n", suite, len(suiteConfig.Tests))
			}

			// Initialize JSON formatter if needed
			if format == output.FormatJSON {
				if jsonFormatter, ok := formatter.(*output.JSONFormatter); ok {
					jsonFormatter.TestResults = &output.TestSuiteResult{
						Suite:     suite,
						Timestamp: time.Now().Format(time.RFC3339),
						Tests:     []output.TestResult{},
					}
				}
			}

			// Initialize YAML formatter if needed
			if format == output.FormatYAML {
				if yamlFormatter, ok := formatter.(*output.YAMLFormatter); ok {
					yamlFormatter.TestResults = &output.TestSuiteResult{
						Suite:     suite,
						Timestamp: time.Now().Format(time.RFC3339),
						Tests:     []output.TestResult{},
					}
				}
			}

			// Run tests
			for i, test := range suiteConfig.Tests {
				if testName == "" || test.Name == testName {
					testStartTime := time.Now()
					testResults := runTest(i+1, test, cfg, env, envVars, client, formatter, timeout, noColor)
					testDuration := time.Since(testStartTime).Milliseconds()

					// For JUnit format, collect test data after the test completes
					if format == output.FormatJUnit && junitFormatter != nil && junitFormatter.CurrentTest != nil {
						testData := *junitFormatter.CurrentTest
						testData.Duration = testDuration
						testData.Passed = testResults.passed
						junitTestData = append(junitTestData, testData)
						// Clear CurrentTest for next test
						junitFormatter.CurrentTest = nil
					}

					totalTests++
					if testResults.passed {
						passedTests++
					} else {
						failedTests++
					}

					totalAssertions += testResults.totalAssertions
					passedAssertions += testResults.passedAssertions
					failedAssertions += testResults.failedAssertions
				}
			}
		} else if testName != "" {
			fmt.Fprintf(os.Stderr, "Error: test name specified without suite\n")
			os.Exit(1)
		} else {
			fmt.Fprintf(os.Stderr, "Error: either suite or test is required\n")
			os.Exit(1)
		}

		// Calculate duration
		duration := time.Since(startTime)

		// Handle different output formats
		if format == output.FormatJSON {
			// Update JSON formatter with final statistics
			if jsonFormatter, ok := formatter.(*output.JSONFormatter); ok {
				if jsonFormatter.TestResults != nil {
					jsonFormatter.TestResults.TotalTests = totalTests
					jsonFormatter.TestResults.PassedTests = passedTests
					jsonFormatter.TestResults.FailedTests = failedTests
					jsonFormatter.TestResults.TotalAssertions = totalAssertions
					jsonFormatter.TestResults.PassedAssertions = passedAssertions
					jsonFormatter.TestResults.FailedAssertions = failedAssertions
					jsonFormatter.TestResults.Duration = duration.Milliseconds()
				}
				// Print the complete JSON output
				fmt.Println(jsonFormatter.GetTestSuiteJSON())
			}
		} else if format == output.FormatYAML {
			// Update YAML formatter with final statistics
			if yamlFormatter, ok := formatter.(*output.YAMLFormatter); ok {
				if yamlFormatter.TestResults != nil {
					yamlFormatter.TestResults.TotalTests = totalTests
					yamlFormatter.TestResults.PassedTests = passedTests
					yamlFormatter.TestResults.FailedTests = failedTests
					yamlFormatter.TestResults.TotalAssertions = totalAssertions
					yamlFormatter.TestResults.PassedAssertions = passedAssertions
					yamlFormatter.TestResults.FailedAssertions = failedAssertions
					yamlFormatter.TestResults.Duration = duration.Milliseconds()
				}
				// Print the complete YAML output
				fmt.Println(yamlFormatter.GetTestSuiteYAML())
			}
		} else if format == output.FormatJUnit {
			// Output JUnit XML format
			if junitFormatter != nil {
				junitFormatter.SetTestSuite(suite, junitTestData, duration.Milliseconds())
				xmlOutput := junitFormatter.GetTestSuiteXML()
				fmt.Print(xmlOutput)
			}
		} else if format == output.FormatText {
			// Print text summary
			fmt.Printf("\n▶ TEST SUITE SUMMARY: %s\n", suite)

			// Format test results
			testColor := color.New(color.Bold)
			if failedTests > 0 {
				testColor.Add(color.FgRed)
			} else {
				testColor.Add(color.FgGreen)
			}

			if noColor {
				testColor.DisableColor()
			}

			testStatus := "✅"
			if failedTests > 0 {
				testStatus = "❌"
			}

			fmt.Printf("  %s Tests: %s passed, %s failed\n",
				testStatus,
				testColor.Sprint(passedTests),
				testColor.Sprint(failedTests))

			assertionStatus := "✅"
			if failedAssertions > 0 {
				assertionStatus = "❌"
			}

			fmt.Printf("  %s Assertions: %s passed, %s failed\n",
				assertionStatus,
				testColor.Sprint(passedAssertions),
				testColor.Sprint(failedAssertions))

			fmt.Printf("  %s Total time: %dms\n", testStatus, duration.Milliseconds())
		}

		// Exit with error if any tests failed
		if failedTests > 0 {
			os.Exit(1)
		}
	},
}

// TestResults holds the results of a test run
type TestResults struct {
	passed           bool
	totalAssertions  int
	passedAssertions int
	failedAssertions int
}

// runTest runs a single test
func runTest(index int, test config.Test, cfg *config.Config, env config.Environment, envVars map[string]string, client *http.Client, formatter output.FormatProvider, timeout time.Duration, noColor bool) TestResults {
	return runTestWithContext(context.Background(), index, test, cfg, env, envVars, client, formatter, timeout, noColor, true)
}

// runTestWithContext runs a single test with the given context and output options
// This function is more testable because it accepts a context and allows disabling output
func runTestWithContext(ctx context.Context, index int, test config.Test, cfg *config.Config, env config.Environment, envVars map[string]string, client *http.Client, formatter output.FormatProvider, timeout time.Duration, noColor bool, printOutput bool) TestResults {
	// Determine the format type
	isJSONFormat := false
	isYAMLFormat := false
	isJUnitFormat := false
	isTextFormat := true
	var jsonFormatter *output.JSONFormatter
	var yamlFormatter *output.YAMLFormatter
	var junitFormatter *output.JUnitFormatter

	if jf, ok := formatter.(*output.JSONFormatter); ok {
		isJSONFormat = true
		isTextFormat = false
		jsonFormatter = jf
		// Start a new test in JSON formatter
		jsonFormatter.StartTest(test.Name)
	} else if yf, ok := formatter.(*output.YAMLFormatter); ok {
		isYAMLFormat = true
		isTextFormat = false
		yamlFormatter = yf
		// Start a new test in YAML formatter
		yamlFormatter.StartTest(test.Name)
	} else if jf, ok := formatter.(*output.JUnitFormatter); ok {
		isJUnitFormat = true
		isTextFormat = false
		junitFormatter = jf
		// Start a new test in JUnit formatter
		junitFormatter.StartTest(test.Name)
	} else if _, ok := formatter.(*output.Formatter); !ok {
		// Not text format
		isTextFormat = false
	}

	// Only print for text format
	if printOutput && isTextFormat {
		fmt.Printf("TEST %d: %s\n", index, test.Name)
	}

	// Get request
	reqConfig := cfg.Requests[test.Request]

	// Process URL with environment variables
	url := config.ProcessEnvironment(reqConfig.URL, envVars)
	if url == "" {
		url = env.BaseURL
	} else if !isAbsoluteURL(url) {
		// Handle paths that start with a slash to avoid double slashes
		if strings.HasPrefix(url, "/") {
			url = env.BaseURL + url
		} else {
			url = env.BaseURL + "/" + url
		}

		// Handle trailing slash in baseURL to avoid double slashes
		url = strings.Replace(url, "//", "/", -1)

		// Fix protocol after replacing slashes
		url = strings.Replace(url, ":/", "://", 1)
	}

	// Parse URL to determine base URL and path
	baseURL, path := parseURL(url)

	// Create request
	req := http.NewRequest(reqConfig.Method, path)

	// Add headers
	for key, value := range reqConfig.Headers {
		req.WithHeader(key, config.ProcessEnvironment(value, envVars))
	}

	// Add query parameters
	for key, value := range reqConfig.QueryParams {
		req.WithQueryParam(key, config.ProcessEnvironment(value, envVars))
	}

	// Add body if present
	if reqConfig.Body != nil {
		req.WithBody(reqConfig.Body)
	}

	// Print request if enabled (only for text format)
	if printOutput && isTextFormat {
		fmt.Print("  " + strings.Replace(formatter.FormatRequest(req, baseURL), "\n", "\n  ", -1))
	} else if isJUnitFormat && junitFormatter != nil {
		// JUnit formatter stores request data internally
		formatter.FormatRequest(req, baseURL)
	} else if isJSONFormat && jsonFormatter != nil && jsonFormatter.CurrentTest != nil {
		// Store request data in JSON formatter
		// Parse the request into structured data for JSON output
		queryParams := make(map[string]string)
		for key, values := range req.QueryParams {
			if len(values) > 0 {
				queryParams[key] = values[0]
			}
		}

		jsonFormatter.CurrentTest.Request = &output.RequestData{
			Method:      req.Method,
			URL:         baseURL + req.Path,
			Headers:     req.Headers,
			QueryParams: queryParams,
			Body:        req.Body,
			Timestamp:   time.Now().Format(time.RFC3339),
		}
	} else if isYAMLFormat && yamlFormatter != nil && yamlFormatter.CurrentTest != nil {
		// Store request data in YAML formatter
		// Parse the request into structured data for YAML output
		queryParams := make(map[string]string)
		for key, values := range req.QueryParams {
			if len(values) > 0 {
				queryParams[key] = values[0]
			}
		}

		yamlFormatter.CurrentTest.Request = &output.RequestData{
			Method:      req.Method,
			URL:         baseURL + req.Path,
			Headers:     req.Headers,
			QueryParams: queryParams,
			Body:        req.Body,
			Timestamp:   time.Now().Format(time.RFC3339),
		}
	}

	// Create a timeout context if one wasn't provided
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// Update client with baseURL
	client = http.NewClient(
		http.WithTimeout(timeout),
		http.WithBaseURL(baseURL),
	)

	startTime := time.Now()
	resp, err := client.Do(ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
		return TestResults{passed: false}
	}

	// Print response if enabled (only for text format)
	if printOutput && isTextFormat {
		fmt.Print("  " + strings.Replace(formatter.FormatResponse(resp), "\n", "\n  ", -1))
	} else if isJUnitFormat && junitFormatter != nil {
		// JUnit formatter stores response data internally
		formatter.FormatResponse(resp)
	} else if isJSONFormat && jsonFormatter != nil && jsonFormatter.CurrentTest != nil {
		// Store response data in JSON formatter
		headers := make(map[string]string)
		for key, values := range resp.Headers {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}

		var body interface{}
		bodyStr, err := resp.GetBodyAsString()
		if err == nil && bodyStr != "" {
			// Try to parse as JSON
			err = json.Unmarshal([]byte(bodyStr), &body)
			if err != nil {
				// If not valid JSON, use as string
				body = bodyStr
			}
		}

		jsonFormatter.CurrentTest.Response = &output.ResponseData{
			StatusCode:   resp.StatusCode,
			Status:       resp.Status,
			Headers:      headers,
			Body:         body,
			ResponseTime: resp.GetResponseTimeMillis(),
			Timing: output.TimingData{
				DNSLookup:       resp.GetDNSLookupTimeMillis(),
				TCPConnection:   resp.GetTCPConnectTimeMillis(),
				TLSHandshake:    resp.GetTLSHandshakeTimeMillis(),
				TimeToFirstByte: resp.GetTimeToFirstByteMillis(),
				ContentTransfer: resp.GetContentTransferTimeMillis(),
				Total:           resp.GetTotalTimeMillis(),
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}
	} else if isYAMLFormat && yamlFormatter != nil && yamlFormatter.CurrentTest != nil {
		// Store response data in YAML formatter
		headers := make(map[string]string)
		for key, values := range resp.Headers {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}

		var body interface{}
		bodyStr, err := resp.GetBodyAsString()
		if err == nil && bodyStr != "" {
			// Try to parse as JSON
			err = json.Unmarshal([]byte(bodyStr), &body)
			if err != nil {
				// If not valid JSON, use as string
				body = bodyStr
			}
		}

		yamlFormatter.CurrentTest.Response = &output.ResponseData{
			StatusCode:   resp.StatusCode,
			Status:       resp.Status,
			Headers:      headers,
			Body:         body,
			ResponseTime: resp.GetResponseTimeMillis(),
			Timing: output.TimingData{
				DNSLookup:       resp.GetDNSLookupTimeMillis(),
				TCPConnection:   resp.GetTCPConnectTimeMillis(),
				TLSHandshake:    resp.GetTLSHandshakeTimeMillis(),
				TimeToFirstByte: resp.GetTimeToFirstByteMillis(),
				ContentTransfer: resp.GetContentTransferTimeMillis(),
				Total:           resp.GetTotalTimeMillis(),
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}

	// Extract variables
	if reqConfig.Extract != nil {
		// TODO: Implement variable extraction
	}

	// Run assertions
	results := TestResults{
		passed:           true,
		totalAssertions:  len(test.Assertions),
		passedAssertions: 0,
		failedAssertions: 0,
	}

	// Get response body as JSON for assertions
	var responseBody interface{}
	bodyStr, _ := resp.GetBodyAsString()
	if bodyStr != "" {
		json.Unmarshal([]byte(bodyStr), &responseBody)
	}

	for _, assertion := range test.Assertions {
		passed, message := runAssertion(assertion, resp, responseBody, startTime, cfg)

		if passed {
			results.passedAssertions++
			if printOutput && isTextFormat {
				fmt.Printf("  %s ASSERTION PASSED: %s\n", output.SuccessIcon(noColor), message)
			}
		} else {
			results.failedAssertions++
			results.passed = false
			if printOutput && isTextFormat {
				fmt.Printf("  %s ASSERTION FAILED: %s\n", output.ErrorIcon(noColor), message)
			}
		}

		// Add assertion to JSON formatter if applicable
		if isJSONFormat && jsonFormatter != nil {
			// Determine assertion type from the assertion map
			assertionType := "unknown"
			var field string
			var expected interface{}
			var actual interface{}

			if _, ok := assertion["status"]; ok {
				assertionType = "status"
				expected = assertion["status"]
				actual = resp.StatusCode
			} else if _, ok := assertion["responseTime"]; ok {
				assertionType = "responseTime"
				expected = assertion["responseTime"]
				actual = resp.GetResponseTimeMillis()
			} else if _, ok := assertion["body"]; ok {
				assertionType = "body"
				field = "body"
				expected = assertion["body"]
				actual = responseBody
			} else if _, ok := assertion["jsonPath"]; ok {
				assertionType = "jsonPath"
				field = fmt.Sprintf("%v", assertion["jsonPath"])
				if val, ok := assertion["value"]; ok {
					expected = val
				}
			} else if _, ok := assertion["jsonSchema"]; ok {
				assertionType = "jsonSchema"
				expected = assertion["jsonSchema"]
			}

			jsonFormatter.AddAssertion(output.AssertionResult{
				Type:     assertionType,
				Field:    field,
				Expected: expected,
				Actual:   actual,
				Passed:   passed,
				Message:  message,
			})
		}

		// Add assertion to YAML formatter if applicable
		if isYAMLFormat && yamlFormatter != nil {
			// Determine assertion type from the assertion map
			assertionType := "unknown"
			var field string
			var expected interface{}
			var actual interface{}

			if _, ok := assertion["status"]; ok {
				assertionType = "status"
				expected = assertion["status"]
				actual = resp.StatusCode
			} else if _, ok := assertion["responseTime"]; ok {
				assertionType = "responseTime"
				expected = assertion["responseTime"]
				actual = resp.GetResponseTimeMillis()
			} else if _, ok := assertion["body"]; ok {
				assertionType = "body"
				field = "body"
				expected = assertion["body"]
				actual = responseBody
			} else if _, ok := assertion["jsonPath"]; ok {
				assertionType = "jsonPath"
				field = fmt.Sprintf("%v", assertion["jsonPath"])
				if val, ok := assertion["value"]; ok {
					expected = val
				}
			} else if _, ok := assertion["jsonSchema"]; ok {
				assertionType = "jsonSchema"
				expected = assertion["jsonSchema"]
			}

			yamlFormatter.AddAssertion(output.AssertionResult{
				Type:     assertionType,
				Field:    field,
				Expected: expected,
				Actual:   actual,
				Passed:   passed,
				Message:  message,
			})
		}

		// Add assertion to JUnit formatter if applicable
		if isJUnitFormat && junitFormatter != nil {
			// Determine assertion type from the assertion map
			assertionType := "unknown"
			var field string
			var expected interface{}
			var actual interface{}

			if _, ok := assertion["status"]; ok {
				assertionType = "status"
				expected = assertion["status"]
				actual = resp.StatusCode
			} else if _, ok := assertion["responseTime"]; ok {
				assertionType = "responseTime"
				expected = assertion["responseTime"]
				actual = resp.GetResponseTimeMillis()
			} else if _, ok := assertion["body"]; ok {
				assertionType = "body"
				field = "body"
				expected = assertion["body"]
				actual = responseBody
			} else if _, ok := assertion["jsonPath"]; ok {
				assertionType = "jsonPath"
				field = fmt.Sprintf("%v", assertion["jsonPath"])
				if val, ok := assertion["value"]; ok {
					expected = val
				}
			} else if _, ok := assertion["jsonSchema"]; ok {
				assertionType = "jsonSchema"
				expected = assertion["jsonSchema"]
			}

			junitFormatter.AddAssertion(output.AssertionResult{
				Type:     assertionType,
				Field:    field,
				Expected: expected,
				Actual:   actual,
				Passed:   passed,
				Message:  message,
			})
		}
	}

	// Print test result if enabled (only for text format)
	if printOutput && isTextFormat {
		if results.passed {
			fmt.Printf("\n  %s TEST PASSED (%dms)\n\n", output.SuccessIcon(noColor), resp.GetResponseTimeMillis())
		} else {
			fmt.Printf("\n  %s TEST FAILED (%dms)\n\n", output.ErrorIcon(noColor), resp.GetResponseTimeMillis())
		}
	}

	// End test in JSON formatter
	if isJSONFormat && jsonFormatter != nil {
		jsonFormatter.EndTest(results.passed, resp.GetResponseTimeMillis())
	}

	// End test in YAML formatter
	if isYAMLFormat && yamlFormatter != nil {
		yamlFormatter.EndTest(results.passed, resp.GetResponseTimeMillis())
	}

	// End test in JUnit formatter
	if isJUnitFormat && junitFormatter != nil {
		junitFormatter.EndTest(results.passed, resp.GetResponseTimeMillis())
	}

	return results
}

// runAssertion runs a single assertion
func runAssertion(assertion map[string]interface{}, resp *http.Response, responseBody interface{}, startTime time.Time, cfg *config.Config) (bool, string) {
	// Check status code assertion
	if status, ok := assertion["status"]; ok {
		statusInt, _ := strconv.Atoi(fmt.Sprintf("%v", status))
		if statusInt != resp.StatusCode {
			return false, fmt.Sprintf("Status code is %d, expected %d", resp.StatusCode, statusInt)
		}
		return true, fmt.Sprintf("Status code is %d", resp.StatusCode)
	}

	// Check response time assertion
	if responseTime, ok := assertion["responseTime"]; ok {
		timeStr := fmt.Sprintf("%v", responseTime)
		actualTime := resp.GetResponseTimeMillis()

		// Less than comparison
		if strings.HasPrefix(timeStr, "<") {
			maxTime, _ := strconv.Atoi(strings.TrimPrefix(timeStr, "<"))
			if actualTime >= int64(maxTime) {
				return false, fmt.Sprintf("Response time %dms is not less than %dms", actualTime, maxTime)
			}
			return true, fmt.Sprintf("Response time %dms is less than %dms", actualTime, maxTime)
		}

		// Greater than comparison
		if strings.HasPrefix(timeStr, ">") {
			minTime, _ := strconv.Atoi(strings.TrimPrefix(timeStr, ">"))
			if actualTime <= int64(minTime) {
				return false, fmt.Sprintf("Response time %dms is not greater than %dms", actualTime, minTime)
			}
			return true, fmt.Sprintf("Response time %dms is greater than %dms", actualTime, minTime)
		}

		// Equal comparison
		if strings.HasPrefix(timeStr, "=") {
			expectedTime, _ := strconv.Atoi(strings.TrimPrefix(timeStr, "="))
			if actualTime != int64(expectedTime) {
				return false, fmt.Sprintf("Response time %dms is not equal to %dms", actualTime, expectedTime)
			}
			return true, fmt.Sprintf("Response time %dms is equal to %dms", actualTime, expectedTime)
		}

		// Less than or equal comparison
		if strings.HasPrefix(timeStr, "<=") {
			maxTimeStr := strings.TrimPrefix(timeStr, "<=")
			maxTime, err := strconv.Atoi(maxTimeStr)
			if err != nil {
				return false, fmt.Sprintf("Invalid response time value: %s", timeStr)
			}

			if actualTime > int64(maxTime) {
				return false, fmt.Sprintf("Response time %dms is not less than or equal to %dms", actualTime, maxTime)
			}
			return true, fmt.Sprintf("Response time %dms is less than or equal to %dms", actualTime, maxTime)
		}

		// Greater than or equal comparison
		if strings.HasPrefix(timeStr, ">=") {
			minTime, _ := strconv.Atoi(strings.TrimPrefix(timeStr, ">="))
			if actualTime < int64(minTime) {
				return false, fmt.Sprintf("Response time %dms is not greater than or equal to %dms", actualTime, minTime)
			}
			return true, fmt.Sprintf("Response time %dms is greater than or equal to %dms", actualTime, minTime)
		}

		// Default to exact match if no operator is provided
		expectedTime, _ := strconv.Atoi(timeStr)
		if actualTime != int64(expectedTime) {
			return false, fmt.Sprintf("Response time %dms is not equal to %dms", actualTime, expectedTime)
		}
		return true, fmt.Sprintf("Response time %dms is equal to %dms", actualTime, expectedTime)
	}

	// Check header assertions
	if header, ok := assertion["header"]; ok {
		headerName := fmt.Sprintf("%v", header)
		headerValues := resp.Headers[headerName]

		// Check header exists assertion
		if exists, ok := assertion["exists"]; ok {
			existsBool, _ := strconv.ParseBool(fmt.Sprintf("%v", exists))
			headerExists := len(headerValues) > 0

			if existsBool == headerExists {
				return true, fmt.Sprintf("Header %s exists: %v", headerName, existsBool)
			} else {
				return false, fmt.Sprintf("Header %s exists: %v, expected: %v", headerName, headerExists, existsBool)
			}
		}

		// Check header equals assertion
		if equals, ok := assertion["equals"]; ok {
			expectedValue := fmt.Sprintf("%v", equals)

			if len(headerValues) > 0 && headerValues[0] == expectedValue {
				return true, fmt.Sprintf("Header %s equals %s", headerName, expectedValue)
			} else {
				actualValue := ""
				if len(headerValues) > 0 {
					actualValue = headerValues[0]
				}
				return false, fmt.Sprintf("Header %s value is %s, expected %s", headerName, actualValue, expectedValue)
			}
		}

		// Check header contains assertion
		if contains, ok := assertion["contains"]; ok {
			containsStr := fmt.Sprintf("%v", contains)

			if len(headerValues) > 0 && strings.Contains(headerValues[0], containsStr) {
				return true, fmt.Sprintf("Header %s contains %s", headerName, containsStr)
			} else {
				actualValue := ""
				if len(headerValues) > 0 {
					actualValue = headerValues[0]
				}
				return false, fmt.Sprintf("Header %s value %s does not contain %s", headerName, actualValue, containsStr)
			}
		}

		// Check header matches assertion
		if matches, ok := assertion["matches"]; ok {
			patternStr := fmt.Sprintf("%v", matches)

			// Compile and match the regex pattern
			pattern, err := regexp.Compile(patternStr)
			if err != nil {
				return false, fmt.Sprintf("Invalid regex pattern: %s", patternStr)
			}

			if len(headerValues) > 0 && pattern.MatchString(headerValues[0]) {
				return true, fmt.Sprintf("Header %s matches pattern %s", headerName, patternStr)
			} else {
				actualValue := ""
				if len(headerValues) > 0 {
					actualValue = headerValues[0]
				}
				return false, fmt.Sprintf("Header %s value %s does not match pattern %s", headerName, actualValue, patternStr)
			}
		}
	}

	// Check path assertion
	if path, ok := assertion["path"]; ok {
		pathStr := fmt.Sprintf("%v", path)
		bodyStr, _ := resp.GetBodyAsString()

		// Check exists assertion first
		if exists, ok := assertion["exists"]; ok {
			existsBool, _ := strconv.ParseBool(fmt.Sprintf("%v", exists))

			// Extract value using JSONPath
			value, err := jsonpath.Extract(bodyStr, pathStr)

			// Handle non-existent paths
			valueExists := err == nil && value != ""

			if existsBool == valueExists {
				return true, fmt.Sprintf("Path %s exists: %v", pathStr, existsBool)
			} else {
				return false, fmt.Sprintf("Path %s exists: %v, expected: %v", pathStr, valueExists, existsBool)
			}
		}

		// For all other assertions, extract the value first
		value, err := jsonpath.Extract(bodyStr, pathStr)
		if err != nil {
			return false, fmt.Sprintf("Failed to extract path %s: %v", pathStr, err)
		}

		// Check equals assertion
		if equals, ok := assertion["equals"]; ok {
			expectedValue := fmt.Sprintf("%v", equals)

			if value == expectedValue {
				return true, fmt.Sprintf("Path %s equals %s", pathStr, expectedValue)
			} else {
				return false, fmt.Sprintf("Path %s value is %s, expected %s", pathStr, value, expectedValue)
			}
		}

		// Check isArray assertion
		if _, ok := assertion["isArray"]; ok {
			// Check if the value starts with [ and ends with ]
			isArray := strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]")

			if isArray {
				return true, fmt.Sprintf("Path %s is an array", pathStr)
			} else {
				return false, fmt.Sprintf("Path %s is not an array", pathStr)
			}
		}

		// Check minLength assertion
		if minLength, ok := assertion["minLength"]; ok {
			minLengthInt, _ := strconv.Atoi(fmt.Sprintf("%v", minLength))

			// Parse the value as JSON to check its length
			var arrayValue []interface{}
			if err := json.Unmarshal([]byte(value), &arrayValue); err != nil {
				return false, fmt.Sprintf("Path %s value is not a valid array", pathStr)
			}

			if len(arrayValue) >= minLengthInt {
				return true, fmt.Sprintf("Path %s has %d items (min: %d)", pathStr, len(arrayValue), minLengthInt)
			} else {
				return false, fmt.Sprintf("Path %s has %d items, expected at least %d", pathStr, len(arrayValue), minLengthInt)
			}
		}

		// Check matches assertion
		if matches, ok := assertion["matches"]; ok {
			patternStr := fmt.Sprintf("%v", matches)

			// Compile and match the regex pattern
			pattern, err := regexp.Compile(patternStr)
			if err != nil {
				return false, fmt.Sprintf("Invalid regex pattern: %s", patternStr)
			}

			if pattern.MatchString(value) {
				return true, fmt.Sprintf("Path %s matches pattern %s", pathStr, patternStr)
			} else {
				return false, fmt.Sprintf("Path %s value %s does not match pattern %s", pathStr, value, patternStr)
			}
		}

		// Check contains assertion
		if contains, ok := assertion["contains"]; ok {
			containsStr := fmt.Sprintf("%v", contains)

			if strings.Contains(value, containsStr) {
				return true, fmt.Sprintf("Path %s contains %s", pathStr, containsStr)
			} else {
				return false, fmt.Sprintf("Path %s value %s does not contain %s", pathStr, value, containsStr)
			}
		}
	}

	// Check schema assertion
	if schema, ok := assertion["schema"]; ok {
		schemaName := fmt.Sprintf("%v", schema)
		bodyStr, _ := resp.GetBodyAsString()

		// Get the schema from the configuration
		schemaStr, err := getSchemaFromConfig(schemaName, cfg)
		if err != nil {
			return false, fmt.Sprintf("Schema validation failed: %v", err)
		}

		// Validate the response against the schema
		valid, errors := jsonschema.ValidateWithErrors(bodyStr, schemaStr)
		if valid {
			return true, fmt.Sprintf("Response body validates against schema %s", schemaName)
		} else {
			return false, fmt.Sprintf("Schema validation failed for %s: %v", schemaName, errors)
		}
	}

	// Default case
	return false, "Unknown assertion"
}

// getSchemaFromConfig retrieves a schema from the configuration by name
func getSchemaFromConfig(schemaName string, cfg *config.Config) (string, error) {
	// Check if schemas section exists
	if cfg.Schemas == nil {
		return "", fmt.Errorf("no schemas defined in configuration")
	}

	// Check if the schema exists
	schemaJSON, ok := cfg.Schemas[schemaName]
	if !ok {
		return "", fmt.Errorf("schema %s not found in configuration", schemaName)
	}

	// Return the schema as a string
	return string(schemaJSON), nil
}

func init() {
	// Add flags to TEST command
	testCmd.Flags().StringP("config", "c", "", "Configuration file (required)")
	testCmd.Flags().StringP("environment", "e", "", "Environment to use (required)")
	testCmd.Flags().StringP("suite", "s", "", "Test suite to run")
	testCmd.Flags().StringP("test", "t", "", "Specific test to run")
	testCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	testCmd.Flags().DurationP("timeout", "T", 30*time.Second, "Request timeout")
	testCmd.Flags().Bool("no-color", false, "Disable colored output")
	testCmd.Flags().String("format", "", "Output format (text, json, yaml, junit)")
}
