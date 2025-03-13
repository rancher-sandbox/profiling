package storage

import "time"

type NoopStore struct{}

var _ Store = (*NoopStore)(nil)

func NewNoopStore() *NoopStore {
	return &NoopStore{}
}

func (n *NoopStore) Put(startTime, endTime time.Time, profileType string, key string, labels map[string]string, value []byte) error {
	return nil
}

func (n *NoopStore) ListKeys() ([]string, error) {
	return []string{}, nil
}

func (n *NoopStore) GroupKeys() (map[string]map[string]map[string][]string, error) {
	return map[string]map[string]map[string][]string{}, nil
}

func (n *NoopStore) Get(profileType, key string) (filepaths []string, err error) {
	return []string{}, nil
}
