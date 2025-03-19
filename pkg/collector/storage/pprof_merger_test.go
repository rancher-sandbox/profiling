package storage_test

import (
	"bytes"
	"testing"

	"github.com/google/pprof/profile"
	"github.com/rancher-sandbox/profiling/pkg/collector/storage"
	"github.com/rancher-sandbox/profiling/pkg/test/testdata"
	"github.com/stretchr/testify/assert"
)

type testCase struct {
	base     []byte
	incoming []byte
}

func TestPprofMerger(t *testing.T) {
	tcs := []testCase{
		{
			base:     testdata.TestData("profile1.pb"),
			incoming: testdata.TestData("profile2.pb"),
		},
		{
			base:     testdata.TestData("mutex1.pb"),
			incoming: testdata.TestData("mutex2.pb"),
		},
		{
			base:     testdata.TestData("alloc1.pb"),
			incoming: testdata.TestData("alloc2.pb"),
		},
		{
			base:     testdata.TestData("heap1.pb"),
			incoming: testdata.TestData("heap2.pb"),
		},
		{
			base:     testdata.TestData("blocks1.pb"),
			incoming: testdata.TestData("blocks2.pb"),
		},
		{
			base:     testdata.TestData("goroutine1.pb"),
			incoming: testdata.TestData("goroutine2.pb"),
		},
		// FIXME: pprof allows for merging out of order samples, this might not make sense
		{
			base:     testdata.TestData("profile2.pb"),
			incoming: testdata.TestData("profile1.pb"),
		},
	}

	for _, tc := range tcs {
		merger := &storage.PprofMerger{}

		mergedProfile, err := merger.Merge(tc.base, tc.incoming)
		assert.NoError(t, err)
		assert.NotNil(t, mergedProfile)

		output, err := profile.Parse(bytes.NewReader(mergedProfile))
		assert.NoError(t, err)
		assert.NotNil(t, output)
		assert.NoError(t, output.CheckValid())
	}

	failureTcs := []testCase{
		{
			base:     testdata.TestData("profile1.pb"),
			incoming: testdata.TestData("mutex1.pb"),
		},
		// pprof and trace are two different sets of go tools
		{
			base:     testdata.TestData("trace1.pb"),
			incoming: testdata.TestData("trace2.pb"),
		},
	}

	for _, tc := range failureTcs {
		merger := &storage.PprofMerger{}

		_, err := merger.Merge(tc.base, tc.incoming)
		assert.Error(t, err)
	}

}
