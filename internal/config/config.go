// Package config assembles the application's configuration, most notably the
// observability settings that the platform-go observability suite consumes.
//
// The configuration is built in Go with sensible, zero-dependency defaults so
// the binary boots out of the box: structured slog logging plus noop tracing,
// metrics, and profiling. Callers may override the service name and log level
// (see Options), which is how the CLI threads its flags and environment into
// the platform configuration.
package config

import (
	"context"
	"strings"

	"github.com/primandproper/platform-go/v4/observability"
	"github.com/primandproper/platform-go/v4/observability/logging"
	loggingcfg "github.com/primandproper/platform-go/v4/observability/logging/config"
	metricsnoop "github.com/primandproper/platform-go/v4/observability/metrics/noop"
	profilingnoop "github.com/primandproper/platform-go/v4/observability/profiling/noop"
	tracingnoop "github.com/primandproper/platform-go/v4/observability/tracing/noop"
)

// DefaultServiceName is the service name reported by the observability suite
// when the caller does not supply one.
const DefaultServiceName = "template-go"

// Log level names accepted by Options.LogLevel (case-insensitive).
const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
)

// Config is the application configuration. It wraps the platform-go
// observability configuration; add your own fields here as the application grows.
type Config struct {
	Observability observability.Config
}

// Options tune the values that most deployments care about. Empty fields fall
// back to the built-in defaults.
type Options struct {
	// ServiceName labels telemetry emitted by the service.
	ServiceName string
	// LogLevel is one of "debug", "info", "warn", or "error" (case-insensitive).
	LogLevel string
}

// New builds a Config from the given options, filling in defaults. The result
// validates cleanly and is ready to hand to observability.Config.NewPillars.
func New(opts Options) *Config {
	serviceName := strings.TrimSpace(opts.ServiceName)
	if serviceName == "" {
		serviceName = DefaultServiceName
	}

	cfg := &Config{
		Observability: observability.Config{
			// slog logging to stdout as structured JSON.
			Logging: loggingcfg.Config{
				Provider:    loggingcfg.ProviderSlog,
				ServiceName: serviceName,
				Level:       levelFromString(opts.LogLevel),
			},
			// Tracing, Metrics, and Profiling are left at their zero values,
			// which the platform resolves to noop providers. Enable them by
			// populating the corresponding sub-config.
		},
	}

	return cfg
}

// Validate confirms the assembled configuration is internally consistent.
func (c *Config) Validate(ctx context.Context) error {
	return c.Observability.ValidateWithContext(ctx)
}

// NewPillars builds the observability pillars for the application.
//
// Logging is configured from Config (structured slog by default), while
// tracing, metrics, and profiling default to noop providers so the binary stays
// quiet and dependency-free out of the box. To enable real telemetry, populate
// the Tracing/Metrics/Profiling sub-configs of c.Observability and call
// c.Observability.NewPillars(ctx) instead — it wires OTel/Cloud providers from
// the same config — or replace the noop constructors below with your own.
func (c *Config) NewPillars(ctx context.Context) (*observability.Pillars, error) {
	logger, err := c.Observability.Logging.NewLogger(ctx)
	if err != nil {
		return nil, err
	}

	return &observability.Pillars{
		Logger:          logger,
		TracerProvider:  tracingnoop.NewTracerProvider(),
		MetricsProvider: metricsnoop.NewMetricsProvider(),
		Profiler:        profilingnoop.NewProvider(),
	}, nil
}

// levelFromString maps a human-friendly level name onto a platform log level,
// defaulting to info for empty or unrecognized input.
func levelFromString(s string) logging.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case LevelDebug:
		return logging.DebugLevel
	case LevelWarn, "warning":
		return logging.WarnLevel
	case LevelError:
		return logging.ErrorLevel
	default:
		return logging.InfoLevel
	}
}
