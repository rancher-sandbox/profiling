package v1alpha1

import (
	"fmt"
	"strings"

	"github.com/rancher-sandbox/profiling/pkg/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type GenericImage struct {
	Registry         string                        `json:"registry,omitempty"`
	Repo             string                        `json:"repo"`
	Image            string                        `json:"image"`
	Tag              string                        `json:"tag"`
	Sha              string                        `json:"sha,omitempty"`
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	ImagePullPolicy  corev1.PullPolicy             `json:"imagePullPolicy,omitempty"`
}

// TODO : there's likely a docker/OCI library that does this in a more consistent way
func (g *GenericImage) ImageStr() (string, error) {
	if g.Image == "" {
		return "", fmt.Errorf("image name is empty")
	}
	var parts []string

	// Add registry if present
	if g.Registry != "" {
		parts = append(parts, g.Registry)
	} else {
		parts = append(parts, "docker.io")
	}

	// Add repo if present
	if g.Repo != "" {
		parts = append(parts, g.Repo)
	}

	// Add image
	parts = append(parts, g.Image)

	imageRef := strings.Join(parts, "/")

	// Add tag if present
	if g.Tag != "" {
		imageRef += ":" + g.Tag
	} else {
		imageRef += ":latest"
	}

	if g.Sha != "" {
		imageRef += "@" + g.Sha
	}

	return imageRef, nil
}

type GenericStorage struct {
	// resource.MustParse("1Gi")
	DiskSpace string `json:"diskSpace"`
	// TODO : extend fields to handle pvcs / storage claims volumes
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PprofCollectorStack struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CollectorSpec   `json:"spec"`
	Status CollectorStatus `json:"status"`
}

type CollectorSpec struct {
	CollectorImage GenericImage   `json:"collectorImage"`
	ReloaderImage  GenericImage   `json:"reloaderImage"`
	Storage        GenericStorage `json:"storage"`
}

type CollectorStatus struct {
	// TODO
}

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
	// TODO : document config
	// TODO : pass in as pointer, and handle nil pointer with default config in controller
	Config config.GlobalSamplingConfig `json:"config,omitempty"`
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
}

type PprofStatus struct {
	// TODO
}
