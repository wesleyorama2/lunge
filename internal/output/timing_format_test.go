package output

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	nethttp "net/http"

	"github.com/wesleyorama2/lunge/internal/http"
)

func TestFormatResponseWithTiming(t *testing.T) {
	// Create a response with detailed timing information
	headers := make(nethttp.Header)
	headers.Set("Content-Type", "application/json")

	resp := &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    headers,
		Body:       nethttp.NoBody,
		Timing: http.TimingInfo{
			DNSLookupTime:       10 * time.Millisecond,
			TCPConnectTime:      20 * time.Millisecond,
			TLSHandshakeTime:    30 * time.Millisecond,
			TimeToFirstByte:     40 * time.Millisecond,
			ContentTransferTime: 50 * time.Millisecond,
			TotalTime:           150 * time.Millisecond,
		},
		ResponseTime: 150 * time.Millisecond,
	}

	// Test text formatter
	textFormatter := NewFormatter(true, true) // verbose, no color
	textOutput := textFormatter.FormatResponse(resp)

	// Check that timing information is included
	expectedTimingParts := []string{
		"DNS Lookup:      10ms",
		"TCP Connection:  20ms",
		"TLS Handshake:   30ms",
		"Time to First Byte: 40ms",
		"Content Transfer:  50ms",
		"Total:           150ms",
	}

	for _, part := range expectedTimingParts {
		if !strings.Contains(textOutput, part) {
			t.Errorf("Text formatter output missing timing info: %s", part)
		}
	}

	// Test JSON formatter
	jsonFormatter := &JSONFormatter{Verbose: true, Pretty: true}
	jsonOutput := jsonFormatter.FormatResponse(resp)

	// Parse JSON output
	var jsonData map[string]interface{}
	err := json.Unmarshal([]byte(jsonOutput), &jsonData)
	if err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Check timing information
	timing, ok := jsonData["timing"].(map[string]interface{})
	if !ok {
		t.Fatalf("JSON output missing timing information")
	}

	expectedTimingValues := map[string]float64{
		"dnsLookupMs":       10,
		"tcpConnectionMs":   20,
		"tlsHandshakeMs":    30,
		"timeToFirstByteMs": 40,
		"contentTransferMs": 50,
		"totalMs":           150,
	}

	for key, expectedValue := range expectedTimingValues {
		value, ok := timing[key].(float64)
		if !ok {
			t.Errorf("JSON output missing timing field: %s", key)
			continue
		}
		if value != expectedValue {
			t.Errorf("JSON output timing field %s: expected %.0f, got %.0f", key, expectedValue, value)
		}
	}

	// Test YAML formatter
	yamlFormatter := &YAMLFormatter{Verbose: true}
	yamlOutput := yamlFormatter.FormatResponse(resp)

	// Check that timing information is included in YAML output
	if !strings.Contains(yamlOutput, "timing:") {
		t.Errorf("YAML output missing timing section")
	}

	for key := range expectedTimingValues {
		if !strings.Contains(yamlOutput, key+":") {
			t.Errorf("YAML output missing timing key: %s", key)
		}
	}

	// Test JUnit formatter
	junitFormatter := &JUnitFormatter{Verbose: true, TestName: "TestRequest"}
	junitOutput := junitFormatter.FormatResponse(resp)

	// Check that timing information is included in JUnit output
	expectedJUnitParts := []string{
		`dnsLookup="0.01"`,
		`tcpConnection="0.02"`,
		`tlsHandshake="0.03"`,
		`timeToFirstByte="0.04"`,
		`contentTransfer="0.05"`,
	}

	for _, part := range expectedJUnitParts {
		if !strings.Contains(junitOutput, part) {
			t.Errorf("JUnit formatter output missing timing info: %s", part)
		}
	}
}
