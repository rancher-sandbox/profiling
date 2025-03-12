package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	_ "net/http/pprof"

	"github.com/alexandreLamarre/pprof-controller/pkg/collector"
	"github.com/alexandreLamarre/pprof-controller/pkg/collector/labels"
	"github.com/alexandreLamarre/pprof-controller/pkg/collector/storage"
	"github.com/alexandreLamarre/pprof-controller/pkg/collector/web"
	"github.com/alexandreLamarre/pprof-controller/pkg/config"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var (
	logger = slog.Default()
)

func BuildCollectorCmd() *cobra.Command {
	var configFile string
	var logLevel string
	var webPort int
	var dataDir string
	cmd := &cobra.Command{
		Use: "collector",
		RunE: func(cmd *cobra.Command, args []string) error {
			level := slog.LevelInfo

			switch strings.ToLower(logLevel) {
			case "debug":
				level = slog.LevelDebug
			case "info":
				level = slog.LevelInfo
			case "warn":
				level = slog.LevelWarn
			case "error":
				level = slog.LevelError
			default:
				logger.With("input-log-level", logLevel).Warn("invalid log level, defaulting to info")
			}
			setupLogger(level)
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
			logger.With("config", cfg).Info("loaded config")
			// start web UI

			logger.With("data-dir", dataDir).Info("setting up storage")
			var store storage.Store
			if err := os.MkdirAll(dataDir, 0755); err != nil {
				logger.With("data-dir", dataDir).Error("failed to create data dir")
				return fmt.Errorf("failed to create data dir: %w", err)
			}
			store = storage.NewLabelBasedFileStore(dataDir, []string{labels.NamespaceLabel, labels.NameLabel})

			webServer := web.NewWebServer(logger, webPort, store)
			errC := func() chan error {
				errC := make(chan error)
				go func() {
					errC <- webServer.Start()
				}()
				return errC
			}()
			logger.With("config", configFile).Info("starting collector")

			// start collector
			c := collector.NewCollector(context.Background(), logger, cfg, store)
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
				case <-errC:
					if err := c.Shutdown(); err != nil {
						return fmt.Errorf("failed to shutdown collector: %w", err)
					}
					return fmt.Errorf("failed to start web UI")
				case <-reloader:
					logger.Info("reloading collector config...")
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
	cmd.Flags().StringVarP(&logLevel, "log-level", "l", "info", "Log level")
	cmd.Flags().IntVarP(&webPort, "web-port", "p", 8989, "Port for web UI")
	cmd.Flags().StringVarP(&dataDir, "data-dir", "d", "/tmp/collector", "Directory to store and query profile data")
	return cmd
}

func setupLogger(level slog.Level) {
	// TODO : this is bugged levels aren't working correctly
	lvl := new(slog.LevelVar)
	lvl.Set(level)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	}))
	slog.SetDefault(logger)

}

func main() {
	cmd := BuildCollectorCmd()
	if err := cmd.Execute(); err != nil {
		slog.Default().With("err", err).Error("failed to run collector")
	}

}
