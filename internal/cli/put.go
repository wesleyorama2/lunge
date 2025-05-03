// Copyright (c) 2025, Wesley Brown
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/wesleyorama2/lunge/internal/http"
	"github.com/wesleyorama2/lunge/internal/output"
)

var putCmd = &cobra.Command{
	Use:   "put URL",
	Short: "Make a PUT request to the specified URL",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]
		headers, _ := cmd.Flags().GetStringArray("header")
		data, _ := cmd.Flags().GetString("data")
		jsonData, _ := cmd.Flags().GetString("json")
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
		req := http.NewRequest("PUT", path)

		// Add headers
		for _, header := range headers {
			parts := strings.SplitN(header, ":", 2)
			if len(parts) == 2 {
				req.WithHeader(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}
		}

		// Add body
		if data != "" {
			req.WithBody(data)
		} else if jsonData != "" {
			req.WithBody(jsonData)
			if req.Headers["Content-Type"] == "" {
				req.WithHeader("Content-Type", "application/json")
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

func init() {
	// Add flags to PUT command
	putCmd.Flags().StringArrayP("header", "H", []string{}, "HTTP headers to include (can be used multiple times)")
	putCmd.Flags().StringP("data", "d", "", "Data to send in the request body")
	putCmd.Flags().StringP("json", "j", "", "JSON data to send in the request body")
	putCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	putCmd.Flags().DurationP("timeout", "t", 30*time.Second, "Request timeout")
	putCmd.Flags().Bool("no-color", false, "Disable colored output")
}
