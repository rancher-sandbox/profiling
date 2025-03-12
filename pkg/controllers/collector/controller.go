package collector

//FIXME: this is a hack, we should create a collector to CR and apply changes to the managed resources

import (
	"context"
	"fmt"

	"github.com/alexandreLamarre/pprof-controller/pkg/controllers/common"
	"github.com/rancher/wrangler/v3/pkg/apply"
	"github.com/rancher/wrangler/v3/pkg/generated/controllers/apps"
	"github.com/rancher/wrangler/v3/pkg/generated/controllers/core"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
)

type CollectorHandler struct {
	common.OperatorOptions
	Apply apply.Apply
}

func (h *CollectorHandler) OnChange(key string, deploy *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	// apply objects
	logrus.Warn("on change", key)
	applier := h.Apply.WithSetID(fmt.Sprintf("pprof-controller-collector-%s", h.OperatorOptions.OperatorName))
	if err := applier.ApplyObjects(h.Objects()...); err != nil {
		logrus.Errorf("Failed to apply objects: %v", err)
	}
	return deploy, nil
}

func (h *CollectorHandler) OnRemove(key string, deploy *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	// apply objects
	logrus.Warn("on remove", key)
	applier := h.Apply.WithSetID(fmt.Sprintf("pprof-controller-collector-%s", h.OperatorOptions.OperatorName))
	if err := applier.ApplyObjects(h.Objects()...); err != nil {
		logrus.Errorf("Failed to apply objects: %v", err)
	}
	return deploy, nil
}

// deploys collector stack
func Register(
	ctx context.Context,
	operatorOpts common.OperatorOptions,
	core core.Interface,
	apps apps.Interface,
	applier apply.Apply,
) {

	applier = applier.WithSetOwnerReference(true, false).WithCacheTypes(
		apps.V1().StatefulSet(),
		core.V1().ConfigMap(),
		core.V1().Service(),
		core.V1().PersistentVolumeClaim(),
	)
	handler := &CollectorHandler{
		OperatorOptions: operatorOpts,
		Apply:           applier,
	}

	apps.V1().StatefulSet().OnChange(ctx, "collector-controller", handler.OnChange)
	apps.V1().StatefulSet().OnRemove(ctx, "collector-controller", handler.OnRemove)

}
