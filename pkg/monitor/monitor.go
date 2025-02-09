package monitor

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/alexandreLamarre/pprof-controller/pkg/config"
)

type reqWrapper struct {
	req *http.Request
}

type Monitor struct {
	logger *slog.Logger
	config *config.MonitorConfig

	stopper chan struct{}
	ca      context.CancelFunc
}

func NewMonitor(logger *slog.Logger, config *config.MonitorConfig) *Monitor {
	return &Monitor{
		logger:  logger,
		config:  config,
		stopper: make(chan struct{}),
		ca:      nil,
	}
}

func (c *Monitor) newClient() *http.Client {
	// TODO
	return http.DefaultClient
}

func (c *Monitor) constructRequest(suffix string, seconds int) (reqWrapper, error) {
	target := c.config.Endpoint + "/debug/pprof/" + suffix
	if seconds != 0 {
		target += fmt.Sprintf("?seconds=%d", seconds)
	}
	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		return reqWrapper{}, err
	}
	return reqWrapper{
		req: req,
	}, err
}

func (c *Monitor) requestsFromMonitorConfig() ([]reqWrapper, error) {
	reqs := []reqWrapper{}
	if c.config.Allocs != nil {
		req, err := c.constructRequest("alloc", c.config.Allocs.Seconds)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	if c.config.Block != nil {
		req, err := c.constructRequest("block", c.config.Block.Seconds)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	if c.config.Goroutine != nil {
		req, err := c.constructRequest("goroutine", c.config.Goroutine.Seconds)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	if c.config.Heap != nil {
		req, err := c.constructRequest("heap", c.config.Heap.Seconds)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	if c.config.Mutex != nil {
		req, err := c.constructRequest("goroutine", c.config.Goroutine.Seconds)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	if c.config.Profile != nil {
		req, err := c.constructRequest("profile", c.config.Profile.Seconds)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	if c.config.ThreadCrate != nil {
		req, err := c.constructRequest("threadcrate", c.config.ThreadCrate.Seconds)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	if c.config.Trace != nil {
		req, err := c.constructRequest("trace", c.config.Trace.Seconds)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	return reqs, nil
}

// Spawns a goroutine to start monitor collection
func (c *Monitor) Start(ctx context.Context) error {
	c.logger.With("name", c.config.Name).Info("starting monitor")
	reqs, err := c.requestsFromMonitorConfig()
	if err != nil {
		return err
	}
	c.start(ctx, reqs)
	return nil
}

func (c *Monitor) start(ctx context.Context, reqs []reqWrapper) {
	client := c.newClient()
	ctxca, ca := context.WithCancel(ctx)
	c.ca = ca
	for _, req := range reqs {
		req := req
		doReq := req.req.WithContext(ctxca)
		go func() {
			for {
				select {
				case <-c.stopper:
					c.logger.With("name", c.config.Name, "endpoint", doReq.URL.Path).Info("monitor shutdown")
					c.ca()
					return
				default:
					c.logger.With("name", c.config.Name, "endpoint", doReq.URL.Path).Debug("sending request")
					resp, err := client.Do(doReq)
					if err != nil {
						c.logger.Error(err.Error())
					} else {
						data, err := io.ReadAll(resp.Body)
						if err != nil {
							c.logger.Error(err.Error())
						}
						c.logger.With("name", c.config.Name, "endpoint", doReq.URL.Path, "size", len(data)).Debug("got response")
					}
				}
			}
		}()
	}

}

func (c *Monitor) Shutdown() error {
	c.logger.With("name", c.config.Name).Info("shutting down monitor...")
	c.ca()
	c.stopper <- struct{}{}
	return nil
}
