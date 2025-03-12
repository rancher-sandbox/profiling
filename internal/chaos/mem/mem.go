package mem

import (
	"time"

	"github.com/sirupsen/logrus"
)

type MemLeaker struct {
	data [][]byte

	workloadSize int
}

func NewMemLeaker() *MemLeaker {
	return &MemLeaker{
		data:         [][]byte{},
		workloadSize: 5000,
	}
}

func (m *MemLeaker) Run() {
	i := 1
	for {
		if i%5 == 0 {
			m.UnsafeAddAndDelete()
			i = 0
		} else {
			m.SafeAddAndDelete()
		}
		i++
	}
}

func (m *MemLeaker) SafeAddAndDelete() {
	logrus.Info("Running memory safe workload")
	for i := 0; i < m.workloadSize; i++ {
		m.data = append(m.data, make([]byte, 1024))
	}
	m.simluateWorkload()

	for i := 0; i < m.workloadSize; i++ {
		m.data = m.data[:len(m.data)-1]
	}
	logrus.Info("Done running memory safe workload")
}

func (m *MemLeaker) UnsafeAddAndDelete() {
	logrus.Info("Running memory unsafe workload")
	for i := 0; i < m.workloadSize; i++ {
		m.data = append(m.data, make([]byte, 1024))
	}
	m.simluateWorkload()

	for i := 0; i < m.workloadSize-100; i++ {
		m.data = m.data[:len(m.data)-1]
	}
	logrus.Infof("Done running memory unsafe workload : %d", len(m.data))
}

func (m *MemLeaker) simluateWorkload() {
	time.Sleep(1 * time.Second)
}
