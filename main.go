package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/tupyy/dcm-agent/cmd"
	"github.com/tupyy/dcm-agent/internal/config"
	"github.com/tupyy/dcm-agent/pkg/logger"
)

// These are set at build time via -ldflags
var (
	version   = "v0.0.0"  // Set via -ldflags "-X main.version=..."
	gitCommit = "unknown" // Set via -ldflags "-X main.gitCommit=..."
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "dcm-agent",
		Short: "DCM Agent",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
		},
	}

	// default configuration
	cfg := config.NewConfigurationWithOptionsAndDefaults(
		config.WithLogFormat("console"),
		config.WithLogLevel("debug"),
	)
	registerLoggingFlags(rootCmd, cfg)

	if err := validateConfig(cfg); err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	log := logger.Init(cfg.LogFormat, cfg.LogLevel)
	defer func() { _ = log.Sync() }()

	undo := zap.ReplaceGlobals(log)
	defer undo()

	rootCmd.AddCommand(cmd.NewRunCommand(cfg))

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
}

func validateConfig(cfg *config.Configuration) error {
	switch cfg.LogFormat {
	case "console":
	case "json":
	default:
		return fmt.Errorf("invalid log-format: %s", cfg.LogFormat)
	}

	if _, err := zapcore.ParseLevel(cfg.LogLevel); err != nil {
		return fmt.Errorf("invalid log level %s", cfg.LogLevel)
	}

	return nil
}

func registerLoggingFlags(cmd *cobra.Command, config *config.Configuration) {
	cmd.PersistentFlags().StringVar(&config.LogFormat, "log-format", config.LogFormat, "format of the logs: console or json")
	cmd.PersistentFlags().StringVar(&config.LogLevel, "log-level", config.LogLevel, "log level")
}
