package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/rancher-sandbox/profiling/internal/chaos/lock"
	"github.com/rancher-sandbox/profiling/internal/chaos/mem"
)

func main() {
	fmt.Println("Starting server")
	runtime.SetCPUProfileRate(1)
	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)
	ctx, ca := context.WithCancel(context.Background())
	defer ca()
	sigChan := make(chan os.Signal, 1)

	// Notify the channel on interrupt and termination signals.
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				sumRange(100000)
			}
		}
	}()

	lockRoutine := lock.NewLockRoutine()
	go lockRoutine.Run()

	memRoute := mem.NewMemLeaker()
	go memRoute.Run()

	// Block until a signal is received.
	sig := <-sigChan
	fmt.Printf("Received signal: %s\n", sig)
}

func sumRange(n int) {
	sum := 0
	for i := 0; i < n; i++ {
		sum += i
	}
}
