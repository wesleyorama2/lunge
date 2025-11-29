package config

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// ValidationErrors is a collection of validation errors.
type ValidationErrors struct {
	Errors []*ValidationError
}

func (e *ValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return "no validation errors"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d validation errors:\n", len(e.Errors)))
	for i, err := range e.Errors {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	return sb.String()
}

// Add adds an error to the collection.
func (e *ValidationErrors) Add(field, message string) {
	e.Errors = append(e.Errors, &ValidationError{Field: field, Message: message})
}

// HasErrors returns true if there are any errors.
func (e *ValidationErrors) HasErrors() bool {
	return len(e.Errors) > 0
}

// Validate validates the entire test configuration.
//
// Returns nil if valid, or a ValidationErrors containing all validation errors.
func (c *TestConfig) Validate() error {
	errs := &ValidationErrors{}

	// Validate scenarios
	if len(c.Scenarios) == 0 {
		errs.Add("scenarios", "at least one scenario is required")
	}

	for name, scenario := range c.Scenarios {
		validateScenario(name, scenario, &c.Settings, errs)
	}

	// Validate thresholds
	if c.Thresholds != nil {
		validateThresholds(c.Thresholds, errs)
	}

	// Validate settings
	validateSettings(&c.Settings, errs)

	if errs.HasErrors() {
		return errs
	}
	return nil
}

// validateScenario validates a single scenario configuration.
func validateScenario(name string, sc *ScenarioConfig, settings *GlobalSettings, errs *ValidationErrors) {
	prefix := fmt.Sprintf("scenarios.%s", name)

	// Validate executor type
	validExecutors := map[string]bool{
		"constant-vus":          true,
		"ramping-vus":           true,
		"constant-arrival-rate": true,
		"ramping-arrival-rate":  true,
		"per-vu-iterations":     true,
		"shared-iterations":     true,
	}

	if sc.Executor == "" {
		errs.Add(prefix+".executor", "executor type is required")
	} else if !validExecutors[sc.Executor] {
		errs.Add(prefix+".executor", fmt.Sprintf("unknown executor type: %s", sc.Executor))
	}

	// Executor-specific validation
	switch sc.Executor {
	case "constant-vus":
		validateConstantVUs(prefix, sc, errs)
	case "ramping-vus":
		validateRampingVUs(prefix, sc, errs)
	case "constant-arrival-rate":
		validateConstantArrivalRate(prefix, sc, errs)
	case "ramping-arrival-rate":
		validateRampingArrivalRate(prefix, sc, errs)
	case "per-vu-iterations", "shared-iterations":
		validateIterationBased(prefix, sc, errs)
	}

	// Validate requests
	if len(sc.Requests) == 0 {
		errs.Add(prefix+".requests", "at least one request is required")
	}

	for i, req := range sc.Requests {
		validateRequest(fmt.Sprintf("%s.requests[%d]", prefix, i), &req, settings, errs)
	}

	// Validate pacing
	if sc.Pacing != nil {
		validatePacing(prefix+".pacing", sc.Pacing, errs)
	}

	// Validate stages
	for i, stage := range sc.Stages {
		validateStage(fmt.Sprintf("%s.stages[%d]", prefix, i), &stage, errs)
	}
}

// validateConstantVUs validates constant-vus executor config.
func validateConstantVUs(prefix string, sc *ScenarioConfig, errs *ValidationErrors) {
	if sc.VUs <= 0 {
		errs.Add(prefix+".vus", "vus must be greater than 0")
	}

	if sc.Duration == "" {
		errs.Add(prefix+".duration", "duration is required for constant-vus executor")
	} else {
		if _, err := ParseDurationString(sc.Duration); err != nil {
			errs.Add(prefix+".duration", fmt.Sprintf("invalid duration: %v", err))
		}
	}
}

// validateRampingVUs validates ramping-vus executor config.
func validateRampingVUs(prefix string, sc *ScenarioConfig, errs *ValidationErrors) {
	if len(sc.Stages) == 0 {
		errs.Add(prefix+".stages", "at least one stage is required for ramping-vus executor")
	}
}

// validateConstantArrivalRate validates constant-arrival-rate executor config.
func validateConstantArrivalRate(prefix string, sc *ScenarioConfig, errs *ValidationErrors) {
	if sc.Rate <= 0 {
		errs.Add(prefix+".rate", "rate must be greater than 0")
	}

	if sc.Duration == "" {
		errs.Add(prefix+".duration", "duration is required for constant-arrival-rate executor")
	} else {
		if _, err := ParseDurationString(sc.Duration); err != nil {
			errs.Add(prefix+".duration", fmt.Sprintf("invalid duration: %v", err))
		}
	}

	// Validate pre-allocated VUs
	if sc.PreAllocatedVUs < 0 {
		errs.Add(prefix+".preAllocatedVUs", "preAllocatedVUs cannot be negative")
	}

	// Validate max VUs
	if sc.MaxVUs > 0 && sc.PreAllocatedVUs > sc.MaxVUs {
		errs.Add(prefix+".preAllocatedVUs", "preAllocatedVUs cannot be greater than maxVUs")
	}
}

// validateRampingArrivalRate validates ramping-arrival-rate executor config.
func validateRampingArrivalRate(prefix string, sc *ScenarioConfig, errs *ValidationErrors) {
	if len(sc.Stages) == 0 {
		errs.Add(prefix+".stages", "at least one stage is required for ramping-arrival-rate executor")
	}

	// Validate pre-allocated VUs
	if sc.PreAllocatedVUs < 0 {
		errs.Add(prefix+".preAllocatedVUs", "preAllocatedVUs cannot be negative")
	}

	// Validate max VUs
	if sc.MaxVUs > 0 && sc.PreAllocatedVUs > sc.MaxVUs {
		errs.Add(prefix+".preAllocatedVUs", "preAllocatedVUs cannot be greater than maxVUs")
	}
}

// validateIterationBased validates per-vu-iterations and shared-iterations executor config.
func validateIterationBased(prefix string, sc *ScenarioConfig, errs *ValidationErrors) {
	if sc.VUs <= 0 {
		errs.Add(prefix+".vus", "vus must be greater than 0")
	}

	// Note: iterations would need to be added to ScenarioConfig if these executors are fully implemented
}

// validateRequest validates a single request configuration.
func validateRequest(prefix string, req *RequestConfig, settings *GlobalSettings, errs *ValidationErrors) {
	// Validate method
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true,
		"PATCH": true, "HEAD": true, "OPTIONS": true,
	}

	method := strings.ToUpper(req.Method)
	if method == "" {
		errs.Add(prefix+".method", "method is required")
	} else if !validMethods[method] {
		errs.Add(prefix+".method", fmt.Sprintf("invalid HTTP method: %s", req.Method))
	}

	// Validate URL
	if req.URL == "" {
		errs.Add(prefix+".url", "url is required")
	} else {
		// Check if URL is valid (allowing for variable placeholders)
		urlToCheck := req.URL
		// Replace common placeholders for URL validation
		urlToCheck = strings.ReplaceAll(urlToCheck, "{{baseUrl}}", "http://example.com")
		urlToCheck = strings.ReplaceAll(urlToCheck, "{{baseURL}}", "http://example.com")
		// Replace any remaining {{var}} patterns
		for strings.Contains(urlToCheck, "{{") {
			start := strings.Index(urlToCheck, "{{")
			end := strings.Index(urlToCheck, "}}")
			if end > start {
				urlToCheck = urlToCheck[:start] + "placeholder" + urlToCheck[end+2:]
			} else {
				break
			}
		}

		if _, err := url.Parse(urlToCheck); err != nil {
			errs.Add(prefix+".url", fmt.Sprintf("invalid URL: %v", err))
		}
	}

	// Validate timeout if specified
	if req.Timeout != "" {
		if _, err := ParseDurationString(req.Timeout); err != nil {
			errs.Add(prefix+".timeout", fmt.Sprintf("invalid timeout: %v", err))
		}
	}

	// Validate think time if specified
	if req.ThinkTime != "" {
		if _, err := ParseDurationString(req.ThinkTime); err != nil {
			errs.Add(prefix+".thinkTime", fmt.Sprintf("invalid thinkTime: %v", err))
		}
	}

	// Validate extract configs
	for i, extract := range req.Extract {
		validateExtract(fmt.Sprintf("%s.extract[%d]", prefix, i), &extract, errs)
	}

	// Validate assertions
	for i, assertion := range req.Assertions {
		validateAssertion(fmt.Sprintf("%s.assertions[%d]", prefix, i), &assertion, errs)
	}
}

// validatePacing validates pacing configuration.
func validatePacing(prefix string, pacing *PacingConfig, errs *ValidationErrors) {
	validTypes := map[string]bool{
		"none": true, "constant": true, "random": true,
	}

	if !validTypes[pacing.Type] {
		errs.Add(prefix+".type", fmt.Sprintf("invalid pacing type: %s", pacing.Type))
	}

	switch pacing.Type {
	case "constant":
		if pacing.Duration == "" {
			errs.Add(prefix+".duration", "duration is required for constant pacing")
		} else if _, err := ParseDurationString(pacing.Duration); err != nil {
			errs.Add(prefix+".duration", fmt.Sprintf("invalid duration: %v", err))
		}

	case "random":
		if pacing.Min == "" {
			errs.Add(prefix+".min", "min is required for random pacing")
		} else if _, err := ParseDurationString(pacing.Min); err != nil {
			errs.Add(prefix+".min", fmt.Sprintf("invalid min: %v", err))
		}

		if pacing.Max == "" {
			errs.Add(prefix+".max", "max is required for random pacing")
		} else if _, err := ParseDurationString(pacing.Max); err != nil {
			errs.Add(prefix+".max", fmt.Sprintf("invalid max: %v", err))
		}

		// Validate min < max
		if pacing.Min != "" && pacing.Max != "" {
			minDur, _ := ParseDurationString(pacing.Min)
			maxDur, _ := ParseDurationString(pacing.Max)
			if minDur > maxDur {
				errs.Add(prefix, "min must be less than or equal to max")
			}
		}
	}
}

// validateStage validates a single stage configuration.
func validateStage(prefix string, stage *StageConfig, errs *ValidationErrors) {
	if stage.Duration == "" {
		errs.Add(prefix+".duration", "duration is required")
	} else if _, err := ParseDurationString(stage.Duration); err != nil {
		errs.Add(prefix+".duration", fmt.Sprintf("invalid duration: %v", err))
	}

	if stage.Target < 0 {
		errs.Add(prefix+".target", "target cannot be negative")
	}
}

// validateExtract validates an extract configuration.
func validateExtract(prefix string, extract *ExtractConfig, errs *ValidationErrors) {
	if extract.Name == "" {
		errs.Add(prefix+".name", "name is required")
	}

	validSources := map[string]bool{
		"body": true, "header": true, "status": true,
	}

	if extract.Source == "" {
		errs.Add(prefix+".source", "source is required")
	} else if !validSources[extract.Source] {
		errs.Add(prefix+".source", fmt.Sprintf("invalid source: %s", extract.Source))
	}
}

// validateAssertion validates an assertion configuration.
func validateAssertion(prefix string, assertion *AssertionConfig, errs *ValidationErrors) {
	validTypes := map[string]bool{
		"status": true, "body": true, "header": true, "duration": true,
	}

	if assertion.Type == "" {
		errs.Add(prefix+".type", "type is required")
	} else if !validTypes[assertion.Type] {
		errs.Add(prefix+".type", fmt.Sprintf("invalid assertion type: %s", assertion.Type))
	}

	validConditions := map[string]bool{
		"eq": true, "ne": true, "gt": true, "lt": true,
		"gte": true, "lte": true, "contains": true, "matches": true,
	}

	if assertion.Condition == "" {
		errs.Add(prefix+".condition", "condition is required")
	} else if !validConditions[assertion.Condition] {
		errs.Add(prefix+".condition", fmt.Sprintf("invalid condition: %s", assertion.Condition))
	}
}

// validateThresholds validates threshold configuration.
func validateThresholds(t *ThresholdsConfig, errs *ValidationErrors) {
	// Validate duration thresholds
	for i, threshold := range t.HTTPReqDuration {
		if err := validateThresholdExpression(threshold); err != nil {
			errs.Add(fmt.Sprintf("thresholds.http_req_duration[%d]", i), err.Error())
		}
	}

	// Validate failure rate thresholds
	for i, threshold := range t.HTTPReqFailed {
		if err := validateThresholdExpression(threshold); err != nil {
			errs.Add(fmt.Sprintf("thresholds.http_req_failed[%d]", i), err.Error())
		}
	}

	// Validate request count thresholds
	for i, threshold := range t.HTTPReqs {
		if err := validateThresholdExpression(threshold); err != nil {
			errs.Add(fmt.Sprintf("thresholds.http_reqs[%d]", i), err.Error())
		}
	}

	// Validate custom thresholds
	for name, thresholds := range t.Custom {
		for i, threshold := range thresholds {
			if err := validateThresholdExpression(threshold); err != nil {
				errs.Add(fmt.Sprintf("thresholds.custom.%s[%d]", name, i), err.Error())
			}
		}
	}
}

// validateThresholdExpression validates a threshold expression.
//
// Valid formats:
//   - "p95 < 500ms"
//   - "avg < 200ms"
//   - "rate < 0.01"
//   - "count > 1000"
func validateThresholdExpression(expr string) error {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return fmt.Errorf("threshold expression cannot be empty")
	}

	// Valid metrics
	validMetrics := []string{"p50", "p90", "p95", "p99", "min", "max", "avg", "med", "rate", "count"}

	// Valid operators
	validOps := []string{"<", ">", "<=", ">=", "==", "!="}

	// Check if expression starts with a valid metric
	found := false
	for _, metric := range validMetrics {
		if strings.HasPrefix(expr, metric) {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("threshold must start with a valid metric (p50, p90, p95, p99, min, max, avg, med, rate, count)")
	}

	// Check for valid operator
	hasOp := false
	for _, op := range validOps {
		if strings.Contains(expr, op) {
			hasOp = true
			break
		}
	}
	if !hasOp {
		return fmt.Errorf("threshold must contain a comparison operator (<, >, <=, >=, ==, !=)")
	}

	return nil
}

// validateSettings validates global settings.
func validateSettings(s *GlobalSettings, errs *ValidationErrors) {
	// Validate base URL if provided
	if s.BaseURL != "" {
		if _, err := url.Parse(s.BaseURL); err != nil {
			errs.Add("settings.baseUrl", fmt.Sprintf("invalid URL: %v", err))
		}
	}

	// Validate connection settings
	if s.MaxConnectionsPerHost < 0 {
		errs.Add("settings.maxConnectionsPerHost", "cannot be negative")
	}
	if s.MaxIdleConnsPerHost < 0 {
		errs.Add("settings.maxIdleConnsPerHost", "cannot be negative")
	}
}
