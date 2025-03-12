package collector

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/alexandreLamarre/pprof-controller/pkg/collector/labels"
	"github.com/alexandreLamarre/pprof-controller/pkg/collector/storage"
	"github.com/alexandreLamarre/pprof-controller/pkg/config"
	"github.com/alexandreLamarre/pprof-controller/pkg/monitor"
	"golang.org/x/sync/errgroup"
)

type Collector struct {
	ctx         context.Context
	logger      *slog.Logger
	pprofServer *http.Server
	Config      *config.CollectorConfig
	Monitors    []*monitor.Monitor
	Store       storage.Store
}

func NewCollector(ctx context.Context, logger *slog.Logger, cfg *config.CollectorConfig, store storage.Store) *Collector {
	return &Collector{
		ctx:      ctx,
		logger:   logger,
		Config:   cfg,
		Monitors: nil,
		Store:    store,
	}
}

func (c *Collector) Start(ctx context.Context) error {
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
					Seconds: 5,
				},
				Heap: &config.SamplerConfig{
					Seconds: 5,
				},
			},
		},
			c.Store,
		)

		// FIXME: hack
		maxRetries := 5
		for range maxRetries {
			req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/debug/pprof", addr), nil)
			if err != nil {
				panic(err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				c.logger.Warn("error connecting to internal pprof server, retrying...")
				time.Sleep(5 * time.Second)
				continue
			}
			c.logger.Info("connected to internal pprof server")
			resp.Body.Close()
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
		go mon.Start(ctx)
	}
	return nil
}

func (c *Collector) Shutdown() error {
	var eg errgroup.Group
	for _, mon := range c.Monitors {
		mon := mon
		eg.Go(mon.Shutdown)
	}

	if c.pprofServer != nil {
		c.logger.Info("shutting down internal pprof server...")
		if err := c.pprofServer.Shutdown(c.ctx); err != nil {
			c.logger.With("err", err).Warn("error shutting down pprof server")
			return err
		}
		c.pprofServer = nil
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
