package logging

// LoggerType is a type of logger output.
// Possible types:
//   - LoggerText: The standard slog.TextHandler.
//   - LoggerJSON: The standard slog.JSONHandler.
//   - LoggerPretty: The logger outputs pretty messages.
//   - LoggerPrettyNoColor: The logger outputs pretty messages without colors.
//
//go:generate enumer -type LoggerType -trimprefix Logger -text -output loggertype_string.go
type LoggerType int

const (
	LoggerText LoggerType = iota
	LoggerJSON
	LoggerPretty
	LoggerPrettyNoColor
	LoggerTextDev
	LoggerJSONDev
	LoggerPrettyDev
	LoggerPrettyNoColorDev
)
