package lock

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type LockRoutine struct {
	sharedMu     sync.Mutex
	workloadSize int
}

func NewLockRoutine() *LockRoutine {
	return &LockRoutine{
		workloadSize: 50,
	}
}

func (l *LockRoutine) Run() {
	i := 1
	for {
		if i%5 == 0 {
			l.contentiousPath()
			i = 0
		} else {
			l.normalPath()
		}
		i++
	}
}

func (l *LockRoutine) contentiousPath() {
	logrus.Debug("starting contentiousPath mutex workload")
	wg := sync.WaitGroup{}
	wg.Add(l.workloadSize)
	for i := 0; i < l.workloadSize; i++ {
		go func() {
			defer wg.Done()
			l.sharedMu.Lock()
			time.Sleep(100 * time.Millisecond)
			l.sharedMu.Unlock()
		}()
	}
	wg.Wait()
	logrus.Debug("contentiousPath mutex workload done")

}

func (l *LockRoutine) normalPath() {
	logrus.Debug("starting normalPath mutex workload")
	wg := sync.WaitGroup{}
	wg.Add(l.workloadSize)
	for i := 0; i < l.workloadSize; i++ {
		go func() {
			defer wg.Done()
			l.sharedMu.Lock()
			time.Sleep(1 * time.Millisecond)
			l.sharedMu.Unlock()
		}()
	}
	wg.Wait()
	logrus.Debug("normalPath mutex workload done")
}
