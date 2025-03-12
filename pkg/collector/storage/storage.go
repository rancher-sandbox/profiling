package storage

// FIXME: this entire implementation is a mess, done for speed / demonstration purposes

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type Store interface {
	Put(startTime, endTime time.Time, profileType string, key string, labels map[string]string, value []byte) error
	ListKeys() ([]string, error)
	Get(key string) (filepaths []string, err error)
}

type LabelBasedFileStore struct {
	DataDir string

	IndexBy []string
}

func NewLabelBasedFileStore(dataDir string, indexBy []string) *LabelBasedFileStore {
	return &LabelBasedFileStore{
		DataDir: dataDir,
		IndexBy: indexBy,
	}
}

var _ Store = (*LabelBasedFileStore)(nil)

func (s *LabelBasedFileStore) basePath(
	labels map[string]string,
	key string,
) (string, error) {
	base := s.DataDir
	for _, idx := range s.IndexBy {
		if _, ok := labels[idx]; !ok {
			return "", fmt.Errorf("missing label %s to use as index", idx)
		}
		base = path.Join(base, labels[idx])
	}
	base = path.Join(base, key)
	return base, nil
}

func (s *LabelBasedFileStore) Put(startTime, endTime time.Time, profileType, key string, labels map[string]string, value []byte) error {
	basePath, err := s.basePath(labels, key)
	if err != nil {
		return err
	}
	basePath = path.Join(basePath, profileType)

	startNano := startTime.UnixNano()
	endNano := endTime.UnixNano()

	filename := fmt.Sprintf("%d_%d", startNano, endNano)
	target := path.Join(basePath, filename)

	if err := os.MkdirAll(basePath, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(target, value, 0644); err != nil {
		return err
	}
	return nil
}

// FIXME: this entire implementation is a hack
func (s *LabelBasedFileStore) ListKeys() ([]string, error) {
	dirDepths := make(map[string]int)

	err := filepath.Walk(s.DataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			depth := strings.Count(path, string(os.PathSeparator))
			dirDepths[path] = depth
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Find max depth
	maxDepth := 0
	for _, depth := range dirDepths {
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	ret := []string{}
	// Print only the deepest directories
	for dir, depth := range dirDepths {
		if depth == maxDepth {
			ret = append(ret, strings.TrimPrefix(dir, s.DataDir))
		}
	}

	return ret, nil
}

func (s *LabelBasedFileStore) Get(fullKey string) (filepaths []string, err error) {
	basePath := path.Join(s.DataDir, fullKey)
	ret := []string{}
	err = filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			ret = append(ret, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(ret)
	return ret, nil
}
