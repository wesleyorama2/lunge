package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	http "github.com/wesleyorama2/lunge/internal/http"

	"github.com/fatih/color"
)

// Formatter is responsible for formatting HTTP requests and responses in text format
type Formatter struct {
	Verbose bool
	NoColor bool
}

// NewFormatter creates a new formatter with the given options
func NewFormatter(verbose, noColor bool) *Formatter {
	return &Formatter{
		Verbose: verbose,
		NoColor: noColor,
	}
}

// NewFormatterWithFormat creates a new formatter with the specified output format
func NewFormatterWithFormat(format OutputFormat, verbose, noColor bool) FormatProvider {
	return GetFormatter(format, verbose, noColor)
}

// FormatRequest formats an HTTP request for display
func (f *Formatter) FormatRequest(req *http.Request, baseURL string) string {
	var buf strings.Builder

	// Format method and URL
	methodColor := color.New(color.FgBlue, color.Bold)
	if f.NoColor {
		methodColor.DisableColor()
	}

	fullURL := baseURL
	if !strings.HasSuffix(baseURL, "/") && !strings.HasPrefix(req.Path, "/") {
		fullURL += "/"
	}
	fullURL += req.Path

	// Add query parameters if any
	if len(req.QueryParams) > 0 {
		fullURL += "?" + req.QueryParams.Encode()
	}

	buf.WriteString(fmt.Sprintf("▶ REQUEST: %s %s\n", methodColor.Sprint(req.Method), fullURL))

	// Format headers if verbose or if there are headers
	if f.Verbose || len(req.Headers) > 0 {
		buf.WriteString("  Headers:\n")
		for key, value := range req.Headers {
			buf.WriteString(fmt.Sprintf("    %s: %s\n", key, value))
		}
	}

	// Format body if present
	if req.Body != nil {
		buf.WriteString("  Body: ")
		switch body := req.Body.(type) {
		case string:
			buf.WriteString(formatJSONString(body))
		case []byte:
			buf.WriteString(formatJSONString(string(body)))
		default:
			// Try to marshal as JSON
			jsonBody, err := json.Marshal(body)
			if err != nil {
				buf.WriteString(fmt.Sprintf("%v", body))
			} else {
				buf.WriteString(formatJSONString(string(jsonBody)))
			}
		}
		buf.WriteString("\n")
	}

	return buf.String()
}

// FormatResponse formats an HTTP response for display
func (f *Formatter) FormatResponse(resp *http.Response) string {
	var buf strings.Builder

	// Format status
	statusColor := color.New(color.Bold)
	if resp.IsSuccess() {
		statusColor.Add(color.FgGreen)
	} else if resp.IsRedirect() {
		statusColor.Add(color.FgYellow)
	} else {
		statusColor.Add(color.FgRed)
	}

	if f.NoColor {
		statusColor.DisableColor()
	}

	buf.WriteString(fmt.Sprintf("◀ RESPONSE: %s (%dms)\n",
		statusColor.Sprint(resp.Status),
		resp.GetResponseTimeMillis()))

	// Format detailed timing information if verbose
	if f.Verbose {
		buf.WriteString("  Timing:\n")
		buf.WriteString(fmt.Sprintf("    DNS Lookup:      %dms\n", resp.GetDNSLookupTimeMillis()))
		buf.WriteString(fmt.Sprintf("    TCP Connection:  %dms\n", resp.GetTCPConnectTimeMillis()))
		buf.WriteString(fmt.Sprintf("    TLS Handshake:   %dms\n", resp.GetTLSHandshakeTimeMillis()))
		buf.WriteString(fmt.Sprintf("    Time to First Byte: %dms\n", resp.GetTimeToFirstByteMillis()))
		buf.WriteString(fmt.Sprintf("    Content Transfer:  %dms\n", resp.GetContentTransferTimeMillis()))
		buf.WriteString(fmt.Sprintf("    Total:           %dms\n", resp.GetTotalTimeMillis()))
	}

	// Format headers if verbose
	if f.Verbose {
		buf.WriteString("  Headers:\n")
		for key, values := range resp.Headers {
			for _, value := range values {
				buf.WriteString(fmt.Sprintf("    %s: %s\n", key, value))
			}
		}
	}

	// Format body
	body, err := resp.GetBodyAsString()
	if err == nil && body != "" {
		buf.WriteString("  Body:\n")
		buf.WriteString(formatJSONString(body))
		buf.WriteString("\n")
	}

	return buf.String()
}

// formatJSONString attempts to pretty-print a JSON string
func formatJSONString(s string) string {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, []byte(s), "  ", "  ")
	if err != nil {
		return s
	}
	return prettyJSON.String()
}
