package storage

// FIXME: this entire implementation is a mess, done for speed / demonstration purposes

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

type Merger interface {
	Merge(base []byte, incoming []byte) ([]byte, error)
}

type Store interface {
	Put(startTime, endTime time.Time, profileType string, key string, labels map[string]string, value []byte) error
	ListKeys() ([]string, error)
	GroupKeys() (map[string]map[string]map[string][]string, error)
	Get(profileType, key string) (filepaths []string, err error)
}

type LabelBasedFileStore struct {
	DataDir string

	IndexBy []string
	Merger  Merger
}

func NewLabelBasedFileStore(dataDir string, indexBy []string, merger Merger) *LabelBasedFileStore {
	return &LabelBasedFileStore{
		DataDir: dataDir,
		IndexBy: indexBy,
		Merger:  merger,
	}
}

var _ Store = (*LabelBasedFileStore)(nil)

func (s *LabelBasedFileStore) basePath(
	labels map[string]string,
	profileType string,
	key string,
) (string, error) {
	base := path.Join(s.DataDir, profileType)
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
	basePath, err := s.basePath(labels, profileType, key)
	if err != nil {
		return err
	}

	startNano := startTime.UnixNano()
	endNano := endTime.UnixNano()

	defaultFileName := fmt.Sprintf("%d_%d", startNano, endNano)

	if err := os.MkdirAll(basePath, 0755); err != nil {
		return err
	}
	files := []string{}
	pathErr := filepath.WalkDir(basePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if pathErr != nil {
		return pathErr
	}

	slices.Sort(files)
	var filename string
	if len(files) > 0 {
		lastFile := files[len(files)-1]
		oldFileName := path.Base(lastFile)
		strings.Split(oldFileName, "_")
		oldStartStr := strings.Split(oldFileName, "_")[0]
		startNano, err := strconv.ParseInt(oldStartStr, 10, 64)
		if err != nil {
			return err
		}
		filename = fmt.Sprintf("%d_%d", startNano, endNano)
		data, err := os.ReadFile(lastFile)
		if err != nil {
			return err
		}
		merged, err := s.Merger.Merge(data, value)
		if err != nil {
			return err
		}
		value = merged
	} else {
		filename = defaultFileName
	}
	target := path.Join(basePath, filename)
	if err := os.WriteFile(target, value, 0755); err != nil {
		return err
	}
	return nil
}

// namespace -> name -> resource -> keys
func (s *LabelBasedFileStore) GroupKeys() (map[string]map[string]map[string][]string, error) {
	keys, err := s.ListKeys()
	if err != nil {
		return nil, err
	}
	ret := make(map[string]map[string]map[string][]string)
	for _, key := range keys {
		key = strings.TrimPrefix(key, string(os.PathSeparator))
		parts := strings.Split(key, string(os.PathSeparator))
		if len(parts) < 4 || len(parts) > 4 {
			return nil, fmt.Errorf("invalid key %s", key)
		}
		// profileType := parts[0]
		namespace := parts[1]
		name := parts[2]
		resourceName := parts[3]

		if _, ok := ret[namespace]; !ok {
			ret[namespace] = make(map[string]map[string][]string)
		}
		if _, ok := ret[namespace][name]; !ok {
			ret[namespace][name] = make(map[string][]string)
		}
		if _, ok := ret[namespace][name][resourceName]; !ok {
			ret[namespace][name][resourceName] = []string{}
		}
		ret[namespace][name][resourceName] = append(ret[namespace][name][resourceName], key)
	}

	for nsKey, ns := range ret {
		for nameKey, name := range ns {
			for nameRsc, keys := range name {
				tempKeys := []string{}
				tempKeys = append(tempKeys, keys...)
				slices.Sort(tempKeys)
				ret[nsKey][nameKey][nameRsc] = tempKeys
			}
		}
	}

	return ret, nil
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

func (s *LabelBasedFileStore) Get(profileType, key string) (filepaths []string, err error) {
	basePath := path.Join(s.DataDir, profileType)
	basePath = path.Join(basePath, key)
	ret := []string{}
	err = filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
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
