package logging

import (
	"flag"
	"fmt"
	"log/slog"
)

const (
	DefaultLogLevelFlag   = "info"
	DefaultLoggerTypeFlag = "pretty"
)

type ParametersFlags struct {
	LogLevel   string
	LoggerType string
}

type Parameters struct {
	Level slog.Level
	Type  LoggerType

	flag ParametersFlags
}

// ParametersFromFlags creates a Parameters instance from the provided ParametersFlags.
// If flags is nil, it initializes with default values.
// It returns an error if the parameters cannot be parsed.
func ParametersFromFlags(flags *ParametersFlags) (Parameters, error) {
	if flags == nil {
		flags = &ParametersFlags{
			LogLevel:   DefaultLogLevelFlag,
			LoggerType: DefaultLoggerTypeFlag,
		}
	}
	p := Parameters{flag: *flags}
	if err := p.Parse(); err != nil {
		return Parameters{}, fmt.Errorf("failed to parse logger parameters: %w", err)
	}
	return p, nil
}

// Initialize adds logging command line parameters to the global flag set.
func (p *Parameters) Initialize() {
	flag.StringVar(&p.flag.LogLevel, "log-level", DefaultLogLevelFlag,
		"Set the logging level. Supported values: debug, info, warn, error.")
	flag.StringVar(&p.flag.LoggerType, "log-type", DefaultLoggerTypeFlag,
		"Set the logger output format. Supported types: text, json, pretty.")
}

// Parse parses the command line parameters for logging.
func (p *Parameters) Parse() error {
	var err error
	p.Level, err = p.parseLevel(p.flag.LogLevel)
	if err != nil {
		return fmt.Errorf("failed to parse logger parameters: %w", err)
	}
	p.Type, err = p.parseType(p.flag.LoggerType)
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
