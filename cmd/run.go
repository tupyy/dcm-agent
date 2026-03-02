package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/kubev2v/assisted-migration-agent/pkg/scheduler"

	"github.com/tupyy/dcm-agent/internal/config"
	"github.com/tupyy/dcm-agent/internal/services"
)

// NewRunCommand creates and returns the run command
func NewRunCommand(cfg *config.Configuration) *cobra.Command {
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run the DCM agent",
		Long:  `Run the DCM agent service.`,
		Run: func(cmd *cobra.Command, args []string) {
			zap.L().Info("starting dcm-agent")

			// Connect to NATS
			nc, err := nats.Connect(cfg.NatsURL)
			if err != nil {
				zap.S().Fatalw("failed to connect to NATS", "url", cfg.NatsURL, "error", err)
			}
			defer nc.Close()
			zap.S().Infow("connected to NATS", "url", cfg.NatsURL)

			// Create scheduler
			sched := scheduler.NewDefaultScheduler(cfg.Workers)
			defer sched.Close()
			zap.S().Infow("scheduler created", "workers", cfg.Workers)

			// Create ConsumerService (subscribes to NATS, returns dispatchCh)
			consumer, dispatchCh := services.NewConsumerService(nc, cfg.NatsSubject)
			if consumer == nil {
				zap.S().Fatal("failed to create consumer service")
			}

			// Create Executor (receives dispatchCh, returns statusChan)
			executor, statusChan := services.NewExecutor(sched, dispatchCh)

			// Create Watcher (receives statusChan from Executor)
			watcher := services.NewWatcher(statusChan)

			zap.S().Info("all services started, waiting for shutdown signal")

			// Wait for OS signal
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			sig := <-sigChan
			zap.S().Infow("received shutdown signal", "signal", sig)

			// Shutdown in order: ConsumerService -> Executor -> Watcher
			// Channel closure propagates: dispatchCh closed -> Executor exits and closes statusChan -> Watcher exits
			zap.S().Info("stopping consumer service (closes dispatchCh)")
			consumer.Stop()

			zap.S().Info("stopping executor (closes statusChan)")
			executor.Stop()

			zap.S().Info("stopping watcher")
			watcher.Stop()

			zap.S().Info("dcm-agent stopped")
		},
	}

	return runCmd
}
