package monitor

import (
	"github.com/alexandreLamarre/pprof-controller/pkg/config"
	"github.com/alexandreLamarre/pprof-controller/pkg/controllers/common"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
)

func (h *PprofHandler) Objects(config config.CollectorConfig) ([]runtime.Object, error) {
	//TODO: config validation

	data, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.NamespacedConfigName(h.OperatorOptions),
			Namespace: h.ControllerNamespace,
		},
		Data: map[string]string{
			"config.yaml": string(data),
		},
	}

	// logrus.Warn("configMap: ", configMap)

	return []runtime.Object{
		configMap,
	}, nil
}
