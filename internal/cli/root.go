// Package cli wires the command-line interface together and bootstraps the
// platform-go observability suite that the rest of the application builds on.
//
// The CLI is the template's single entrypoint: whether you are building a
// one-off tool, a long-running worker, or an HTTP service, you start here and
// hang new subcommands off the root command.
package cli

import (
	"context"
	"os"
	"time"

	"github.com/primandproper/template-go/internal/config"

	"github.com/primandproper/platform-go/v4/observability"
	"github.com/primandproper/platform-go/v4/observability/logging"

	"github.com/spf13/cobra"
)

// shutdownTimeout bounds how long we wait for telemetry to flush on exit.
const shutdownTimeout = 5 * time.Second

// application holds the process-wide dependencies constructed during startup.
// The logger is populated by bootstrap and shared with subcommands, which reach
// it through the application receiver on their RunE closures.
type application struct {
	pillars *observability.Pillars
	logger  logging.Logger
}

// Execute builds the root command, runs it, and tears down the observability
// suite afterwards so buffered telemetry is flushed even when a command fails.
func Execute(ctx context.Context) error {
	app := &application{}

	rootCmd := app.newRootCommand()
	err := rootCmd.ExecuteContext(ctx)

	app.shutdown(ctx)

	return err
}

// newRootCommand constructs the cobra root command and registers subcommands.
func (a *application) newRootCommand() *cobra.Command {
	var (
		logLevel    string
		serviceName string
	)

	rootCmd := &cobra.Command{
		Use:          config.DefaultServiceName,
		Short:        "A Go application template built on primandproper/platform-go.",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return a.bootstrap(cmd.Context(), config.Options{ServiceName: serviceName, LogLevel: logLevel})
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			a.log().Info("no subcommand provided; run `template-go help` to see what's available")

			return cmd.Help()
		},
	}

	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", envOr("TEMPLATE_GO_LOG_LEVEL", config.LevelInfo), "log level: debug, info, warn, or error")
	rootCmd.PersistentFlags().StringVar(&serviceName, "service-name", envOr("TEMPLATE_GO_SERVICE_NAME", config.DefaultServiceName), "service name reported in telemetry")

	rootCmd.AddCommand(a.newVersionCommand())

	return rootCmd
}

// bootstrap assembles configuration and stands up the observability pillars,
// caching the logger on the application for subcommands to use.
func (a *application) bootstrap(ctx context.Context, opts config.Options) error {
	cfg := config.New(opts)
	if err := cfg.Validate(ctx); err != nil {
		return err
	}

	pillars, err := cfg.NewPillars(ctx)
	if err != nil {
		return err
	}

	a.pillars = pillars
	a.logger = logging.NewNamedLogger(pillars.Logger, cfg.Observability.Logging.ServiceName)

	a.logger.Debug("observability suite initialized")

	return nil
}

// shutdown flushes and releases the observability pillars. It is safe to call
// when startup never completed. The parent context is typically already
// cancelled (SIGINT/SIGTERM), so we strip cancellation but keep its values and
// bound the flush with our own timeout.
func (a *application) shutdown(ctx context.Context) {
	if a.pillars == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()

	if err := a.pillars.Shutdown(ctx); err != nil {
		a.logger.Error("shutting down observability suite", err)
	}
}

// log returns the application logger, or a noop logger if bootstrap has not run.
func (a *application) log() logging.Logger {
	return logging.EnsureLogger(a.logger)
}

// envOr returns the value of the named environment variable, or fallback when
// it is unset or empty.
func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}

	return fallback
}
