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

// JSONFormatter formats output as JSON
type JSONFormatter struct {
	Verbose bool
	Pretty  bool
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

// YAMLFormatter formats output as YAML
type YAMLFormatter struct {
	Verbose bool
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

// JUnitFormatter formats output as JUnit XML for CI/CD integration
type JUnitFormatter struct {
	Verbose     bool
	TestName    string
	SuiteName   string
	TestCases   []JUnitTestCase
	CurrentTest *JUnitTestCase
}

// JUnitTestSuite represents a JUnit test suite
type JUnitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Time      float64         `xml:"time,attr"`
	Timestamp string          `xml:"timestamp,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents a JUnit test case
type JUnitTestCase struct {
	Name            string        `xml:"name,attr"`
	Classname       string        `xml:"classname,attr"`
	Time            float64       `xml:"time,attr"`
	DNSLookup       float64       `xml:"dnsLookup,attr,omitempty"`
	TCPConnection   float64       `xml:"tcpConnection,attr,omitempty"`
	TLSHandshake    float64       `xml:"tlsHandshake,attr,omitempty"`
	TimeToFirstByte float64       `xml:"timeToFirstByte,attr,omitempty"`
	ContentTransfer float64       `xml:"contentTransfer,attr,omitempty"`
	Failure         *JUnitFailure `xml:"failure,omitempty"`
}

// JUnitFailure represents a JUnit test failure
type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// FormatRequest formats a request as JUnit XML
// Note: JUnit format is primarily for responses, so this returns minimal info
func (f *JUnitFormatter) FormatRequest(req *http.Request, baseURL string) string {
	// Build full URL
	fullURL := baseURL
	if !strings.HasSuffix(baseURL, "/") && !strings.HasPrefix(req.Path, "/") {
		fullURL += "/"
	}
	fullURL += req.Path
	if len(req.QueryParams) > 0 {
		fullURL += "?" + req.QueryParams.Encode()
	}

	// For requests, we just return a simple XML comment
	return fmt.Sprintf("<!-- Request: %s %s -->", req.Method, fullURL)
}

// FormatResponse formats a response as JUnit XML
func (f *JUnitFormatter) FormatResponse(resp *http.Response) string {
	testName := f.TestName
	if testName == "" {
		testName = "HTTP Request"
	}

	// Create test case
	testCase := JUnitTestCase{
		Name:            testName,
		Classname:       "lunge.http",
		Time:            float64(resp.GetResponseTimeMillis()) / 1000.0,
		DNSLookup:       float64(resp.GetDNSLookupTimeMillis()) / 1000.0,
		TCPConnection:   float64(resp.GetTCPConnectTimeMillis()) / 1000.0,
		TLSHandshake:    float64(resp.GetTLSHandshakeTimeMillis()) / 1000.0,
		TimeToFirstByte: float64(resp.GetTimeToFirstByteMillis()) / 1000.0,
		ContentTransfer: float64(resp.GetContentTransferTimeMillis()) / 1000.0,
	}

	// Add failure if response is not successful
	if !resp.IsSuccess() {
		bodyStr, _ := resp.GetBodyAsString()
		testCase.Failure = &JUnitFailure{
			Message: fmt.Sprintf("HTTP Status %d: %s", resp.StatusCode, resp.Status),
			Type:    "HttpStatusError",
			Content: bodyStr,
		}
	}

	// Store the current test case
	f.CurrentTest = &testCase
	f.TestCases = append(f.TestCases, testCase)

	// Create test suite with all test cases
	suiteName := f.SuiteName
	if suiteName == "" {
		suiteName = "Lunge HTTP Tests"
	}

	suite := JUnitTestSuite{
		Name:      suiteName,
		Tests:     len(f.TestCases),
		Failures:  0,
		Errors:    0,
		Time:      float64(resp.GetResponseTimeMillis()) / 1000.0, // This will be updated with total time
		Timestamp: time.Now().Format(time.RFC3339),
		TestCases: f.TestCases,
	}

	// Count failures
	for _, tc := range f.TestCases {
		if tc.Failure != nil {
			suite.Failures++
		}
	}

	// Marshal to XML
	output, err := xml.MarshalIndent(suite, "", "  ")
	if err != nil {
		return fmt.Sprintf("<!-- Error: Failed to marshal response: %s -->", err)
	}

	// Add XML header
	return xml.Header + string(output)
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
			TestCases: make([]JUnitTestCase, 0),
			SuiteName: "Lunge HTTP Tests",
		}
	default:
		// Default to text formatter (the original implementation)
		return &Formatter{Verbose: verbose, NoColor: noColor}
	}
}
