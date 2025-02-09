package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	_ "net/http/pprof"

	"github.com/alexandreLamarre/pprof-controller/pkg/collector"
	"github.com/alexandreLamarre/pprof-controller/pkg/config"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func BuildCollectorCmd() *cobra.Command {
	var configFile string
	cmd := &cobra.Command{
		Use: "collector",
		RunE: func(cmd *cobra.Command, args []string) error {
			stopper := make(chan os.Signal, 1)
			signal.Notify(stopper, os.Interrupt, syscall.SIGTERM)
			reloader := make(chan os.Signal, 1)
			signal.Notify(reloader, syscall.SIGHUP)

			var cfg *config.CollectorConfig
			data, err := os.ReadFile(configFile)
			if err != nil {
				return fmt.Errorf("failed to read config file: %w", err)
			}
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return fmt.Errorf("failed to unmarshal config file: %w", err)
			}

			c := collector.NewCollector(context.Background(), slog.Default(), cfg)

			err = c.Start(context.Background())
			if err != nil {
				return fmt.Errorf("failed to start collector: %w", err)
			}
			for {
				select {
				case <-stopper:
					if err := c.Shutdown(); err != nil {
						return fmt.Errorf("failed to shutdown collector: %w", err)
					}
					return nil
				case <-reloader:
					slog.Default().Info("reloading collector config...")
					data, err := os.ReadFile(configFile)
					if err != nil {
						return fmt.Errorf("failed to read config during reload file: %w", err)
					}
					if err := yaml.Unmarshal(data, &cfg); err != nil {
						return fmt.Errorf("failed to unmarshal config file during reload: %w", err)
					}
					if err := c.Reload(cfg); err != nil {
						return fmt.Errorf("failed to reload config: %w", err)
					}
				}
			}
		},
	}
	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to collector config file")
	return cmd
}

func init() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	slog.SetLogLoggerLevel(slog.LevelDebug)
}

func main() {
	cmd := BuildCollectorCmd()
	if err := cmd.Execute(); err != nil {
		slog.Default().With("err", err).Error("failed to run collector")
	}

}
