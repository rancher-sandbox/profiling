package collector

import (
	"github.com/alexandreLamarre/pprof-controller/pkg/controllers/common"
	"github.com/samber/lo"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (h *CollectorHandler) Objects() []runtime.Object {

	deploy := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.NamespacedCollectorName(h.OperatorOptions),
			Namespace: h.OperatorOptions.ControllerNamespace,
		},
		Spec: appsv1.DeploymentSpec{
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
					},
					Containers: []corev1.Container{
						{
							Name:            "pprof-collector",
							Image:           "docker.io/alex7285/pprof-collector:latest",
							ImagePullPolicy: corev1.PullAlways,
							Args: []string{
								"--config",
								"/var/lib/config.yaml",
								"--log-level",
								"debug",
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
	return []runtime.Object{&deploy}
}
