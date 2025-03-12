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

type byteMerger struct{}

func (b *byteMerger) Merge(base []byte, incoming []byte) ([]byte, error) {
	return append(base, incoming...), nil
}

func TestStorage(t *testing.T) {
	pathName, err := os.MkdirTemp("/tmp", "collector_test")
	assert.NoError(t, err)
	defer os.RemoveAll(pathName)

	store := storage.NewLabelBasedFileStore(pathName, []string{labels.NamespaceLabel, labels.NameLabel}, &byteMerger{})
	assert.NotNil(t, store)
	now := time.Now()
	start := now.Add(-time.Minute)
	const profileType = "profile"
	err = store.Put(
		start,
		now,
		profileType,
		"pod/example1",
		map[string]string{
			labels.NamespaceLabel: "default",
			labels.NameLabel:      "example1",
		},
		[]byte("hello"),
	)

	assert.NoError(t, err)

	const expected = "default/example1/pod/example1"

	filepaths, err := store.Get("profile", expected)
	assert.NoError(t, err)
	assert.Len(t, filepaths, 1)
	assert.Equal(t, []string{
		path.Join(pathName, profileType, expected, fmt.Sprintf("%d_%d", start.UnixNano(), now.UnixNano())),
	}, filepaths)

	data, err := os.ReadFile(filepaths[0])
	assert.NoError(t, err)
	assert.Equal(t, "hello", string(data))

	start2 := time.Now()
	end := start.Add(time.Minute)
	err = store.Put(
		start2,
		end,
		profileType,
		"pod/example1",
		map[string]string{
			labels.NamespaceLabel: "default",
			labels.NameLabel:      "example1",
		},
		[]byte("world"),
	)

	assert.NoError(t, err)
	filepaths2, err := store.Get(profileType, expected)
	assert.NoError(t, err)
	assert.Len(t, filepaths2, 1)
	assert.Equal(t, []string{
		path.Join(pathName, profileType, expected, fmt.Sprintf("%d_%d", start.UnixNano(), end.UnixNano())),
	}, filepaths2)

	data2, err := os.ReadFile(filepaths2[0])
	assert.NoError(t, err)
	assert.Equal(t, "helloworld", string(data2))

	keys, err := store.ListKeys()
	assert.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Equal(t, []string{"/" + path.Join(profileType, expected)}, keys)

}
