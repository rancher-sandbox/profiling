package collector_test

import (
	"context"
	"log/slog"
	"net/http"
	"testing"

	"github.com/alexandreLamarre/pprof-controller/pkg/collector"
	"github.com/alexandreLamarre/pprof-controller/pkg/collector/storage"
	"github.com/alexandreLamarre/pprof-controller/pkg/config"
	"github.com/stretchr/testify/assert"
)

type lifesycleTc struct {
	baseConfig     *config.CollectorConfig
	incomingConfig *config.CollectorConfig
}

const NoLogsLevel = 100

func TestLifecycle(t *testing.T) {
	server := &http.Server{
		Addr:    "localhost:6060",
		Handler: nil,
	}
	go server.ListenAndServe()
	defer server.Close()
	// slog.SetLogLoggerLevel(NoLogsLevel)
	ctx, ca := context.WithCancel(context.Background())
	defer ca()

	tcs := []lifesycleTc{
		{
			baseConfig: &config.CollectorConfig{
				Monitors: []*config.MonitorConfig{
					{
						Name:     "test",
						Endpoint: "http://localhost:6060",
						GlobalSampling: config.GlobalSamplingConfig{
							Profile: &config.SamplerConfig{
								Seconds: 1,
							},
						},
					},
				},
			},
			incomingConfig: &config.CollectorConfig{
				Monitors: []*config.MonitorConfig{},
			},
		},
		{
			baseConfig: &config.CollectorConfig{
				Monitors: []*config.MonitorConfig{},
			},
			incomingConfig: &config.CollectorConfig{
				Monitors: []*config.MonitorConfig{
					{
						Name:     "test",
						Endpoint: "http://localhost:6060",
						GlobalSampling: config.GlobalSamplingConfig{
							Profile: &config.SamplerConfig{
								Seconds: 1,
							},
						},
					},
				},
			},
		},
		{
			baseConfig: &config.CollectorConfig{
				Monitors: []*config.MonitorConfig{
					{
						Name:     "test",
						Endpoint: "http://localhost:6060",
						GlobalSampling: config.GlobalSamplingConfig{
							Profile: &config.SamplerConfig{
								Seconds: 1,
							},
						},
					},
				},
			},
			incomingConfig: &config.CollectorConfig{
				Monitors: []*config.MonitorConfig{
					{
						Name:     "test",
						Endpoint: "http://localhost:6060",
						GlobalSampling: config.GlobalSamplingConfig{
							Profile: &config.SamplerConfig{
								Seconds: 1,
							},
						},
					},
				},
			},
		},
		// reload pprof self collection
		{
			baseConfig: &config.CollectorConfig{
				SelfTelemetry: &config.SelfTelemetryConfig{
					PprofPort:       7755,
					IntervalSeconds: 1,
				},
			},
			incomingConfig: &config.CollectorConfig{
				SelfTelemetry: &config.SelfTelemetryConfig{
					PprofPort:       7755,
					IntervalSeconds: 1,
				},
			},
		},
		// enable pprof self collection
		{
			baseConfig: &config.CollectorConfig{
				Monitors: []*config.MonitorConfig{},
			},
			incomingConfig: &config.CollectorConfig{
				SelfTelemetry: &config.SelfTelemetryConfig{
					PprofPort:       7755,
					IntervalSeconds: 1,
				},
			},
		},
		// disbale pprof self collection
		{
			baseConfig: &config.CollectorConfig{
				SelfTelemetry: &config.SelfTelemetryConfig{
					PprofPort:       7755,
					IntervalSeconds: 1,
				},
			},
			incomingConfig: &config.CollectorConfig{
				Monitors: []*config.MonitorConfig{},
			},
		},
		// -> switch pprof ports
		{
			baseConfig: &config.CollectorConfig{
				SelfTelemetry: &config.SelfTelemetryConfig{
					PprofPort:       7767,
					IntervalSeconds: 1,
				},
			},
			incomingConfig: &config.CollectorConfig{
				SelfTelemetry: &config.SelfTelemetryConfig{
					PprofPort:       7769,
					IntervalSeconds: 1,
				},
			},
		},
	}

	for _, tc := range tcs {
		c := collector.NewCollector(ctx, slog.Default(), tc.baseConfig, storage.NewNoopStore())
		assert.NoError(t, c.Start(ctx))
		assert.NoError(t, c.Reload(tc.incomingConfig))
		assert.NoError(t, c.Shutdown())
	}
}
