package logging

// LoggerType is a type of logger output.
// Possible types:
//   - LoggerText: The standard slog.TextHandler.
//   - LoggerJSON: The standard slog.JSONHandler.
//   - LoggerPretty: The logger outputs pretty messages.
//   - LoggerPrettyNoColor: The logger outputs pretty messages without colors.
//   - LoggerTextDev: The standard slog.TextHandler with error traces and source code location output.
//   - LoggerJSONDev: The standard slog.JSONHandler with error traces and source code location output.
//   - LoggerPrettyDev: The logger outputs pretty messages with error traces and source code location output.
//   - LoggerPrettyNoColorDev: The logger outputs pretty messages without colors with error traces and source code
//     location output.
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
