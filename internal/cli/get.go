package cli

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/wesleyorama2/lunge/internal/http"
	"github.com/wesleyorama2/lunge/internal/output"
)

var getCmd = &cobra.Command{
	Use:   "get URL",
	Short: "Make a GET request to the specified URL",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]
		headers, _ := cmd.Flags().GetStringArray("header")
		verbose, _ := cmd.Flags().GetBool("verbose")
		timeout, _ := cmd.Flags().GetDuration("timeout")
		noColor, _ := cmd.Flags().GetBool("no-color")

		// Parse URL to determine base URL and path
		baseURL, path := parseURL(url)

		// Create HTTP client
		client := http.NewClient(
			http.WithTimeout(timeout),
			http.WithBaseURL(baseURL),
		)

		// Create request
		req := http.NewRequest("GET", path)

		// Add headers
		for _, header := range headers {
			parts := strings.SplitN(header, ":", 2)
			if len(parts) == 2 {
				req.WithHeader(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}
		}

		// Create formatter
		formatter := output.NewFormatter(verbose, noColor)

		// Print request
		fmt.Print(formatter.FormatRequest(req, baseURL))

		// Execute request
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		resp, err := client.Do(ctx, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Print response
		fmt.Print(formatter.FormatResponse(resp))
	},
}

// parseURL splits a URL into base URL and path
func parseURL(fullURL string) (string, string) {
	// Add scheme if missing
	if !strings.HasPrefix(fullURL, "http://") && !strings.HasPrefix(fullURL, "https://") {
		fullURL = "http://" + fullURL
	}

	// Parse the URL
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		return fullURL, "/"
	}

	// Extract base URL and path
	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	// Include user info in the base URL if present
	if parsedURL.User != nil {
		userInfo := parsedURL.User.String()
		baseURL = fmt.Sprintf("%s://%s@%s", parsedURL.Scheme, userInfo, parsedURL.Host)
	}

	path := parsedURL.Path
	if path == "" {
		path = "/"
	}

	// Include query parameters in the path
	if parsedURL.RawQuery != "" {
		path = path + "?" + parsedURL.RawQuery
	}

	// Include fragment in the path
	if parsedURL.Fragment != "" {
		path = path + "#" + parsedURL.Fragment
	}

	return baseURL, path
}

func init() {
	// Add flags to GET command
	getCmd.Flags().StringArrayP("header", "H", []string{}, "HTTP headers to include (can be used multiple times)")
	getCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	getCmd.Flags().DurationP("timeout", "t", 30*time.Second, "Request timeout")
	getCmd.Flags().Bool("no-color", false, "Disable colored output")
}
