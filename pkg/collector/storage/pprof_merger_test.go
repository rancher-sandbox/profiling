package storage_test

import (
	"bytes"
	"testing"

	"github.com/alexandreLamarre/pprof-controller/pkg/collector/storage"
	"github.com/alexandreLamarre/pprof-controller/pkg/test/testdata"
	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/assert"
)

func TestPprofMerger(t *testing.T) {
	baseProfile := testdata.TestData("profile1.pb")
	incomingProfile := testdata.TestData("profile2.pb")
	merger := &storage.PprofMerger{}

	mergedProfile, err := merger.Merge(baseProfile, incomingProfile)
	assert.NoError(t, err)
	assert.NotNil(t, mergedProfile)

	output, err := profile.Parse(bytes.NewReader(mergedProfile))
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.NoError(t, output.CheckValid())
}
