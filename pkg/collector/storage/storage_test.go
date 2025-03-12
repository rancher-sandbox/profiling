package storage_test

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/alexandreLamarre/pprof-controller/pkg/collector/labels"
	"github.com/alexandreLamarre/pprof-controller/pkg/collector/storage"
	"github.com/stretchr/testify/assert"
)

func TestStorage(t *testing.T) {
	pathName, err := os.MkdirTemp("/tmp", "collector_test")
	assert.NoError(t, err)
	defer os.RemoveAll(pathName)

	store := storage.NewLabelBasedFileStore(pathName, []string{labels.NamespaceLabel, labels.NameLabel})
	assert.NotNil(t, store)
	now := time.Now()
	start := now.Add(-time.Minute)
	err = store.Put(
		start,
		now,
		"profile",
		"pod/example1",
		map[string]string{
			labels.NamespaceLabel: "default",
			labels.NameLabel:      "example1",
		},
		[]byte("test"),
	)

	assert.NoError(t, err)

	const expected = "default/example1/pod/example1/profile"

	filepaths, err := store.Get(expected)
	assert.NoError(t, err)
	assert.Len(t, filepaths, 1)
	assert.Equal(t, filepaths, []string{
		path.Join(pathName, expected, fmt.Sprintf("%d_%d", start.UnixNano(), now.UnixNano())),
	})
	start2 := time.Now()
	end := start.Add(time.Minute)
	err = store.Put(
		start2,
		end,
		"profile",
		"pod/example1",
		map[string]string{
			labels.NamespaceLabel: "default",
			labels.NameLabel:      "example1",
		},
		[]byte("test"),
	)

	assert.NoError(t, err)
	filepaths2, err := store.Get(expected)
	assert.NoError(t, err)
	assert.Len(t, filepaths2, 2)
	assert.Equal(t, filepaths2, []string{
		path.Join(pathName, expected, fmt.Sprintf("%d_%d", start.UnixNano(), now.UnixNano())),
		path.Join(pathName, expected, fmt.Sprintf("%d_%d", start2.UnixNano(), end.UnixNano())),
	})

}
