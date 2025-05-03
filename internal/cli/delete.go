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

var deleteCmd = &cobra.Command{
	Use:   "delete URL",
	Short: "Make a DELETE request to the specified URL",
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
		req := http.NewRequest("DELETE", path)

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

func init() {
	// Add flags to DELETE command
	deleteCmd.Flags().StringArrayP("header", "H", []string{}, "HTTP headers to include (can be used multiple times)")
	deleteCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	deleteCmd.Flags().DurationP("timeout", "t", 30*time.Second, "Request timeout")
	deleteCmd.Flags().Bool("no-color", false, "Disable colored output")
}
