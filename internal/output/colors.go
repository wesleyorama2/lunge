package output

import (
	"github.com/fatih/color"
)

// ColorScheme defines the colors used for different elements in the output
type ColorScheme struct {
	Method      *color.Color
	URL         *color.Color
	StatusOK    *color.Color
	StatusWarn  *color.Color
	StatusError *color.Color
	HeaderKey   *color.Color
	HeaderValue *color.Color
	JsonKey     *color.Color
	JsonValue   *color.Color
	Success     *color.Color
	Error       *color.Color
	Highlight   *color.Color
}

// DefaultColorScheme returns the default color scheme
func DefaultColorScheme() *ColorScheme {
	return &ColorScheme{
		Method:      color.New(color.FgBlue, color.Bold),
		URL:         color.New(color.FgCyan),
		StatusOK:    color.New(color.FgGreen, color.Bold),
		StatusWarn:  color.New(color.FgYellow, color.Bold),
		StatusError: color.New(color.FgRed, color.Bold),
		HeaderKey:   color.New(color.FgYellow),
		HeaderValue: color.New(color.FgWhite),
		JsonKey:     color.New(color.FgBlue),
		JsonValue:   color.New(color.FgWhite),
		Success:     color.New(color.FgGreen),
		Error:       color.New(color.FgRed),
		Highlight:   color.New(color.FgMagenta, color.Bold),
	}
}

// NoColorScheme returns a color scheme with all colors disabled
func NoColorScheme() *ColorScheme {
	scheme := DefaultColorScheme()

	// Disable all colors
	scheme.Method.DisableColor()
	scheme.URL.DisableColor()
	scheme.StatusOK.DisableColor()
	scheme.StatusWarn.DisableColor()
	scheme.StatusError.DisableColor()
	scheme.HeaderKey.DisableColor()
	scheme.HeaderValue.DisableColor()
	scheme.JsonKey.DisableColor()
	scheme.JsonValue.DisableColor()
	scheme.Success.DisableColor()
	scheme.Error.DisableColor()
	scheme.Highlight.DisableColor()

	return scheme
}

// SuccessIcon returns a checkmark symbol with appropriate color
func SuccessIcon(noColor bool) string {
	if noColor {
		return "✓"
	}
	return color.New(color.FgGreen).Sprint("✓")
}

// ErrorIcon returns an X symbol with appropriate color
func ErrorIcon(noColor bool) string {
	if noColor {
		return "✗"
	}
	return color.New(color.FgRed).Sprint("✗")
}

// InfoIcon returns an info symbol with appropriate color
func InfoIcon(noColor bool) string {
	if noColor {
		return "ℹ"
	}
	return color.New(color.FgBlue).Sprint("ℹ")
}

// WarningIcon returns a warning symbol with appropriate color
func WarningIcon(noColor bool) string {
	if noColor {
		return "⚠"
	}
	return color.New(color.FgYellow).Sprint("⚠")
}
