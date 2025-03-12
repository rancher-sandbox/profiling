// Code generated by main. DO NOT EDIT.

// +k8s:deepcopy-gen=package
// +groupName=resources.cattle.io
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PprofMonitorList is a list of PprofMonitor resources
type PprofMonitorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []PprofMonitor `json:"items"`
}

func NewPprofMonitor(namespace, name string, obj PprofMonitor) *PprofMonitor {
	obj.APIVersion, obj.Kind = SchemeGroupVersion.WithKind("PprofMonitor").ToAPIVersionAndKind()
	obj.Name = name
	obj.Namespace = namespace
	return &obj
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PprofCollectorStackList is a list of PprofCollectorStack resources
type PprofCollectorStackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []PprofCollectorStack `json:"items"`
}

func NewPprofCollectorStack(namespace, name string, obj PprofCollectorStack) *PprofCollectorStack {
	obj.APIVersion, obj.Kind = SchemeGroupVersion.WithKind("PprofCollectorStack").ToAPIVersionAndKind()
	obj.Name = name
	obj.Namespace = namespace
	return &obj
}
