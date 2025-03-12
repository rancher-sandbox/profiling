package config

type CollectorConfig struct {
	SelfTelemetry *SelfTelemetryConfig `json:"self_telemetry" yaml:"self_telemetry"`

	Monitors []*MonitorConfig `json:"monitors" yaml:"monitors"`
}

type SelfTelemetryConfig struct {
	PprofPort       int `json:"pprof_port" yaml:"pprof_port"`
	IntervalSeconds int `json:"interval_seconds" yaml:"interval_seconds"`
}

type SamplerConfig struct {
	Seconds int `json:"seconds" yaml:"seconds"`
}

type MonitorConfig struct {
	// Name starting with `__`` are reserved for internal monitors
	Name           string               `json:"name" yaml:"name"`
	Endpoint       string               `json:"endpoint" yaml:"endpoint"`
	Labels         map[string]string    `json:"labels" yaml:"labels"`
	GlobalSampling GlobalSamplingConfig `json:"sampling" yaml:"sampling"`
}

type GlobalSamplingConfig struct {
	Allocs      *SamplerConfig `json:"allocs" yaml:"allocs"`
	Block       *SamplerConfig `json:"block" yaml:"block"`
	Goroutine   *SamplerConfig `json:"goroutine" yaml:"goroutine"`
	Heap        *SamplerConfig `json:"heap" yaml:"heap"`
	Mutex       *SamplerConfig `json:"mutex" yaml:"mutex"`
	Profile     *SamplerConfig `json:"profile" yaml:"profile"`
	ThreadCrate *SamplerConfig `json:"threadcreate" yaml:"threadcreate"`
	Trace       *SamplerConfig `json:"trace" yaml:"trace"`
	// TODO : unused fields
	Compression string
}

// FIXME: didn't check this is fully working as expected
func (g *GlobalSamplingConfig) DeepCopyInto(out *GlobalSamplingConfig) {
	*out = *g
	if g.Allocs != nil {
		allocs := *g.Allocs
		out.Allocs = &allocs
	}
	if g.Block != nil {
		block := *g.Block
		out.Block = &block
	}
	if g.Goroutine != nil {
		goroutine := *g.Goroutine
		out.Goroutine = &goroutine
	}
	if g.Heap != nil {
		heap := *g.Heap
		out.Heap = &heap
	}
	if g.Mutex != nil {
		mutex := *g.Mutex
		out.Mutex = &mutex
	}
	if g.Profile != nil {
		profile := *g.Profile
		out.Profile = &profile
	}
	if g.ThreadCrate != nil {
		threadCreate := *g.ThreadCrate
		out.ThreadCrate = &threadCreate
	}
	if g.Trace != nil {
		trace := *g.Trace
		out.Trace = &trace
	}
}
