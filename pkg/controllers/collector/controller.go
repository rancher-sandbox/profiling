package collector

//FIXME: this is a hack, we should create a collector to CR and apply changes to the managed resources

import (
	"context"
	"fmt"

	"github.com/rancher-sandbox/profiling/pkg/controllers/common"
	"github.com/rancher-sandbox/profiling/pkg/operator/apis/v1alpha1"
	"github.com/rancher-sandbox/profiling/pkg/operator/generated/controllers/resources.cattle.io"
	"github.com/rancher/wrangler/v3/pkg/apply"
	"github.com/rancher/wrangler/v3/pkg/generated/controllers/apps"
	"github.com/rancher/wrangler/v3/pkg/generated/controllers/core"
	"github.com/rancher/wrangler/v3/pkg/relatedresource"

	"github.com/sirupsen/logrus"
)

type CollectorHandler struct {
	common.OperatorOptions
	Apply apply.Apply
}

func (h *CollectorHandler) OnChange(key string, stack *v1alpha1.PprofCollectorStack) (*v1alpha1.PprofCollectorStack, error) {
	// apply objects
	logrus.Warn("on change", key)
	applier := h.Apply.WithSetID(fmt.Sprintf("pprof-controller-collector-%s", h.OperatorOptions.OperatorName))
	if stack != nil {
		applier = applier.WithOwner(stack)
	}
	objs, err := h.Objects(stack)
	if err != nil {
		return nil, err
	}
	if err := applier.ApplyObjects(objs...); err != nil {
		logrus.Errorf("Failed to apply objects: %v", err)
	}
	return stack, nil
}

func (h *CollectorHandler) OnRemove(key string, stack *v1alpha1.PprofCollectorStack) (*v1alpha1.PprofCollectorStack, error) {
	// apply objects
	logrus.Warn("on remove", key)
	applier := h.Apply.WithSetID(fmt.Sprintf("pprof-controller-collector-%s", h.OperatorOptions.OperatorName))
	objs, err := h.Objects(stack)
	if err != nil {
		return nil, err
	}
	if err := applier.ApplyObjects(objs...); err != nil {
		logrus.Errorf("Failed to apply objects: %v", err)
	}
	return stack, nil
}

// deploys collector stack
func Register(
	ctx context.Context,
	operatorOpts common.OperatorOptions,
	core core.Interface,
	apps apps.Interface,
	pprofController *resources.Factory,
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
	resolver := relatedresource.OwnerResolver(true, v1alpha1.SchemeGroupVersion.String(), "PprofCollectorStack")
	relatedresource.Watch(
		ctx,
		"pprof-collector-watch",
		resolver,
		apps.V1().StatefulSet(),
		core.V1().Service(),
		core.V1().ConfigMap(),
		core.V1().PersistentVolumeClaim(),
	)
	name := operatorOpts.OperatorName
	pprofController.Resources().V1alpha1().PprofCollectorStack().OnChange(ctx, fmt.Sprintf("%s-collector-controller-change", name), handler.OnChange)
	pprofController.Resources().V1alpha1().PprofCollectorStack().OnRemove(ctx, fmt.Sprintf("%s-collector-controller-remove", name), handler.OnRemove)

}
