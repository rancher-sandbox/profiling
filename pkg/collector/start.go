package collector

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/rancher-sandbox/profiling/pkg/collector/labels"
	"github.com/rancher-sandbox/profiling/pkg/collector/monitor"
	"github.com/rancher-sandbox/profiling/pkg/collector/storage"
	"github.com/rancher-sandbox/profiling/pkg/config"
	"golang.org/x/sync/errgroup"
)

type Collector struct {
	ctx         context.Context
	logger      *slog.Logger
	pprofServer *http.Server
	Config      *config.CollectorConfig
	Monitors    []*monitor.Monitor
	Store       storage.Store

	lifecycleMu sync.Mutex
}

func NewCollector(ctx context.Context, logger *slog.Logger, cfg *config.CollectorConfig, store storage.Store) *Collector {
	return &Collector{
		ctx:         ctx,
		logger:      logger,
		Config:      cfg,
		Monitors:    nil,
		Store:       store,
		lifecycleMu: sync.Mutex{},
	}
}

func (c *Collector) Start(ctx context.Context) error {
	c.lifecycleMu.Lock()
	defer c.lifecycleMu.Unlock()
	mons := []*monitor.Monitor{}

	if c.Config.SelfTelemetry != nil {
		addr := fmt.Sprintf("127.0.0.1:%d", c.Config.SelfTelemetry.PprofPort)
		c.logger.With("addr", addr).Info("configuring internal pprof server")
		server := &http.Server{
			Addr:    addr,
			Handler: nil,
		}

		if c.pprofServer != nil {
			panic("pprof server should be nil here")
		}
		go func() {
			c.logger.With("addr", addr).Info("launching pprof server")
			if err := server.ListenAndServe(); err != nil {
				c.logger.Error(err.Error())
			}
		}()
		c.pprofServer = server
		mon := monitor.NewMonitor(c.logger, &config.MonitorConfig{
			Name:     "__self",
			Endpoint: fmt.Sprintf("http://%s", addr),
			Labels: map[string]string{
				// FIXME: temporary hack
				labels.NamespaceLabel: "self",
				labels.NameLabel:      "self",
			},
			GlobalSampling: config.GlobalSamplingConfig{
				Profile: &config.SamplerConfig{
					Seconds: c.Config.SelfTelemetry.IntervalSeconds,
				},
				Heap: &config.SamplerConfig{
					Seconds: c.Config.SelfTelemetry.IntervalSeconds,
				},
				Goroutine: &config.SamplerConfig{
					Seconds: c.Config.SelfTelemetry.IntervalSeconds,
				},
				Allocs: &config.SamplerConfig{
					Seconds: c.Config.SelfTelemetry.IntervalSeconds,
				},
				Block: &config.SamplerConfig{
					Seconds: c.Config.SelfTelemetry.IntervalSeconds,
				},
				Mutex: &config.SamplerConfig{
					Seconds: c.Config.SelfTelemetry.IntervalSeconds,
				},
				ThreadCreate: &config.SamplerConfig{
					Seconds: c.Config.SelfTelemetry.IntervalSeconds,
				},
			},
		},
			c.Store,
		)

		// FIXME: hack
		maxRetries := 50
		for range maxRetries {
			req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/debug/pprof", addr), nil)
			if err != nil {
				panic(err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				c.logger.Info("waiting for internal pprof endpoint to be available, retrying...")
				time.Sleep(50 * time.Millisecond)
				continue
			}
			c.logger.Info("connected to internal pprof server")
			resp.Body.Close()
			break
		}
		mons = append(mons, mon)
	}

	c.logger.With("len", len(c.Config.Monitors)).Info("starting external monitors...")
	for _, cfg := range c.Config.Monitors {
		mons = append(mons, monitor.NewMonitor(c.logger, cfg, c.Store))
	}
	c.Monitors = mons
	for _, mon := range c.Monitors {
		mon := mon
		mon.Start(ctx)
	}
	return nil
}

func (c *Collector) Shutdown() error {
	c.lifecycleMu.Lock()
	defer c.lifecycleMu.Unlock()
	var eg errgroup.Group
	for _, mon := range c.Monitors {
		mon := mon
		eg.Go(mon.Shutdown)
	}

	if c.pprofServer != nil {
		eg.Go(func() error {
			c.logger.Info("shutting down internal pprof server...")
			if err := c.pprofServer.Shutdown(c.ctx); err != nil {
				c.logger.With("err", err).Warn("error shutting down pprof server")
				return err
			}
			c.pprofServer = nil
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		c.logger.With("err", err).Error("shutting down monitors")
		return err
	}
	return nil
}

func (c *Collector) Reload(cfg *config.CollectorConfig) error {
	if err := c.Shutdown(); err != nil {
		return err
	}
	c.Config = cfg
	if err := c.Start(context.TODO()); err != nil {
		return err
	}

	return nil
}
