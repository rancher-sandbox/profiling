// Code generated by main. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"sync"
	"time"

	v1alpha1 "github.com/rancher-sandbox/profiling/pkg/operator/apis/v1alpha1"
	"github.com/rancher/wrangler/v3/pkg/apply"
	"github.com/rancher/wrangler/v3/pkg/condition"
	"github.com/rancher/wrangler/v3/pkg/generic"
	"github.com/rancher/wrangler/v3/pkg/kv"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// PprofCollectorStackController interface for managing PprofCollectorStack resources.
type PprofCollectorStackController interface {
	generic.ControllerInterface[*v1alpha1.PprofCollectorStack, *v1alpha1.PprofCollectorStackList]
}

// PprofCollectorStackClient interface for managing PprofCollectorStack resources in Kubernetes.
type PprofCollectorStackClient interface {
	generic.ClientInterface[*v1alpha1.PprofCollectorStack, *v1alpha1.PprofCollectorStackList]
}

// PprofCollectorStackCache interface for retrieving PprofCollectorStack resources in memory.
type PprofCollectorStackCache interface {
	generic.CacheInterface[*v1alpha1.PprofCollectorStack]
}

// PprofCollectorStackStatusHandler is executed for every added or modified PprofCollectorStack. Should return the new status to be updated
type PprofCollectorStackStatusHandler func(obj *v1alpha1.PprofCollectorStack, status v1alpha1.CollectorStatus) (v1alpha1.CollectorStatus, error)

// PprofCollectorStackGeneratingHandler is the top-level handler that is executed for every PprofCollectorStack event. It extends PprofCollectorStackStatusHandler by a returning a slice of child objects to be passed to apply.Apply
type PprofCollectorStackGeneratingHandler func(obj *v1alpha1.PprofCollectorStack, status v1alpha1.CollectorStatus) ([]runtime.Object, v1alpha1.CollectorStatus, error)

// RegisterPprofCollectorStackStatusHandler configures a PprofCollectorStackController to execute a PprofCollectorStackStatusHandler for every events observed.
// If a non-empty condition is provided, it will be updated in the status conditions for every handler execution
func RegisterPprofCollectorStackStatusHandler(ctx context.Context, controller PprofCollectorStackController, condition condition.Cond, name string, handler PprofCollectorStackStatusHandler) {
	statusHandler := &pprofCollectorStackStatusHandler{
		client:    controller,
		condition: condition,
		handler:   handler,
	}
	controller.AddGenericHandler(ctx, name, generic.FromObjectHandlerToHandler(statusHandler.sync))
}

// RegisterPprofCollectorStackGeneratingHandler configures a PprofCollectorStackController to execute a PprofCollectorStackGeneratingHandler for every events observed, passing the returned objects to the provided apply.Apply.
// If a non-empty condition is provided, it will be updated in the status conditions for every handler execution
func RegisterPprofCollectorStackGeneratingHandler(ctx context.Context, controller PprofCollectorStackController, apply apply.Apply,
	condition condition.Cond, name string, handler PprofCollectorStackGeneratingHandler, opts *generic.GeneratingHandlerOptions) {
	statusHandler := &pprofCollectorStackGeneratingHandler{
		PprofCollectorStackGeneratingHandler: handler,
		apply:                                apply,
		name:                                 name,
		gvk:                                  controller.GroupVersionKind(),
	}
	if opts != nil {
		statusHandler.opts = *opts
	}
	controller.OnChange(ctx, name, statusHandler.Remove)
	RegisterPprofCollectorStackStatusHandler(ctx, controller, condition, name, statusHandler.Handle)
}

type pprofCollectorStackStatusHandler struct {
	client    PprofCollectorStackClient
	condition condition.Cond
	handler   PprofCollectorStackStatusHandler
}

// sync is executed on every resource addition or modification. Executes the configured handlers and sends the updated status to the Kubernetes API
func (a *pprofCollectorStackStatusHandler) sync(key string, obj *v1alpha1.PprofCollectorStack) (*v1alpha1.PprofCollectorStack, error) {
	if obj == nil {
		return obj, nil
	}

	origStatus := obj.Status.DeepCopy()
	obj = obj.DeepCopy()
	newStatus, err := a.handler(obj, obj.Status)
	if err != nil {
		// Revert to old status on error
		newStatus = *origStatus.DeepCopy()
	}

	if a.condition != "" {
		if errors.IsConflict(err) {
			a.condition.SetError(&newStatus, "", nil)
		} else {
			a.condition.SetError(&newStatus, "", err)
		}
	}
	if !equality.Semantic.DeepEqual(origStatus, &newStatus) {
		if a.condition != "" {
			// Since status has changed, update the lastUpdatedTime
			a.condition.LastUpdated(&newStatus, time.Now().UTC().Format(time.RFC3339))
		}

		var newErr error
		obj.Status = newStatus
		newObj, newErr := a.client.UpdateStatus(obj)
		if err == nil {
			err = newErr
		}
		if newErr == nil {
			obj = newObj
		}
	}
	return obj, err
}

type pprofCollectorStackGeneratingHandler struct {
	PprofCollectorStackGeneratingHandler
	apply apply.Apply
	opts  generic.GeneratingHandlerOptions
	gvk   schema.GroupVersionKind
	name  string
	seen  sync.Map
}

// Remove handles the observed deletion of a resource, cascade deleting every associated resource previously applied
func (a *pprofCollectorStackGeneratingHandler) Remove(key string, obj *v1alpha1.PprofCollectorStack) (*v1alpha1.PprofCollectorStack, error) {
	if obj != nil {
		return obj, nil
	}

	obj = &v1alpha1.PprofCollectorStack{}
	obj.Namespace, obj.Name = kv.RSplit(key, "/")
	obj.SetGroupVersionKind(a.gvk)

	if a.opts.UniqueApplyForResourceVersion {
		a.seen.Delete(key)
	}

	return nil, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects()
}

// Handle executes the configured PprofCollectorStackGeneratingHandler and pass the resulting objects to apply.Apply, finally returning the new status of the resource
func (a *pprofCollectorStackGeneratingHandler) Handle(obj *v1alpha1.PprofCollectorStack, status v1alpha1.CollectorStatus) (v1alpha1.CollectorStatus, error) {
	if !obj.DeletionTimestamp.IsZero() {
		return status, nil
	}

	objs, newStatus, err := a.PprofCollectorStackGeneratingHandler(obj, status)
	if err != nil {
		return newStatus, err
	}
	if !a.isNewResourceVersion(obj) {
		return newStatus, nil
	}

	err = generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects(objs...)
	if err != nil {
		return newStatus, err
	}
	a.storeResourceVersion(obj)
	return newStatus, nil
}

// isNewResourceVersion detects if a specific resource version was already successfully processed.
// Only used if UniqueApplyForResourceVersion is set in generic.GeneratingHandlerOptions
func (a *pprofCollectorStackGeneratingHandler) isNewResourceVersion(obj *v1alpha1.PprofCollectorStack) bool {
	if !a.opts.UniqueApplyForResourceVersion {
		return true
	}

	// Apply once per resource version
	key := obj.Namespace + "/" + obj.Name
	previous, ok := a.seen.Load(key)
	return !ok || previous != obj.ResourceVersion
}

// storeResourceVersion keeps track of the latest resource version of an object for which Apply was executed
// Only used if UniqueApplyForResourceVersion is set in generic.GeneratingHandlerOptions
func (a *pprofCollectorStackGeneratingHandler) storeResourceVersion(obj *v1alpha1.PprofCollectorStack) {
	if !a.opts.UniqueApplyForResourceVersion {
		return
	}

	key := obj.Namespace + "/" + obj.Name
	a.seen.Store(key, obj.ResourceVersion)
}
