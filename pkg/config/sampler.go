package config

type CollectorConfig struct {
	SelfTelemetry *SelfTelemetryConfig `json:"self_telemetry" yaml:"self_telemetry"`

	Monitors []*MonitorConfig `json:"monitors" yaml:"monitors"`
}

type SelfTelemetryConfig struct {
	PprofPort int `json:"pprof_port" yaml:"pprof_port"`
}

type SamplerConfig struct {
	Seconds int `json:"seconds" yaml:"seconds"`
}

type MonitorConfig struct {
	// Name starting with `__`` are reserved for internal monitors
	Name           string               `json:"name" yaml:"name"`
	Endpoint       string               `json:"endpoint" yaml:"endpoint"`
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
