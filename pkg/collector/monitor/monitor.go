package monitor

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/alexandreLamarre/pprof-controller/pkg/collector/storage"
	"github.com/alexandreLamarre/pprof-controller/pkg/config"
)

type reqWrapper struct {
	req         *http.Request
	profileType string
}

type Monitor struct {
	logger *slog.Logger
	config *config.MonitorConfig

	lifecycleMu sync.Mutex
	stopper     chan struct{}
	ca          context.CancelFunc
	store       storage.Store
}

func NewMonitor(logger *slog.Logger, config *config.MonitorConfig, store storage.Store) *Monitor {
	return &Monitor{
		logger:      logger,
		config:      config,
		stopper:     nil,
		ca:          nil,
		lifecycleMu: sync.Mutex{},
		store:       store,
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
		req:         req,
		profileType: suffix,
	}, err
}

func (c *Monitor) requestsFromMonitorConfig() ([]reqWrapper, error) {
	reqs := []reqWrapper{}

	if c.config.GlobalSampling.Allocs != nil {
		req, err := c.constructRequest("allocs", c.config.GlobalSampling.Allocs.Seconds)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	if c.config.GlobalSampling.Block != nil {
		req, err := c.constructRequest("block", c.config.GlobalSampling.Block.Seconds)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	if c.config.GlobalSampling.Goroutine != nil {
		req, err := c.constructRequest("goroutine", c.config.GlobalSampling.Goroutine.Seconds)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	if c.config.GlobalSampling.Heap != nil {
		req, err := c.constructRequest("heap", c.config.GlobalSampling.Heap.Seconds)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	if c.config.GlobalSampling.Mutex != nil {
		req, err := c.constructRequest("mutex", c.config.GlobalSampling.Goroutine.Seconds)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	if c.config.GlobalSampling.Profile != nil {
		req, err := c.constructRequest("profile", c.config.GlobalSampling.Profile.Seconds)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	if c.config.GlobalSampling.ThreadCrate != nil {
		req, err := c.constructRequest("threadcrate", c.config.GlobalSampling.ThreadCrate.Seconds)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	// Disbale since this conflicts with some other pprof-like functionality
	// if c.config.GlobalSampling.Trace != nil {
	// 	req, err := c.constructRequest("trace", c.config.GlobalSampling.Trace.Seconds)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	reqs = append(reqs, req)
	// }
	return reqs, nil
}

// Spawns a goroutine to start monitor collection
func (c *Monitor) Start(ctx context.Context) error {
	logger := c.logger.With("name", c.config.Name)
	logger.Info("configuring monitor...")
	c.lifecycleMu.Lock()
	defer c.lifecycleMu.Unlock()
	reqs, err := c.requestsFromMonitorConfig()
	if err != nil {
		return err
	}
	logger.With("numRequests", len(reqs)).Info("monitors configured, starting...")
	c.start(ctx, reqs)
	return nil
}

func (c *Monitor) start(ctx context.Context, reqs []reqWrapper) {
	client := c.newClient()
	ctxca, ca := context.WithCancel(ctx)
	c.ca = ca
	c.stopper = make(chan struct{})
	for _, req := range reqs {
		req := req
		doReq := req.req.WithContext(ctxca)
		go func() {
			logger := c.logger.With("name", c.config.Name, "endpoint", doReq.URL.Path)
			for {
			RETRY:
				select {
				case <-c.stopper:
					logger.Info("monitor shutdown")
					c.ca()
					return
				default:
					logger.Debug("sending request")
					startTime := time.Now()
					resp, err := client.Do(doReq)
					endTime := time.Now()
					if err != nil {
						logger.Error(err.Error())
					} else {
						data, err := io.ReadAll(resp.Body)
						if err != nil {
							logger.Error(err.Error())
							// FIXME: backoff retrier
							time.Sleep(5 * time.Second)
							goto RETRY
						}
						logger.With("start-time", startTime, "end-time", endTime, "size", len(data)).Debug("got response")
						if err := c.store.Put(startTime, endTime, req.profileType, c.config.Name, c.config.Labels, data); err != nil {
							logger.With("err", err).Error("failed to store profile")
						}
						logger.With("start-time", startTime, "end-time", endTime, "size", len(data)).Debug("stored response")
					}
				}
			}
		}()
	}

}

func (c *Monitor) Shutdown() error {
	c.lifecycleMu.Lock()
	defer c.lifecycleMu.Unlock()
	c.logger.With("name", c.config.Name).Info("shutting down monitor...")
	// FIXME: hack
	if c.ca != nil {
		c.ca()
	}
	if c.stopper != nil {
		close(c.stopper)
	}
	// select {
	// case c.stopper <- struct{}{}:
	// default:
	// }
	return nil
}
