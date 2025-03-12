package storage

import (
	"bytes"

	"github.com/google/pprof/profile"
)

type PprofMerger struct{}

func (p *PprofMerger) Merge(base []byte, incoming []byte) ([]byte, error) {

	baseProfile, err := profile.Parse(bytes.NewReader(base))
	if err != nil {
		return nil, err
	}
	incomingProfile, err := profile.Parse(bytes.NewReader(incoming))
	if err != nil {
		return nil, err
	}

	newProfile, err := profile.Merge([]*profile.Profile{baseProfile, incomingProfile})
	if err != nil {
		return nil, err
	}
	b := bytes.NewBuffer([]byte{})
	if err := newProfile.Write(b); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
