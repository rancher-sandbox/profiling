// Code generated by main. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/alexandreLamarre/pprof-controller/pkg/operator/apis/v1alpha1"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/v3/pkg/generic"
	"github.com/rancher/wrangler/v3/pkg/schemes"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func init() {
	schemes.Register(v1alpha1.AddToScheme)
}

type Interface interface {
	PprofCollectorStack() PprofCollectorStackController
	PprofMonitor() PprofMonitorController
}

func New(controllerFactory controller.SharedControllerFactory) Interface {
	return &version{
		controllerFactory: controllerFactory,
	}
}

type version struct {
	controllerFactory controller.SharedControllerFactory
}

func (v *version) PprofCollectorStack() PprofCollectorStackController {
	return generic.NewController[*v1alpha1.PprofCollectorStack, *v1alpha1.PprofCollectorStackList](schema.GroupVersionKind{Group: "resources.cattle.io", Version: "v1alpha1", Kind: "PprofCollectorStack"}, "pprofcollectorstacks", true, v.controllerFactory)
}

func (v *version) PprofMonitor() PprofMonitorController {
	return generic.NewController[*v1alpha1.PprofMonitor, *v1alpha1.PprofMonitorList](schema.GroupVersionKind{Group: "resources.cattle.io", Version: "v1alpha1", Kind: "PprofMonitor"}, "pprofmonitors", true, v.controllerFactory)
}
