package http

import (
	"testing"
	"time"
)

func TestTimingInfo(t *testing.T) {
	// Create a response with timing information
	resp := &Response{
		StatusCode: 200,
		Status:     "200 OK",
		Timing: TimingInfo{
			DNSLookupTime:       10 * time.Millisecond,
			TCPConnectTime:      20 * time.Millisecond,
			TLSHandshakeTime:    30 * time.Millisecond,
			TimeToFirstByte:     40 * time.Millisecond,
			ContentTransferTime: 50 * time.Millisecond,
			TotalTime:           150 * time.Millisecond,
		},
		ResponseTime: 150 * time.Millisecond,
	}

	// Test that the timing information is correctly accessible
	if resp.GetDNSLookupTimeMillis() != 10 {
		t.Errorf("Expected DNS lookup time to be 10ms, got %dms", resp.GetDNSLookupTimeMillis())
	}

	if resp.GetTCPConnectTimeMillis() != 20 {
		t.Errorf("Expected TCP connect time to be 20ms, got %dms", resp.GetTCPConnectTimeMillis())
	}

	if resp.GetTLSHandshakeTimeMillis() != 30 {
		t.Errorf("Expected TLS handshake time to be 30ms, got %dms", resp.GetTLSHandshakeTimeMillis())
	}

	if resp.GetTimeToFirstByteMillis() != 40 {
		t.Errorf("Expected time to first byte to be 40ms, got %dms", resp.GetTimeToFirstByteMillis())
	}

	if resp.GetContentTransferTimeMillis() != 50 {
		t.Errorf("Expected content transfer time to be 50ms, got %dms", resp.GetContentTransferTimeMillis())
	}

	if resp.GetTotalTimeMillis() != 150 {
		t.Errorf("Expected total time to be 150ms, got %dms", resp.GetTotalTimeMillis())
	}

	// Test backward compatibility
	if resp.GetResponseTimeMillis() != 150 {
		t.Errorf("Expected response time to be 150ms, got %dms", resp.GetResponseTimeMillis())
	}
}

func TestTimingInfoZeroValues(t *testing.T) {
	// Create a response with zero timing information
	resp := &Response{
		StatusCode: 200,
		Status:     "200 OK",
		Timing:     TimingInfo{},
	}

	// Test that zero values are handled correctly
	if resp.GetDNSLookupTimeMillis() != 0 {
		t.Errorf("Expected DNS lookup time to be 0ms, got %dms", resp.GetDNSLookupTimeMillis())
	}

	if resp.GetTCPConnectTimeMillis() != 0 {
		t.Errorf("Expected TCP connect time to be 0ms, got %dms", resp.GetTCPConnectTimeMillis())
	}

	if resp.GetTLSHandshakeTimeMillis() != 0 {
		t.Errorf("Expected TLS handshake time to be 0ms, got %dms", resp.GetTLSHandshakeTimeMillis())
	}

	if resp.GetTimeToFirstByteMillis() != 0 {
		t.Errorf("Expected time to first byte to be 0ms, got %dms", resp.GetTimeToFirstByteMillis())
	}

	if resp.GetContentTransferTimeMillis() != 0 {
		t.Errorf("Expected content transfer time to be 0ms, got %dms", resp.GetContentTransferTimeMillis())
	}

	if resp.GetTotalTimeMillis() != 0 {
		t.Errorf("Expected total time to be 0ms, got %dms", resp.GetTotalTimeMillis())
	}
}
