package ingest_test

import (
	"testing"

	"github.com/rancher-sandbox/profiling/pkg/collector/ingest"
	"github.com/rancher-sandbox/profiling/pkg/test/testdata"
	"github.com/stretchr/testify/assert"
	profilespb "go.opentelemetry.io/proto/otlp/profiles/v1development"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestConvert(t *testing.T) {
	data := testdata.TestData("profile.json")
	var a profilespb.Profile
	assert.NoError(t, protojson.Unmarshal(data, &a))

	prof := ingest.Convert(&a)
	assert.NoError(t, prof.CheckValid())
}
