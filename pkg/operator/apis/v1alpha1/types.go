package v1alpha1

import (
	"github.com/alexandreLamarre/pprof-controller/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PprofMonitor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PprofSpec   `json:"spec"`
	Status PprofStatus `json:"status"`
}

type PprofSpec struct {
	// Selector to select Service objects.
	Selector metav1.LabelSelector `json:"selector"`
	// Selector to select which namespaces the Kubernetes Service objects are discovered from.
	NamespaceSelector NamespaceSelector `json:"namespaceSelector,omitempty"`

	Endpoint Endpoint `json:"endpoint,omitempty"`
}

type NamespaceSelector struct {
	// Boolean describing whether all namespaces are selected in contrast to a
	// list restricting them.
	Any bool `json:"any,omitempty"`
	// List of namespace names to select from.
	MatchNames []string `json:"matchNames,omitempty"`
}

type Endpoint struct {
	// Name of the service port this endpoint refers to. Mutually exclusive with targetPort.
	Port string `json:"port,omitempty"`
	// Name or number of the target port of the Pod behind the Service, the port must be specified with container port property. Mutually exclusive with port.
	TargetPort *intstr.IntOrString `json:"targetPort,omitempty"`
	// HTTP path prefix to collect pprof profiles from
	// If empty, uses the default value "/debug/pprof".
	// Appends "/debug/pprof" to the URL if the path is not empty.
	Path string `json:"path,omitempty"`
	// HTTP scheme to use for scraping.
	// `http` and `https` are the expected values unless you rewrite the `__scheme__` label via relabeling.
	// If empty, Prometheus uses the default value `http`.
	// +kubebuilder:validation:Enum=http;https
	Scheme string `json:"scheme,omitempty"`
	// TODO : document config
	Config config.MonitorConfig `json:"config,omitempty"`
}

type PprofStatus struct {
}
