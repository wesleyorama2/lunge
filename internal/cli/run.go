package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"lunge/internal/config"
	"lunge/internal/http"
	"lunge/internal/output"
	"lunge/pkg/jsonpath"
	"lunge/pkg/jsonschema"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run requests or suites from a configuration file",
	Run: func(cmd *cobra.Command, args []string) {
		configFile, _ := cmd.Flags().GetString("config")
		environment, _ := cmd.Flags().GetString("environment")
		request, _ := cmd.Flags().GetString("request")
		suite, _ := cmd.Flags().GetString("suite")
		verbose, _ := cmd.Flags().GetBool("verbose")
		timeout, _ := cmd.Flags().GetDuration("timeout")
		noColor, _ := cmd.Flags().GetBool("no-color")

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

		if request == "" && suite == "" {
			fmt.Println("Error: either request or suite is required")
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

		// Create formatter
		formatter := output.NewFormatter(verbose, noColor)

		// Create HTTP client
		client := http.NewClient(
			http.WithTimeout(timeout),
		)

		// Get environment
		env := cfg.Environments[environment]
		envVars := env.Vars

		if request != "" {
			// Validate request
			if err := config.ValidateRequest(cfg, request); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Execute single request
			executeRequest(cfg, request, env, envVars, client, formatter, timeout, verbose)
		} else if suite != "" {
			// Validate suite
			if err := config.ValidateSuite(cfg, suite); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Execute suite
			executeSuite(cfg, suite, env, envVars, client, formatter, timeout, verbose)
		}
	},
}

// executeRequest executes a single request
func executeRequest(cfg *config.Config, requestName string, env config.Environment, envVars map[string]string, client *http.Client, formatter *output.Formatter, timeout time.Duration, verbose bool) {
	// Get request
	reqConfig := cfg.Requests[requestName]

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
		// Process body based on type
		switch body := reqConfig.Body.(type) {
		case map[string]interface{}:
			// Process each field in the map
			processedBody := make(map[string]interface{})
			for k, v := range body {
				if strValue, ok := v.(string); ok {
					// Process string values for variable substitution
					processedBody[k] = config.ProcessEnvironment(strValue, envVars)
				} else {
					// Keep non-string values as is
					processedBody[k] = v
				}
			}
			req.WithBody(processedBody)
		case string:
			// Process string body
			processedBody := config.ProcessEnvironment(body, envVars)
			req.WithBody(processedBody)
		default:
			// Use body as is for other types
			req.WithBody(reqConfig.Body)
		}
	}

	// Print request
	fmt.Print(formatter.FormatRequest(req, baseURL))

	// Execute request
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Update client with baseURL
	client = http.NewClient(
		http.WithTimeout(timeout),
		http.WithBaseURL(baseURL),
	)

	resp, err := client.Do(ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print response
	fmt.Print(formatter.FormatResponse(resp))

	// Extract variables
	if reqConfig.Extract != nil && len(reqConfig.Extract) > 0 {
		// Get response body as string
		body, err := resp.GetBodyAsString()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading response body for variable extraction: %v\n", err)
		} else {
			// Extract variables
			extracted, err := jsonpath.ExtractMultiple(body, reqConfig.Extract)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Variable extraction partial or failed: %v\n", err)
			}

			// Add extracted variables to environment
			for name, value := range extracted {
				envVars[name] = value
				if verbose {
					fmt.Printf("Extracted variable %s = %s\n", name, value)
				}
			}
		}
	}

	// Validate response against JSON Schema if specified
	if reqConfig.Validate != nil && len(reqConfig.Validate) > 0 {
		// Get response body as string
		body, err := resp.GetBodyAsString()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading response body for schema validation: %v\n", err)
		} else {
			// Convert the validate map to a JSON schema string
			schemaBytes, err := json.Marshal(reqConfig.Validate)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshaling schema for validation: %v\n", err)
			} else {
				// Validate the response body against the schema
				valid, validationErrors := jsonschema.ValidateWithErrors(body, string(schemaBytes))
				if !valid {
					fmt.Fprintf(os.Stderr, "%s Schema validation failed: %v\n", output.ErrorIcon(false), validationErrors)
				} else if verbose {
					fmt.Printf("%s Schema validation passed\n", output.SuccessIcon(false))
				}
			}
		}
	}
}

// executeSuite executes a suite of requests
func executeSuite(cfg *config.Config, suiteName string, env config.Environment, envVars map[string]string, client *http.Client, formatter *output.Formatter, timeout time.Duration, verbose bool) {
	// Get suite
	suite := cfg.Suites[suiteName]

	// Merge suite variables with environment variables
	if suite.Vars != nil {
		for key, value := range suite.Vars {
			envVars[key] = config.ProcessEnvironment(value, envVars)
		}
	}

	// Execute requests in order
	for _, requestName := range suite.Requests {
		fmt.Printf("\n=== Executing request: %s ===\n\n", requestName)
		executeRequest(cfg, requestName, env, envVars, client, formatter, timeout, verbose)
	}
}

// isAbsoluteURL checks if a URL is absolute (has a scheme and host)
func isAbsoluteURL(url string) bool {
	return len(url) > 0 && ((len(url) >= 7 && url[0:7] == "http://") ||
		(len(url) >= 8 && url[0:8] == "https://"))
}

func init() {
	// Add flags to RUN command
	runCmd.Flags().StringP("config", "c", "", "Configuration file (required)")
	runCmd.Flags().StringP("environment", "e", "", "Environment to use (required)")
	runCmd.Flags().StringP("request", "r", "", "Request to run")
	runCmd.Flags().StringP("suite", "s", "", "Suite to run")
	runCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	runCmd.Flags().DurationP("timeout", "t", 30*time.Second, "Request timeout")
	runCmd.Flags().Bool("no-color", false, "Disable colored output")
}
