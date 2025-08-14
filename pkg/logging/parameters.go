package logging

import (
	"flag"
	"fmt"
	"log/slog"
)

type Parameters struct {
	Level slog.Level
	Type  LoggerType

	flagLogLevel   string
	flagLoggerType string
}

// Initialize adds logging command line parameters to the global flag set.
func (p *Parameters) Initialize() {
	flag.StringVar(&p.flagLogLevel, "log-level", "info",
		"Set the logging level. Supported values: debug, info, warn, error. Default: info.")
	flag.StringVar(&p.flagLoggerType, "log-type", "pretty",
		"Set the logger output format. Supported types: text, json, pretty. Default: pretty.")
}

// Parse parses the command line parameters for logging.
func (p *Parameters) Parse() error {
	var err error
	p.Level, err = p.parseLevel(p.flagLogLevel)
	if err != nil {
		return fmt.Errorf("failed to parse logger parameters: %w", err)
	}
	p.Type, err = p.parseType(p.flagLoggerType)
	if err != nil {
		return fmt.Errorf("failed to parse logger parameters: %w", err)
	}
	return nil
}

func (p *Parameters) String() string {
	return fmt.Sprintf("{Level: %s, Type: %s}", p.Level, p.Type)
}

func (p *Parameters) parseLevel(l string) (slog.Level, error) {
	var level slog.Level
	err := level.UnmarshalText([]byte(l))
	if err != nil {
		return slog.Level(0), fmt.Errorf("invalid log level: %w", err)
	}
	return level, nil
}

func (p *Parameters) parseType(t string) (LoggerType, error) {
	var lt LoggerType
	err := lt.UnmarshalText([]byte(t))
	if err != nil {
		return LoggerText, fmt.Errorf("invalid logger type: %w", err)
	}
	return lt, nil
}
