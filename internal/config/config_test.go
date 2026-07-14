package config

import (
	"context"
	"testing"

	"github.com/primandproper/platform-go/v4/observability/logging"
	loggingcfg "github.com/primandproper/platform-go/v4/observability/logging/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("defaults are valid and build pillars", func(t *testing.T) {
		t.Parallel()

		cfg := New(Options{})

		assert.Equal(t, DefaultServiceName, cfg.Observability.Logging.ServiceName)
		assert.Equal(t, loggingcfg.ProviderSlog, cfg.Observability.Logging.Provider)
		assert.True(t, logging.LevelsEqual(logging.InfoLevel, cfg.Observability.Logging.Level))

		require.NoError(t, cfg.Validate(context.Background()))

		pillars, err := cfg.Observability.NewPillars(context.Background())
		require.NoError(t, err)
		require.NotNil(t, pillars)
		require.NotNil(t, pillars.Logger)
	})

	t.Run("options override defaults", func(t *testing.T) {
		t.Parallel()

		cfg := New(Options{ServiceName: "custom", LogLevel: "debug"})

		assert.Equal(t, "custom", cfg.Observability.Logging.ServiceName)
		assert.True(t, logging.LevelsEqual(logging.DebugLevel, cfg.Observability.Logging.Level))
		require.NoError(t, cfg.Validate(context.Background()))
	})
}

func TestLevelFromString(t *testing.T) {
	t.Parallel()

	cases := map[string]logging.Level{
		"debug":     logging.DebugLevel,
		"info":      logging.InfoLevel,
		"warn":      logging.WarnLevel,
		"warning":   logging.WarnLevel,
		"error":     logging.ErrorLevel,
		"ERROR":     logging.ErrorLevel,
		"":          logging.InfoLevel,
		"nonsense":  logging.InfoLevel,
		"  debug  ": logging.DebugLevel,
	}

	for input, want := range cases {
		assert.Truef(t, logging.LevelsEqual(want, levelFromString(input)), "input %q", input)
	}
}
