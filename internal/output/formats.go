package output

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	http "github.com/wesleyorama2/lunge/internal/http"
	"gopkg.in/yaml.v3"
)

// OutputFormat represents the available output formats
type OutputFormat string

const (
	// FormatText is the default human-readable text format
	FormatText OutputFormat = "text"
	// FormatJSON outputs in JSON format
	FormatJSON OutputFormat = "json"
	// FormatYAML outputs in YAML format
	FormatYAML OutputFormat = "yaml"
	// FormatJUnit outputs in JUnit XML format (for CI/CD integration)
	FormatJUnit OutputFormat = "junit"
)

// FormatProvider is an interface for different output formatters
type FormatProvider interface {
	FormatRequest(req *http.Request, baseURL string) string
	FormatResponse(resp *http.Response) string
}

// RequestData represents the structured data of an HTTP request
type RequestData struct {
	Method      string            `json:"method" yaml:"method" xml:"method,attr"`
	URL         string            `json:"url" yaml:"url" xml:"url,attr"`
	Headers     map[string]string `json:"headers,omitempty" yaml:"headers,omitempty" xml:"headers>header,omitempty"`
	QueryParams map[string]string `json:"queryParams,omitempty" yaml:"queryParams,omitempty" xml:"queryParams>param,omitempty"`
	Body        interface{}       `json:"body,omitempty" yaml:"body,omitempty" xml:"body,omitempty"`
	Timestamp   string            `json:"timestamp" yaml:"timestamp" xml:"timestamp,attr"`
}

// TimingData represents detailed timing information for an HTTP request
type TimingData struct {
	DNSLookup       int64 `json:"dnsLookupMs,omitempty" yaml:"dnsLookupMs,omitempty" xml:"dnsLookupMs,attr,omitempty"`
	TCPConnection   int64 `json:"tcpConnectionMs,omitempty" yaml:"tcpConnectionMs,omitempty" xml:"tcpConnectionMs,attr,omitempty"`
	TLSHandshake    int64 `json:"tlsHandshakeMs,omitempty" yaml:"tlsHandshakeMs,omitempty" xml:"tlsHandshakeMs,attr,omitempty"`
	TimeToFirstByte int64 `json:"timeToFirstByteMs,omitempty" yaml:"timeToFirstByteMs,omitempty" xml:"timeToFirstByteMs,attr,omitempty"`
	ContentTransfer int64 `json:"contentTransferMs,omitempty" yaml:"contentTransferMs,omitempty" xml:"contentTransferMs,attr,omitempty"`
	Total           int64 `json:"totalMs" yaml:"totalMs" xml:"totalMs,attr"`
}

// ResponseData represents the structured data of an HTTP response
type ResponseData struct {
	StatusCode    int               `json:"statusCode" yaml:"statusCode" xml:"statusCode,attr"`
	Status        string            `json:"status" yaml:"status" xml:"status,attr"`
	Headers       map[string]string `json:"headers,omitempty" yaml:"headers,omitempty" xml:"headers>header,omitempty"`
	Body          interface{}       `json:"body,omitempty" yaml:"body,omitempty" xml:"body,omitempty"`
	ResponseTime  int64             `json:"responseTimeMs" yaml:"responseTimeMs" xml:"responseTimeMs,attr"`
	Timing        TimingData        `json:"timing,omitempty" yaml:"timing,omitempty" xml:"timing,omitempty"`
	Timestamp     string            `json:"timestamp" yaml:"timestamp" xml:"timestamp,attr"`
	ContentLength int64             `json:"contentLength,omitempty" yaml:"contentLength,omitempty" xml:"contentLength,attr,omitempty"`
}

// TestResult represents the result of a single test
type TestResult struct {
	Name       string            `json:"name" yaml:"name"`
	Passed     bool              `json:"passed" yaml:"passed"`
	Duration   int64             `json:"durationMs" yaml:"durationMs"`
	Assertions []AssertionResult `json:"assertions,omitempty" yaml:"assertions,omitempty"`
	Request    *RequestData      `json:"request,omitempty" yaml:"request,omitempty"`
	Response   *ResponseData     `json:"response,omitempty" yaml:"response,omitempty"`
}

// AssertionResult represents the result of a single assertion
type AssertionResult struct {
	Type     string      `json:"type" yaml:"type"`
	Field    string      `json:"field,omitempty" yaml:"field,omitempty"`
	Expected interface{} `json:"expected,omitempty" yaml:"expected,omitempty"`
	Actual   interface{} `json:"actual,omitempty" yaml:"actual,omitempty"`
	Passed   bool        `json:"passed" yaml:"passed"`
	Message  string      `json:"message" yaml:"message"`
}

// TestSuiteResult represents the result of a test suite
type TestSuiteResult struct {
	Suite            string       `json:"suite" yaml:"suite"`
	TotalTests       int          `json:"totalTests" yaml:"totalTests"`
	PassedTests      int          `json:"passedTests" yaml:"passedTests"`
	FailedTests      int          `json:"failedTests" yaml:"failedTests"`
	TotalAssertions  int          `json:"totalAssertions" yaml:"totalAssertions"`
	PassedAssertions int          `json:"passedAssertions" yaml:"passedAssertions"`
	FailedAssertions int          `json:"failedAssertions" yaml:"failedAssertions"`
	Duration         int64        `json:"durationMs" yaml:"durationMs"`
	Tests            []TestResult `json:"tests" yaml:"tests"`
	Timestamp        string       `json:"timestamp" yaml:"timestamp"`
}

// JSONFormatter formats output as JSON
type JSONFormatter struct {
	Verbose     bool
	Pretty      bool
	TestResults *TestSuiteResult // Store test results for final output
	CurrentTest *TestResult      // Current test being executed
}

// FormatRequest formats a request as JSON
func (f *JSONFormatter) FormatRequest(req *http.Request, baseURL string) string {
	// Build full URL
	fullURL := baseURL
	if !strings.HasSuffix(baseURL, "/") && !strings.HasPrefix(req.Path, "/") {
		fullURL += "/"
	}
	fullURL += req.Path
	if len(req.QueryParams) > 0 {
		fullURL += "?" + req.QueryParams.Encode()
	}

	// Convert query params to map for easier serialization
	queryParams := make(map[string]string)
	for key, values := range req.QueryParams {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}

	// Create request data structure
	data := RequestData{
		Method:      req.Method,
		URL:         fullURL,
		Headers:     req.Headers,
		QueryParams: queryParams,
		Body:        req.Body,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	// Marshal to JSON
	var output []byte
	var err error
	if f.Pretty {
		output, err = json.MarshalIndent(data, "", "  ")
	} else {
		output, err = json.Marshal(data)
	}

	if err != nil {
		return fmt.Sprintf(`{"error":"Failed to marshal request: %s"}`, err)
	}

	return string(output)
}

// FormatResponse formats a response as JSON
func (f *JSONFormatter) FormatResponse(resp *http.Response) string {
	// Convert headers to map for easier serialization
	headers := make(map[string]string)
	for key, values := range resp.Headers {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Parse body if present
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

	// Create response data structure
	data := ResponseData{
		StatusCode:   resp.StatusCode,
		Status:       resp.Status,
		Headers:      headers,
		Body:         body,
		ResponseTime: resp.GetResponseTimeMillis(),
		Timing: TimingData{
			DNSLookup:       resp.GetDNSLookupTimeMillis(),
			TCPConnection:   resp.GetTCPConnectTimeMillis(),
			TLSHandshake:    resp.GetTLSHandshakeTimeMillis(),
			TimeToFirstByte: resp.GetTimeToFirstByteMillis(),
			ContentTransfer: resp.GetContentTransferTimeMillis(),
			Total:           resp.GetTotalTimeMillis(),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Add content length if available
	if contentLength := resp.GetHeader("Content-Length"); contentLength != "" {
		var length int64
		fmt.Sscanf(contentLength, "%d", &length)
		data.ContentLength = length
	}

	// Marshal to JSON
	var output []byte
	if f.Pretty {
		output, err = json.MarshalIndent(data, "", "  ")
	} else {
		output, err = json.Marshal(data)
	}

	if err != nil {
		return fmt.Sprintf(`{"error":"Failed to marshal response: %s"}`, err)
	}

	return string(output)
}

// StartTest initializes a new test in the JSONFormatter
func (f *JSONFormatter) StartTest(name string) {
	if f.TestResults == nil {
		f.TestResults = &TestSuiteResult{
			Tests:     []TestResult{},
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}
	f.CurrentTest = &TestResult{
		Name:       name,
		Assertions: []AssertionResult{},
	}
}

// AddAssertion adds an assertion result to the current test
func (f *JSONFormatter) AddAssertion(assertion AssertionResult) {
	if f.CurrentTest != nil {
		f.CurrentTest.Assertions = append(f.CurrentTest.Assertions, assertion)
	}
}

// EndTest finalizes the current test and adds it to the suite results
func (f *JSONFormatter) EndTest(passed bool, duration int64) {
	if f.CurrentTest != nil {
		f.CurrentTest.Passed = passed
		f.CurrentTest.Duration = duration
		if f.TestResults != nil {
			f.TestResults.Tests = append(f.TestResults.Tests, *f.CurrentTest)
		}
		f.CurrentTest = nil
	}
}

// GetTestSuiteJSON returns the complete test suite results as JSON
func (f *JSONFormatter) GetTestSuiteJSON() string {
	if f.TestResults == nil {
		return "{}"
	}

	var output []byte
	var err error
	if f.Pretty {
		output, err = json.MarshalIndent(f.TestResults, "", "  ")
	} else {
		output, err = json.Marshal(f.TestResults)
	}

	if err != nil {
		return fmt.Sprintf(`{"error":"Failed to marshal test results: %s"}`, err)
	}

	return string(output)
}

// YAMLFormatter formats output as YAML
type YAMLFormatter struct {
	Verbose     bool
	TestResults *TestSuiteResult // Store test results for final output
	CurrentTest *TestResult      // Current test being executed
}

// FormatRequest formats a request as YAML
func (f *YAMLFormatter) FormatRequest(req *http.Request, baseURL string) string {
	// Build full URL
	fullURL := baseURL
	if !strings.HasSuffix(baseURL, "/") && !strings.HasPrefix(req.Path, "/") {
		fullURL += "/"
	}
	fullURL += req.Path
	if len(req.QueryParams) > 0 {
		fullURL += "?" + req.QueryParams.Encode()
	}

	// Convert query params to map for easier serialization
	queryParams := make(map[string]string)
	for key, values := range req.QueryParams {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}

	// Create request data structure
	data := RequestData{
		Method:      req.Method,
		URL:         fullURL,
		Headers:     req.Headers,
		QueryParams: queryParams,
		Body:        req.Body,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	// Marshal to YAML
	output, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Sprintf("error: Failed to marshal request: %s", err)
	}

	return string(output)
}

// FormatResponse formats a response as YAML
func (f *YAMLFormatter) FormatResponse(resp *http.Response) string {
	// Convert headers to map for easier serialization
	headers := make(map[string]string)
	for key, values := range resp.Headers {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Parse body if present
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

	// Create response data structure
	data := ResponseData{
		StatusCode:   resp.StatusCode,
		Status:       resp.Status,
		Headers:      headers,
		Body:         body,
		ResponseTime: resp.GetResponseTimeMillis(),
		Timing: TimingData{
			DNSLookup:       resp.GetDNSLookupTimeMillis(),
			TCPConnection:   resp.GetTCPConnectTimeMillis(),
			TLSHandshake:    resp.GetTLSHandshakeTimeMillis(),
			TimeToFirstByte: resp.GetTimeToFirstByteMillis(),
			ContentTransfer: resp.GetContentTransferTimeMillis(),
			Total:           resp.GetTotalTimeMillis(),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Add content length if available
	if contentLength := resp.GetHeader("Content-Length"); contentLength != "" {
		var length int64
		fmt.Sscanf(contentLength, "%d", &length)
		data.ContentLength = length
	}

	// Marshal to YAML
	output, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Sprintf("error: Failed to marshal response: %s", err)
	}

	return string(output)
}

// StartTest initializes a new test in the YAMLFormatter
func (f *YAMLFormatter) StartTest(name string) {
	if f.TestResults == nil {
		f.TestResults = &TestSuiteResult{
			Tests:     []TestResult{},
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}
	f.CurrentTest = &TestResult{
		Name:       name,
		Assertions: []AssertionResult{},
	}
}

// AddAssertion adds an assertion result to the current test
func (f *YAMLFormatter) AddAssertion(assertion AssertionResult) {
	if f.CurrentTest != nil {
		f.CurrentTest.Assertions = append(f.CurrentTest.Assertions, assertion)
	}
}

// EndTest finalizes the current test and adds it to the suite results
func (f *YAMLFormatter) EndTest(passed bool, duration int64) {
	if f.CurrentTest != nil {
		f.CurrentTest.Passed = passed
		f.CurrentTest.Duration = duration
		if f.TestResults != nil {
			f.TestResults.Tests = append(f.TestResults.Tests, *f.CurrentTest)
		}
		f.CurrentTest = nil
	}
}

// GetTestSuiteYAML returns the complete test suite results as YAML
func (f *YAMLFormatter) GetTestSuiteYAML() string {
	if f.TestResults == nil {
		return "---\n{}\n"
	}

	output, err := yaml.Marshal(f.TestResults)
	if err != nil {
		return fmt.Sprintf("---\nerror: Failed to marshal test results: %s\n", err)
	}

	return "---\n" + string(output)
}

// JUnitFormatter formats output as JUnit XML for CI/CD integration
type JUnitFormatter struct {
	Verbose     bool
	TestName    string
	SuiteName   string
	TestResults *JUnitTestSuites   // Store complete test results
	CurrentTest *JUnitTestCaseData // Current test being executed with full data
	StartTime   time.Time
}

// JUnitTestSuites represents the root element containing all test suites
type JUnitTestSuites struct {
	XMLName    xml.Name         `xml:"testsuites"`
	TestSuites []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite represents a JUnit test suite
type JUnitTestSuite struct {
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Time      float64         `xml:"time,attr"`
	Timestamp string          `xml:"timestamp,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
	SystemOut string          `xml:"system-out,omitempty"`
	SystemErr string          `xml:"system-err,omitempty"`
}

// JUnitTestCase represents a JUnit test case
type JUnitTestCase struct {
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Time      float64       `xml:"time,attr"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
	SystemOut string        `xml:"system-out,omitempty"`
}

// JUnitFailure represents a JUnit test failure
type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// JUnitTestCaseData holds complete test data during execution
type JUnitTestCaseData struct {
	Name       string
	StartTime  time.Time
	Duration   int64
	Passed     bool
	Assertions []AssertionResult
	Request    *RequestData
	Response   *ResponseData
}

// FormatRequest formats a request as JUnit XML
func (f *JUnitFormatter) FormatRequest(req *http.Request, baseURL string) string {
	// Build full URL for storage
	fullURL := baseURL
	if !strings.HasSuffix(baseURL, "/") && !strings.HasPrefix(req.Path, "/") {
		fullURL += "/"
	}
	fullURL += req.Path
	if len(req.QueryParams) > 0 {
		fullURL += "?" + req.QueryParams.Encode()
	}

	// Initialize current test if not already done
	if f.CurrentTest == nil {
		testName := f.TestName
		if testName == "" {
			testName = "TestRequest"
		}
		f.CurrentTest = &JUnitTestCaseData{
			Name:       testName,
			StartTime:  time.Now(),
			Assertions: []AssertionResult{},
		}
	}

	// Store request data
	queryParams := make(map[string]string)
	for key, values := range req.QueryParams {
		if len(values) > 0 {
			queryParams[key] = values[0]
		}
	}

	f.CurrentTest.Request = &RequestData{
		Method:      req.Method,
		URL:         fullURL,
		Headers:     req.Headers,
		QueryParams: queryParams,
		Body:        req.Body,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	// Return XML comment with request info for immediate output
	return fmt.Sprintf("<!-- Request: %s %s -->", req.Method, fullURL)
}

// FormatResponse formats a response as JUnit XML
func (f *JUnitFormatter) FormatResponse(resp *http.Response) string {
	// Initialize test results if not already done
	if f.TestResults == nil {
		f.TestResults = &JUnitTestSuites{
			TestSuites: []JUnitTestSuite{},
		}
	}

	// Store response data if we have a current test
	if f.CurrentTest != nil {
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

		f.CurrentTest.Response = &ResponseData{
			StatusCode:   resp.StatusCode,
			Status:       resp.Status,
			Headers:      headers,
			Body:         body,
			ResponseTime: resp.GetResponseTimeMillis(),
			Timing: TimingData{
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

	// Create or update test suite with current test
	testName := f.TestName
	if testName == "" {
		testName = "TestRequest"
	}

	// Find or create test suite
	var testSuite *JUnitTestSuite
	suiteName := f.SuiteName
	if suiteName == "" {
		suiteName = "TestSuite"
	}

	// Look for existing test suite
	for i := range f.TestResults.TestSuites {
		if f.TestResults.TestSuites[i].Name == suiteName {
			testSuite = &f.TestResults.TestSuites[i]
			break
		}
	}

	// Create new test suite if not found
	if testSuite == nil {
		f.TestResults.TestSuites = append(f.TestResults.TestSuites, JUnitTestSuite{
			Name:      suiteName,
			Tests:     0,
			Failures:  0,
			Errors:    0,
			Time:      0,
			Timestamp: time.Now().Format(time.RFC3339),
			TestCases: []JUnitTestCase{},
		})
		testSuite = &f.TestResults.TestSuites[len(f.TestResults.TestSuites)-1]
	}

	// Add test case
	testCase := JUnitTestCase{
		Name:      testName,
		Classname: suiteName,
		Time:      float64(resp.GetResponseTimeMillis()) / 1000.0,
	}

	// Check if test case already exists (for updating count)
	found := false
	for i, tc := range testSuite.TestCases {
		if tc.Name == testName {
			testSuite.TestCases[i] = testCase
			found = true
			break
		}
	}

	if !found {
		testSuite.TestCases = append(testSuite.TestCases, testCase)
		testSuite.Tests++
	}

	// Add timing information to system-out
	timingInfo := fmt.Sprintf("dnsLookup=\"%.2f\" tcpConnection=\"%.2f\" tlsHandshake=\"%.2f\" timeToFirstByte=\"%.2f\" contentTransfer=\"%.2f\"",
		float64(resp.GetDNSLookupTimeMillis())/1000.0,
		float64(resp.GetTCPConnectTimeMillis())/1000.0,
		float64(resp.GetTLSHandshakeTimeMillis())/1000.0,
		float64(resp.GetTimeToFirstByteMillis())/1000.0,
		float64(resp.GetContentTransferTimeMillis())/1000.0)

	testSuite.SystemOut = timingInfo

	// Generate and return XML
	output, err := xml.MarshalIndent(f.TestResults, "", "  ")
	if err != nil {
		return fmt.Sprintf("<!-- Error generating JUnit XML: %s -->", err)
	}

	return string(output)
}

// StartTest initializes a new test in the JUnitFormatter
func (f *JUnitFormatter) StartTest(name string) {
	if f.TestResults == nil {
		f.TestResults = &JUnitTestSuites{
			TestSuites: []JUnitTestSuite{},
		}
		f.StartTime = time.Now()
	}
	f.CurrentTest = &JUnitTestCaseData{
		Name:       name,
		StartTime:  time.Now(),
		Assertions: []AssertionResult{},
	}
}

// AddAssertion adds an assertion result to the current test
func (f *JUnitFormatter) AddAssertion(assertion AssertionResult) {
	if f.CurrentTest != nil {
		f.CurrentTest.Assertions = append(f.CurrentTest.Assertions, assertion)
	}
}

// EndTest finalizes the current test and stores it for later output
func (f *JUnitFormatter) EndTest(passed bool, duration int64) {
	if f.CurrentTest != nil {
		f.CurrentTest.Passed = passed
		f.CurrentTest.Duration = duration
		// Don't clear CurrentTest here - it will be used by the caller
	}
}

// GetTestSuiteXML returns the complete test suite results as JUnit XML
func (f *JUnitFormatter) GetTestSuiteXML() string {
	if f.TestResults == nil || len(f.TestResults.TestSuites) == 0 {
		// Create a default empty test suite
		emptyTestSuites := &JUnitTestSuites{
			TestSuites: []JUnitTestSuite{
				{
					Name:      f.SuiteName,
					Tests:     0,
					Failures:  0,
					Errors:    0,
					Time:      0,
					Timestamp: time.Now().Format(time.RFC3339),
					TestCases: []JUnitTestCase{},
				},
			},
		}
		output, _ := xml.MarshalIndent(emptyTestSuites, "", "  ")
		return xml.Header + string(output)
	}

	// Marshal to XML
	output, err := xml.MarshalIndent(f.TestResults, "", "  ")
	if err != nil {
		return fmt.Sprintf("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!-- Error: Failed to marshal test results: %s -->", err)
	}

	// Add XML header
	return xml.Header + string(output)
}

// SetTestSuite sets the test suite information with all test results
func (f *JUnitFormatter) SetTestSuite(suiteName string, tests []JUnitTestCaseData, totalDuration int64) {
	if f.TestResults == nil {
		f.TestResults = &JUnitTestSuites{
			TestSuites: []JUnitTestSuite{},
		}
	}

	suite := JUnitTestSuite{
		Name:      suiteName,
		Tests:     len(tests),
		Failures:  0,
		Errors:    0,
		Time:      float64(totalDuration) / 1000.0,
		Timestamp: f.StartTime.Format(time.RFC3339),
		TestCases: []JUnitTestCase{},
	}

	// Convert test data to JUnit test cases
	for _, test := range tests {
		testCase := JUnitTestCase{
			Name:      test.Name,
			Classname: "lunge." + suiteName,
			Time:      float64(test.Duration) / 1000.0,
		}

		// If test failed, create failure element
		if !test.Passed {
			failureMessages := []string{}
			for _, assertion := range test.Assertions {
				if !assertion.Passed {
					failureMessages = append(failureMessages, assertion.Message)
				}
			}

			testCase.Failure = &JUnitFailure{
				Message: fmt.Sprintf("Test failed with %d assertion failures", len(failureMessages)),
				Type:    "AssertionError",
				Content: strings.Join(failureMessages, "\n"),
			}
			suite.Failures++
		}

		// Add system output with request/response details if verbose
		if f.Verbose && test.Request != nil && test.Response != nil {
			var systemOut []string
			systemOut = append(systemOut, fmt.Sprintf("Request: %s %s", test.Request.Method, test.Request.URL))
			systemOut = append(systemOut, fmt.Sprintf("Response: %d %s", test.Response.StatusCode, test.Response.Status))
			systemOut = append(systemOut, fmt.Sprintf("Response Time: %dms", test.Response.ResponseTime))
			testCase.SystemOut = strings.Join(systemOut, "\n")
		}

		suite.TestCases = append(suite.TestCases, testCase)
	}

	f.TestResults.TestSuites = []JUnitTestSuite{suite}
}

// GetFormatter returns the appropriate formatter for the given format
func GetFormatter(format OutputFormat, verbose bool, noColor bool) FormatProvider {
	switch format {
	case FormatJSON:
		return &JSONFormatter{Verbose: verbose, Pretty: !noColor}
	case FormatYAML:
		return &YAMLFormatter{Verbose: verbose}
	case FormatJUnit:
		return &JUnitFormatter{
			Verbose:   verbose,
			SuiteName: "Lunge HTTP Tests",
		}
	default:
		// Default to text formatter (the original implementation)
		return &Formatter{Verbose: verbose, NoColor: noColor}
	}
}
