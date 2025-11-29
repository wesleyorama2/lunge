package executor_test

import (
	"context"
	"testing"
	"time"

	"github.com/wesleyorama2/lunge/internal/performance/v2/executor"
)

func TestNewExecutor_ConstantVUs(t *testing.T) {
	e, err := executor.NewExecutor(executor.TypeConstantVUs)
	if err != nil {
		t.Fatalf("NewExecutor(TypeConstantVUs) error = %v", err)
	}
	if e == nil {
		t.Fatal("NewExecutor(TypeConstantVUs) returned nil")
	}
	if e.Type() != executor.TypeConstantVUs {
		t.Errorf("Type() = %v, want %v", e.Type(), executor.TypeConstantVUs)
	}
}

func TestNewExecutor_RampingVUs(t *testing.T) {
	e, err := executor.NewExecutor(executor.TypeRampingVUs)
	if err != nil {
		t.Fatalf("NewExecutor(TypeRampingVUs) error = %v", err)
	}
	if e == nil {
		t.Fatal("NewExecutor(TypeRampingVUs) returned nil")
	}
	if e.Type() != executor.TypeRampingVUs {
		t.Errorf("Type() = %v, want %v", e.Type(), executor.TypeRampingVUs)
	}
}

func TestNewExecutor_ConstantArrivalRate(t *testing.T) {
	e, err := executor.NewExecutor(executor.TypeConstantArrivalRate)
	if err != nil {
		t.Fatalf("NewExecutor(TypeConstantArrivalRate) error = %v", err)
	}
	if e == nil {
		t.Fatal("NewExecutor(TypeConstantArrivalRate) returned nil")
	}
	if e.Type() != executor.TypeConstantArrivalRate {
		t.Errorf("Type() = %v, want %v", e.Type(), executor.TypeConstantArrivalRate)
	}
}

func TestNewExecutor_RampingArrivalRate(t *testing.T) {
	e, err := executor.NewExecutor(executor.TypeRampingArrivalRate)
	if err != nil {
		t.Fatalf("NewExecutor(TypeRampingArrivalRate) error = %v", err)
	}
	if e == nil {
		t.Fatal("NewExecutor(TypeRampingArrivalRate) returned nil")
	}
	if e.Type() != executor.TypeRampingArrivalRate {
		t.Errorf("Type() = %v, want %v", e.Type(), executor.TypeRampingArrivalRate)
	}
}

func TestNewExecutor_PerVUIterations_NotImplemented(t *testing.T) {
	_, err := executor.NewExecutor(executor.TypePerVUIterations)
	if err == nil {
		t.Fatal("NewExecutor(TypePerVUIterations) expected error for not implemented, got nil")
	}
}

func TestNewExecutor_SharedIterations_NotImplemented(t *testing.T) {
	_, err := executor.NewExecutor(executor.TypeSharedIterations)
	if err == nil {
		t.Fatal("NewExecutor(TypeSharedIterations) expected error for not implemented, got nil")
	}
}

func TestNewExecutor_UnknownType(t *testing.T) {
	_, err := executor.NewExecutor(executor.Type("unknown-type"))
	if err == nil {
		t.Fatal("NewExecutor(unknown-type) expected error, got nil")
	}
}

func TestNewExecutor_EmptyType(t *testing.T) {
	_, err := executor.NewExecutor(executor.Type(""))
	if err == nil {
		t.Fatal("NewExecutor(\"\") expected error, got nil")
	}
}

func TestNewExecutorFromString_ConstantVUs(t *testing.T) {
	e, err := executor.NewExecutorFromString("constant-vus")
	if err != nil {
		t.Fatalf("NewExecutorFromString(\"constant-vus\") error = %v", err)
	}
	if e.Type() != executor.TypeConstantVUs {
		t.Errorf("Type() = %v, want %v", e.Type(), executor.TypeConstantVUs)
	}
}

func TestNewExecutorFromString_RampingVUs(t *testing.T) {
	e, err := executor.NewExecutorFromString("ramping-vus")
	if err != nil {
		t.Fatalf("NewExecutorFromString(\"ramping-vus\") error = %v", err)
	}
	if e.Type() != executor.TypeRampingVUs {
		t.Errorf("Type() = %v, want %v", e.Type(), executor.TypeRampingVUs)
	}
}

func TestNewExecutorFromString_ConstantArrivalRate(t *testing.T) {
	e, err := executor.NewExecutorFromString("constant-arrival-rate")
	if err != nil {
		t.Fatalf("NewExecutorFromString(\"constant-arrival-rate\") error = %v", err)
	}
	if e.Type() != executor.TypeConstantArrivalRate {
		t.Errorf("Type() = %v, want %v", e.Type(), executor.TypeConstantArrivalRate)
	}
}

func TestNewExecutorFromString_RampingArrivalRate(t *testing.T) {
	e, err := executor.NewExecutorFromString("ramping-arrival-rate")
	if err != nil {
		t.Fatalf("NewExecutorFromString(\"ramping-arrival-rate\") error = %v", err)
	}
	if e.Type() != executor.TypeRampingArrivalRate {
		t.Errorf("Type() = %v, want %v", e.Type(), executor.TypeRampingArrivalRate)
	}
}

func TestNewExecutorFromString_UnknownType(t *testing.T) {
	_, err := executor.NewExecutorFromString("invalid-executor")
	if err == nil {
		t.Fatal("NewExecutorFromString(\"invalid-executor\") expected error, got nil")
	}
}

func TestCreateAndInitExecutor_ConstantVUs(t *testing.T) {
	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      5,
		Duration: 1 * time.Minute,
	}

	e, err := executor.CreateAndInitExecutor(context.Background(), config)
	if err != nil {
		t.Fatalf("CreateAndInitExecutor() error = %v", err)
	}
	if e == nil {
		t.Fatal("CreateAndInitExecutor() returned nil")
	}
	if e.Type() != executor.TypeConstantVUs {
		t.Errorf("Type() = %v, want %v", e.Type(), executor.TypeConstantVUs)
	}
}

func TestCreateAndInitExecutor_RampingVUs(t *testing.T) {
	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 30 * time.Second, Target: 10},
			{Duration: 1 * time.Minute, Target: 10},
		},
	}

	e, err := executor.CreateAndInitExecutor(context.Background(), config)
	if err != nil {
		t.Fatalf("CreateAndInitExecutor() error = %v", err)
	}
	if e == nil {
		t.Fatal("CreateAndInitExecutor() returned nil")
	}
	if e.Type() != executor.TypeRampingVUs {
		t.Errorf("Type() = %v, want %v", e.Type(), executor.TypeRampingVUs)
	}
}

func TestCreateAndInitExecutor_ConstantArrivalRate(t *testing.T) {
	config := &executor.Config{
		Type:            executor.TypeConstantArrivalRate,
		Rate:            100.0,
		Duration:        1 * time.Minute,
		PreAllocatedVUs: 10,
		MaxVUs:          50,
	}

	e, err := executor.CreateAndInitExecutor(context.Background(), config)
	if err != nil {
		t.Fatalf("CreateAndInitExecutor() error = %v", err)
	}
	if e == nil {
		t.Fatal("CreateAndInitExecutor() returned nil")
	}
	if e.Type() != executor.TypeConstantArrivalRate {
		t.Errorf("Type() = %v, want %v", e.Type(), executor.TypeConstantArrivalRate)
	}
}

func TestCreateAndInitExecutor_RampingArrivalRate(t *testing.T) {
	config := &executor.Config{
		Type: executor.TypeRampingArrivalRate,
		Stages: []executor.Stage{
			{Duration: 30 * time.Second, Target: 50},
			{Duration: 1 * time.Minute, Target: 50},
		},
		PreAllocatedVUs: 10,
		MaxVUs:          50,
	}

	e, err := executor.CreateAndInitExecutor(context.Background(), config)
	if err != nil {
		t.Fatalf("CreateAndInitExecutor() error = %v", err)
	}
	if e == nil {
		t.Fatal("CreateAndInitExecutor() returned nil")
	}
	if e.Type() != executor.TypeRampingArrivalRate {
		t.Errorf("Type() = %v, want %v", e.Type(), executor.TypeRampingArrivalRate)
	}
}

func TestCreateAndInitExecutor_InvalidConfig(t *testing.T) {
	// Invalid config - missing VUs for constant-vus
	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		VUs:      0, // Invalid
		Duration: 1 * time.Minute,
	}

	_, err := executor.CreateAndInitExecutor(context.Background(), config)
	if err == nil {
		t.Fatal("CreateAndInitExecutor() expected error for invalid config, got nil")
	}
}

func TestCreateAndInitExecutor_UnknownType(t *testing.T) {
	config := &executor.Config{
		Type: executor.Type("unknown-type"),
	}

	_, err := executor.CreateAndInitExecutor(context.Background(), config)
	if err == nil {
		t.Fatal("CreateAndInitExecutor() expected error for unknown type, got nil")
	}
}

func TestIsValidExecutorType(t *testing.T) {
	tests := []struct {
		name     string
		execType string
		want     bool
	}{
		{"constant-vus", "constant-vus", true},
		{"ramping-vus", "ramping-vus", true},
		{"constant-arrival-rate", "constant-arrival-rate", true},
		{"ramping-arrival-rate", "ramping-arrival-rate", true},
		{"per-vu-iterations", "per-vu-iterations", true}, // Valid but not implemented
		{"shared-iterations", "shared-iterations", true}, // Valid but not implemented
		{"unknown", "unknown-type", false},
		{"empty", "", false},
		{"typo", "constant-vu", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := executor.IsValidExecutorType(tt.execType)
			if got != tt.want {
				t.Errorf("IsValidExecutorType(%q) = %v, want %v", tt.execType, got, tt.want)
			}
		})
	}
}

func TestGetSupportedExecutors(t *testing.T) {
	supported := executor.GetSupportedExecutors()

	if len(supported) != 4 {
		t.Errorf("GetSupportedExecutors() returned %d types, want 4", len(supported))
	}

	// Check that all expected types are present
	expectedTypes := []executor.Type{
		executor.TypeConstantVUs,
		executor.TypeRampingVUs,
		executor.TypeConstantArrivalRate,
		executor.TypeRampingArrivalRate,
	}

	for _, expected := range expectedTypes {
		found := false
		for _, actual := range supported {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetSupportedExecutors() missing type %v", expected)
		}
	}
}

func TestGetExecutorDescription_ConstantVUs(t *testing.T) {
	desc := executor.GetExecutorDescription(executor.TypeConstantVUs)
	if desc == nil {
		t.Fatal("GetExecutorDescription(TypeConstantVUs) returned nil")
	}
	if desc.Type != executor.TypeConstantVUs {
		t.Errorf("Type = %v, want %v", desc.Type, executor.TypeConstantVUs)
	}
	if desc.Name == "" {
		t.Error("Name should not be empty")
	}
	if desc.Description == "" {
		t.Error("Description should not be empty")
	}
	if len(desc.UseCases) == 0 {
		t.Error("UseCases should not be empty")
	}
}

func TestGetExecutorDescription_RampingVUs(t *testing.T) {
	desc := executor.GetExecutorDescription(executor.TypeRampingVUs)
	if desc == nil {
		t.Fatal("GetExecutorDescription(TypeRampingVUs) returned nil")
	}
	if desc.Type != executor.TypeRampingVUs {
		t.Errorf("Type = %v, want %v", desc.Type, executor.TypeRampingVUs)
	}
	if desc.Name == "" {
		t.Error("Name should not be empty")
	}
	if desc.Description == "" {
		t.Error("Description should not be empty")
	}
	if len(desc.UseCases) == 0 {
		t.Error("UseCases should not be empty")
	}
}

func TestGetExecutorDescription_ConstantArrivalRate(t *testing.T) {
	desc := executor.GetExecutorDescription(executor.TypeConstantArrivalRate)
	if desc == nil {
		t.Fatal("GetExecutorDescription(TypeConstantArrivalRate) returned nil")
	}
	if desc.Type != executor.TypeConstantArrivalRate {
		t.Errorf("Type = %v, want %v", desc.Type, executor.TypeConstantArrivalRate)
	}
	if desc.Name == "" {
		t.Error("Name should not be empty")
	}
	if desc.Description == "" {
		t.Error("Description should not be empty")
	}
	if len(desc.UseCases) == 0 {
		t.Error("UseCases should not be empty")
	}
}

func TestGetExecutorDescription_RampingArrivalRate(t *testing.T) {
	desc := executor.GetExecutorDescription(executor.TypeRampingArrivalRate)
	if desc == nil {
		t.Fatal("GetExecutorDescription(TypeRampingArrivalRate) returned nil")
	}
	if desc.Type != executor.TypeRampingArrivalRate {
		t.Errorf("Type = %v, want %v", desc.Type, executor.TypeRampingArrivalRate)
	}
	if desc.Name == "" {
		t.Error("Name should not be empty")
	}
	if desc.Description == "" {
		t.Error("Description should not be empty")
	}
	if len(desc.UseCases) == 0 {
		t.Error("UseCases should not be empty")
	}
}

func TestGetExecutorDescription_UnknownType(t *testing.T) {
	desc := executor.GetExecutorDescription(executor.Type("unknown-type"))
	if desc != nil {
		t.Error("GetExecutorDescription(unknown-type) should return nil")
	}
}

func TestCalculateEstimatedDuration_ConstantVUs(t *testing.T) {
	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		Duration: 5 * time.Minute,
	}

	duration := executor.CalculateEstimatedDuration(config)
	if duration != 5*time.Minute {
		t.Errorf("CalculateEstimatedDuration() = %v, want 5m", duration)
	}
}

func TestCalculateEstimatedDuration_RampingVUs(t *testing.T) {
	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 1 * time.Minute, Target: 10},
			{Duration: 2 * time.Minute, Target: 10},
			{Duration: 30 * time.Second, Target: 0},
		},
	}

	duration := executor.CalculateEstimatedDuration(config)
	expected := 3*time.Minute + 30*time.Second
	if duration != expected {
		t.Errorf("CalculateEstimatedDuration() = %v, want %v", duration, expected)
	}
}

func TestCalculateEstimatedDuration_ConstantArrivalRate(t *testing.T) {
	config := &executor.Config{
		Type:     executor.TypeConstantArrivalRate,
		Duration: 10 * time.Minute,
	}

	duration := executor.CalculateEstimatedDuration(config)
	if duration != 10*time.Minute {
		t.Errorf("CalculateEstimatedDuration() = %v, want 10m", duration)
	}
}

func TestCalculateMaxVUs_ConstantVUs(t *testing.T) {
	config := &executor.Config{
		Type: executor.TypeConstantVUs,
		VUs:  10,
	}

	maxVUs := executor.CalculateMaxVUs(config)
	if maxVUs != 10 {
		t.Errorf("CalculateMaxVUs() = %d, want 10", maxVUs)
	}
}

func TestCalculateMaxVUs_RampingVUs(t *testing.T) {
	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 1 * time.Minute, Target: 10},
			{Duration: 2 * time.Minute, Target: 50}, // Max target
			{Duration: 30 * time.Second, Target: 0},
		},
	}

	maxVUs := executor.CalculateMaxVUs(config)
	if maxVUs != 50 {
		t.Errorf("CalculateMaxVUs() = %d, want 50", maxVUs)
	}
}

func TestCalculateMaxVUs_ConstantArrivalRate(t *testing.T) {
	config := &executor.Config{
		Type:   executor.TypeConstantArrivalRate,
		MaxVUs: 100,
	}

	maxVUs := executor.CalculateMaxVUs(config)
	if maxVUs != 100 {
		t.Errorf("CalculateMaxVUs() = %d, want 100", maxVUs)
	}
}

func TestCalculateMaxVUs_RampingArrivalRate(t *testing.T) {
	config := &executor.Config{
		Type:   executor.TypeRampingArrivalRate,
		MaxVUs: 200,
	}

	maxVUs := executor.CalculateMaxVUs(config)
	if maxVUs != 200 {
		t.Errorf("CalculateMaxVUs() = %d, want 200", maxVUs)
	}
}

func TestCalculateMaxVUs_UnknownType(t *testing.T) {
	config := &executor.Config{
		Type: executor.Type("unknown"),
		VUs:  5,
	}

	maxVUs := executor.CalculateMaxVUs(config)
	if maxVUs != 5 {
		t.Errorf("CalculateMaxVUs() = %d, want 5 (fallback to VUs)", maxVUs)
	}
}

func TestConfig_Validate_ConstantVUs(t *testing.T) {
	tests := []struct {
		name    string
		config  *executor.Config
		wantErr bool
	}{
		{
			name: "valid",
			config: &executor.Config{
				Type:     executor.TypeConstantVUs,
				VUs:      10,
				Duration: 1 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "missing vus",
			config: &executor.Config{
				Type:     executor.TypeConstantVUs,
				VUs:      0,
				Duration: 1 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "missing duration",
			config: &executor.Config{
				Type:     executor.TypeConstantVUs,
				VUs:      10,
				Duration: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_RampingVUs(t *testing.T) {
	tests := []struct {
		name    string
		config  *executor.Config
		wantErr bool
	}{
		{
			name: "valid",
			config: &executor.Config{
				Type: executor.TypeRampingVUs,
				Stages: []executor.Stage{
					{Duration: 30 * time.Second, Target: 10},
				},
			},
			wantErr: false,
		},
		{
			name: "missing stages",
			config: &executor.Config{
				Type:   executor.TypeRampingVUs,
				Stages: []executor.Stage{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_ConstantArrivalRate(t *testing.T) {
	tests := []struct {
		name    string
		config  *executor.Config
		wantErr bool
	}{
		{
			name: "valid",
			config: &executor.Config{
				Type:     executor.TypeConstantArrivalRate,
				Rate:     100.0,
				Duration: 1 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "missing rate",
			config: &executor.Config{
				Type:     executor.TypeConstantArrivalRate,
				Rate:     0,
				Duration: 1 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "missing duration",
			config: &executor.Config{
				Type:     executor.TypeConstantArrivalRate,
				Rate:     100.0,
				Duration: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_RampingArrivalRate(t *testing.T) {
	tests := []struct {
		name    string
		config  *executor.Config
		wantErr bool
	}{
		{
			name: "valid",
			config: &executor.Config{
				Type: executor.TypeRampingArrivalRate,
				Stages: []executor.Stage{
					{Duration: 30 * time.Second, Target: 50},
				},
			},
			wantErr: false,
		},
		{
			name: "missing stages",
			config: &executor.Config{
				Type:   executor.TypeRampingArrivalRate,
				Stages: []executor.Stage{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_PerVUIterations(t *testing.T) {
	tests := []struct {
		name    string
		config  *executor.Config
		wantErr bool
	}{
		{
			name: "valid",
			config: &executor.Config{
				Type:       executor.TypePerVUIterations,
				VUs:        5,
				Iterations: 100,
			},
			wantErr: false,
		},
		{
			name: "missing vus",
			config: &executor.Config{
				Type:       executor.TypePerVUIterations,
				VUs:        0,
				Iterations: 100,
			},
			wantErr: true,
		},
		{
			name: "missing iterations",
			config: &executor.Config{
				Type:       executor.TypePerVUIterations,
				VUs:        5,
				Iterations: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_SharedIterations(t *testing.T) {
	tests := []struct {
		name    string
		config  *executor.Config
		wantErr bool
	}{
		{
			name: "valid",
			config: &executor.Config{
				Type:       executor.TypeSharedIterations,
				VUs:        5,
				Iterations: 1000,
			},
			wantErr: false,
		},
		{
			name: "missing vus",
			config: &executor.Config{
				Type:       executor.TypeSharedIterations,
				VUs:        0,
				Iterations: 1000,
			},
			wantErr: true,
		},
		{
			name: "missing iterations",
			config: &executor.Config{
				Type:       executor.TypeSharedIterations,
				VUs:        5,
				Iterations: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_UnknownType(t *testing.T) {
	config := &executor.Config{
		Type: executor.Type("unknown-type"),
	}

	err := config.Validate()
	if err == nil {
		t.Error("Validate() expected error for unknown type, got nil")
	}
}

func TestConfig_Validate_EmptyType(t *testing.T) {
	config := &executor.Config{
		Type: executor.Type(""),
	}

	err := config.Validate()
	if err == nil {
		t.Error("Validate() expected error for empty type, got nil")
	}
}

func TestConfig_TotalDuration_ConstantVUs(t *testing.T) {
	config := &executor.Config{
		Type:     executor.TypeConstantVUs,
		Duration: 5 * time.Minute,
	}

	if config.TotalDuration() != 5*time.Minute {
		t.Errorf("TotalDuration() = %v, want 5m", config.TotalDuration())
	}
}

func TestConfig_TotalDuration_RampingVUs(t *testing.T) {
	config := &executor.Config{
		Type: executor.TypeRampingVUs,
		Stages: []executor.Stage{
			{Duration: 1 * time.Minute, Target: 10},
			{Duration: 2 * time.Minute, Target: 10},
		},
	}

	expected := 3 * time.Minute
	if config.TotalDuration() != expected {
		t.Errorf("TotalDuration() = %v, want %v", config.TotalDuration(), expected)
	}
}

func TestConfig_TotalDuration_PerVUIterations(t *testing.T) {
	config := &executor.Config{
		Type:       executor.TypePerVUIterations,
		VUs:        5,
		Iterations: 100,
	}

	// Duration not applicable for iteration-based executors
	if config.TotalDuration() != 0 {
		t.Errorf("TotalDuration() = %v, want 0", config.TotalDuration())
	}
}

func TestConfig_TotalDuration_UnknownType(t *testing.T) {
	config := &executor.Config{
		Type:     executor.Type("unknown"),
		Duration: 5 * time.Minute,
	}

	if config.TotalDuration() != 0 {
		t.Errorf("TotalDuration() = %v, want 0 for unknown type", config.TotalDuration())
	}
}

func TestValidationError(t *testing.T) {
	err := &executor.ValidationError{
		Field:   "vus",
		Message: "vus must be > 0",
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() should not return empty string")
	}

	// Check that error message contains field and message
	if !contains(errStr, "vus") {
		t.Errorf("Error() = %q, should contain field 'vus'", errStr)
	}
	if !contains(errStr, "vus must be > 0") {
		t.Errorf("Error() = %q, should contain message 'vus must be > 0'", errStr)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
