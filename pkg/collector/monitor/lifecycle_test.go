package monitor_test

import (
	"context"
	"log/slog"
	"net/http"
	"testing"

	_ "net/http/pprof"

	"github.com/alexandreLamarre/pprof-controller/pkg/collector/monitor"
	"github.com/alexandreLamarre/pprof-controller/pkg/collector/storage"
	"github.com/alexandreLamarre/pprof-controller/pkg/config"
	"github.com/stretchr/testify/assert"
)

const NoLogsLevel = 100

func TestLifecycle(t *testing.T) {
	server := &http.Server{
		Addr:    "localhost:6060",
		Handler: nil,
	}
	go server.ListenAndServe()
	defer server.Close()

	slog.SetLogLoggerLevel(NoLogsLevel)
	m := monitor.NewMonitor(slog.Default(), &config.MonitorConfig{
		Name:     "test",
		Endpoint: "http://localhost:6060",
		Labels:   map[string]string{},
	}, storage.NewNoopStore())
	ctx, ca := context.WithCancel(context.Background())
	defer ca()

	for i := 0; i < 10; i++ {
		assert.NoError(t, m.Start(ctx))
		assert.NoError(t, m.Shutdown())
	}
}
