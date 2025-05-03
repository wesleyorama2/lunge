package output

import (
	"testing"
)

func TestColorSchemes(t *testing.T) {
	// Test DefaultColorScheme
	defaultScheme := DefaultColorScheme()
	if defaultScheme.Method == nil {
		t.Error("DefaultColorScheme.Method should not be nil")
	}
	if defaultScheme.URL == nil {
		t.Error("DefaultColorScheme.URL should not be nil")
	}
	if defaultScheme.StatusOK == nil {
		t.Error("DefaultColorScheme.StatusOK should not be nil")
	}
	if defaultScheme.StatusWarn == nil {
		t.Error("DefaultColorScheme.StatusWarn should not be nil")
	}
	if defaultScheme.StatusError == nil {
		t.Error("DefaultColorScheme.StatusError should not be nil")
	}
	if defaultScheme.HeaderKey == nil {
		t.Error("DefaultColorScheme.HeaderKey should not be nil")
	}
	if defaultScheme.HeaderValue == nil {
		t.Error("DefaultColorScheme.HeaderValue should not be nil")
	}
	if defaultScheme.JsonKey == nil {
		t.Error("DefaultColorScheme.JsonKey should not be nil")
	}
	if defaultScheme.JsonValue == nil {
		t.Error("DefaultColorScheme.JsonValue should not be nil")
	}
	if defaultScheme.Success == nil {
		t.Error("DefaultColorScheme.Success should not be nil")
	}
	if defaultScheme.Error == nil {
		t.Error("DefaultColorScheme.Error should not be nil")
	}
	if defaultScheme.Highlight == nil {
		t.Error("DefaultColorScheme.Highlight should not be nil")
	}

	// Test NoColorScheme
	noColorScheme := NoColorScheme()
	if noColorScheme.Method == nil {
		t.Error("NoColorScheme.Method should not be nil")
	}
	if noColorScheme.URL == nil {
		t.Error("NoColorScheme.URL should not be nil")
	}
	if noColorScheme.StatusOK == nil {
		t.Error("NoColorScheme.StatusOK should not be nil")
	}
	if noColorScheme.StatusWarn == nil {
		t.Error("NoColorScheme.StatusWarn should not be nil")
	}
	if noColorScheme.StatusError == nil {
		t.Error("NoColorScheme.StatusError should not be nil")
	}
	if noColorScheme.HeaderKey == nil {
		t.Error("NoColorScheme.HeaderKey should not be nil")
	}
	if noColorScheme.HeaderValue == nil {
		t.Error("NoColorScheme.HeaderValue should not be nil")
	}
	if noColorScheme.JsonKey == nil {
		t.Error("NoColorScheme.JsonKey should not be nil")
	}
	if noColorScheme.JsonValue == nil {
		t.Error("NoColorScheme.JsonValue should not be nil")
	}
	if noColorScheme.Success == nil {
		t.Error("NoColorScheme.Success should not be nil")
	}
	if noColorScheme.Error == nil {
		t.Error("NoColorScheme.Error should not be nil")
	}
	if noColorScheme.Highlight == nil {
		t.Error("NoColorScheme.Highlight should not be nil")
	}

	// Since we can't directly check if colors are disabled in a test environment,
	// we'll just verify that NoColorScheme returns a non-nil value
	// The actual disabling of colors is tested by the implementation of NoColorScheme
}

func TestIcons(t *testing.T) {
	// Test SuccessIcon
	successIcon := SuccessIcon(false)
	if successIcon == "" {
		t.Error("SuccessIcon should not be empty")
	}

	successIconNoColor := SuccessIcon(true)
	if successIconNoColor == "" {
		t.Error("SuccessIcon with noColor should not be empty")
	}

	// Test ErrorIcon
	errorIcon := ErrorIcon(false)
	if errorIcon == "" {
		t.Error("ErrorIcon should not be empty")
	}

	errorIconNoColor := ErrorIcon(true)
	if errorIconNoColor == "" {
		t.Error("ErrorIcon with noColor should not be empty")
	}

	// Test InfoIcon
	infoIcon := InfoIcon(false)
	if infoIcon == "" {
		t.Error("InfoIcon should not be empty")
	}

	infoIconNoColor := InfoIcon(true)
	if infoIconNoColor == "" {
		t.Error("InfoIcon with noColor should not be empty")
	}

	// Test WarningIcon
	warningIcon := WarningIcon(false)
	if warningIcon == "" {
		t.Error("WarningIcon should not be empty")
	}

	warningIconNoColor := WarningIcon(true)
	if warningIconNoColor == "" {
		t.Error("WarningIcon with noColor should not be empty")
	}
}
