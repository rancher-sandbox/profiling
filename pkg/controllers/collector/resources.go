package collector

import (
	"fmt"

	"github.com/rancher-sandbox/profiling/pkg/controllers/common"
	"github.com/rancher-sandbox/profiling/pkg/operator/apis/v1alpha1"
	"github.com/samber/lo"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (h *CollectorHandler) Objects(stack *v1alpha1.PprofCollectorStack) ([]runtime.Object, error) {
	if stack == nil {
		return []runtime.Object{}, nil
	}

	spaceQ, err := resource.ParseQuantity(stack.Spec.Storage.DiskSpace)
	if err != nil {
		return nil, fmt.Errorf("failed to parse disk space quantity: %v", err)
	}
	collectorImage, err := stack.Spec.CollectorImage.ImageStr()
	if err != nil {
		return nil, fmt.Errorf("failed to parse collector image: %v", err)
	}

	reloaderImage, err := stack.Spec.ReloaderImage.ImageStr()
	if err != nil {
		return nil, fmt.Errorf("failed to parse reloader image: %v", err)
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.NamespacedCollectorName(h.OperatorOptions),
			Namespace: h.OperatorOptions.ControllerNamespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"cattle.io/managed-by": h.OperatorOptions.OperatorName,
				"cattle.io/app":        "pprof-collector",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "web",
					TargetPort: intstr.FromString("web"),
					Port:       8989,
				},
			},
		},
	}
	mode := corev1.PersistentVolumeFilesystem
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.NamespacedCollectorName(h.OperatorOptions) + "-data",
			Namespace: h.OperatorOptions.ControllerNamespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: spaceQ,
				},
			},
			VolumeMode: &mode,
		},
	}

	ss := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.NamespacedCollectorName(h.OperatorOptions),
			Namespace: h.OperatorOptions.ControllerNamespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: lo.ToPtr(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"cattle.io/managed-by": h.OperatorOptions.OperatorName,
					"cattle.io/app":        "pprof-collector",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      common.NamespacedCollectorName(h.OperatorOptions),
					Namespace: h.OperatorOptions.ControllerNamespace,
					Labels: map[string]string{
						"cattle.io/managed-by": h.OperatorOptions.OperatorName,
						"cattle.io/app":        "pprof-collector",
					},
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: append(stack.Spec.CollectorImage.ImagePullSecrets, stack.Spec.ReloaderImage.ImagePullSecrets...),
					Volumes: []corev1.Volume{
						{
							Name: "pprof-collector-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									Items: []corev1.KeyToPath{
										{
											Key:  "config.yaml",
											Path: "config.yaml",
										},
									},
									LocalObjectReference: corev1.LocalObjectReference{
										Name: common.NamespacedConfigName(h.OperatorOptions),
									},
								},
							},
						},
						{
							Name: "pprof-collector-data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: common.NamespacedCollectorName(h.OperatorOptions) + "-data",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "pprof-collector",
							Image:           collectorImage,
							ImagePullPolicy: stack.Spec.CollectorImage.ImagePullPolicy,
							Command: []string{
								"collector",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "web",
									ContainerPort: 8989,
									HostPort:      8989,
								},
							},
							Args: []string{
								"--config",
								"/var/lib/config.yaml",
								"--log-level",
								"info",
								"--data-dir",
								"/var/collector/data",
								"--web-port",
								"8989",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "pprof-collector-config",
									ReadOnly:  true,
									MountPath: "/var/lib",
								},
								{
									Name:      "pprof-collector-data",
									ReadOnly:  false,
									MountPath: "/var/collector/data",
								},
							},
						},
						{
							Name: "config-reloader",
							// Image: "ghcr.io/jimmidyson/configmap-reload:dev",
							Image:           reloaderImage,
							ImagePullPolicy: stack.Spec.ReloaderImage.ImagePullPolicy,
							Args: []string{
								"--volume-dir=/var/lib",
								"--webhook-url=http://localhost:8989/reload",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "pprof-collector-config",
									ReadOnly:  true,
									MountPath: "/var/lib",
								},
							},
						},
					},
				},
			},
		},
	}
	return []runtime.Object{service, pvc, ss}, nil
}
