package monitor

import (
	"context"
	"fmt"
	"slices"
	"strings"

	collabels "github.com/alexandreLamarre/pprof-controller/pkg/collector/labels"
	"github.com/alexandreLamarre/pprof-controller/pkg/config"
	"github.com/alexandreLamarre/pprof-controller/pkg/controllers/common"
	"github.com/alexandreLamarre/pprof-controller/pkg/operator/apis/v1alpha1"
	"github.com/alexandreLamarre/pprof-controller/pkg/operator/generated/controllers/resources.cattle.io"
	pprofcontroller "github.com/alexandreLamarre/pprof-controller/pkg/operator/generated/controllers/resources.cattle.io/v1alpha1"
	"github.com/rancher/wrangler/v3/pkg/apply"
	"github.com/rancher/wrangler/v3/pkg/generated/controllers/core"
	v1core "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/v3/pkg/relatedresource"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

func Register(
	ctx context.Context,
	operatorOpts common.OperatorOptions,
	core core.Interface,
	k8sClient *kubernetes.Clientset,
	pprofFactory *resources.Factory,
	apply apply.Apply,
) {
	applier := apply.WithSetOwnerReference(true, false).WithCacheTypes(
		core.V1().ConfigMap(),
	)

	h := &PprofHandler{
		OperatorOptions: operatorOpts,
		// FIXME: hardcoded
		k8sClient: k8sClient,

		pprofFactory:   pprofFactory,
		podCache:       core.V1().Pod().Cache(),
		serviceCache:   core.V1().Service().Cache(),
		namespaceCache: core.V1().Namespace().Cache(),
		endpointCache:  core.V1().Endpoints().Cache(),
		monitorCache:   pprofFactory.Resources().V1alpha1().PprofMonitor().Cache(),
		apply:          applier,
	}

	resolver := func(namespace, name string, obj runtime.Object) ([]relatedresource.Key, error) {
		// TODO : this isn't necessarily accurate
		return []relatedresource.Key{
			{
				Namespace: namespace,
				Name:      name,
			},
		}, nil
	}

	relatedresource.Watch(ctx,
		"pprof-watch",
		resolver,
		h.pprofFactory.Resources().V1alpha1().PprofMonitor(),
		core.V1().Service(),
		core.V1().Endpoints(),
		core.V1().ConfigMap(),
	)
	// TODO : we want to watch config map changes to this namespace / owner

	pprofFactory.Resources().V1alpha1().PprofMonitor().OnChange(ctx, "pprofmonitors", h.OnPprofMonitorChange)

}

type PprofHandler struct {
	common.OperatorOptions
	k8sClient      *kubernetes.Clientset
	pprofFactory   *resources.Factory
	podCache       v1core.PodCache
	serviceCache   v1core.ServiceCache
	namespaceCache v1core.NamespaceCache
	endpointCache  v1core.EndpointsCache
	monitorCache   pprofcontroller.PprofMonitorCache
	apply          apply.Apply
}

func nsSelectorToList(nsList []*corev1.Namespace, sel v1alpha1.NamespaceSelector) []*corev1.Namespace {
	if sel.Any {
		return nsList
	}
	check := sel.MatchNames
	ret := []*corev1.Namespace{}
	for _, ns := range nsList {
		if slices.Contains(check, ns.Name) {
			ret = append(ret, ns)
		}
	}
	return ret
}

type serviceAndEndpoint struct {
	service *corev1.Service
	endp    *corev1.Endpoints
}

func endpSelectorToList(
	selectedNs []*corev1.Namespace,
	serviceCache v1core.ServiceCache,
	endpCache v1core.EndpointsCache,
	sel metav1.LabelSelector,
) ([]serviceAndEndpoint, error) {
	if len(sel.MatchLabels) > 0 {
		objSelector := labels.SelectorFromSet(sel.MatchLabels)
		totalEndpointList := []serviceAndEndpoint{}
		for _, ns := range selectedNs {

			// logrus.Warnf("%s : got %d total services", ns.Name, len(svcListA))
			svcList, err := serviceCache.List(ns.Name, objSelector)
			// logrus.Infof("%s : got %d services from selector : %v", ns.Name, len(svcList), sel.MatchLabels)
			if err != nil {
				return totalEndpointList, err
			}
			for _, svc := range svcList {
				endp, err := endpCache.Get(svc.Namespace, svc.Name)
				if err != nil {
					return totalEndpointList, err
				}
				totalEndpointList = append(totalEndpointList, serviceAndEndpoint{
					service: svc,
					endp:    endp,
				})
			}

		}
		return totalEndpointList, nil
	}
	if len(sel.MatchExpressions) > 0 {
		panic("not implemented : select.MatchExpressions")
	}

	// otherwise, by default return all endpoints
	totalEndpointList := []serviceAndEndpoint{}
	for _, ns := range selectedNs {
		svcList, err := serviceCache.List(ns.Name, labels.Everything())
		if err != nil {
			return []serviceAndEndpoint{}, err
		}
		for _, svc := range svcList {
			endp, err := endpCache.Get(svc.Namespace, svc.Name)
			if err != nil {
				return totalEndpointList, err
			}
			totalEndpointList = append(totalEndpointList, serviceAndEndpoint{
				service: svc,
				endp:    endp,
			})
		}
	}
	return totalEndpointList, nil
}

type directAddrAndFriendlyName struct {
	addr         string
	friendlyName string
}

func endpSubsetToAddresses(endpAndSvc serviceAndEndpoint, target v1alpha1.Endpoint) []directAddrAndFriendlyName {
	addrWithoutSchemePath := []directAddrAndFriendlyName{}
	// if target.Port != "" {

	// match to service ports
	svc := endpAndSvc.service
	svcPorts := []corev1.ServicePort{}
	for _, port := range svc.Spec.Ports {
		if target.Port == port.Name {
			svcPorts = append(svcPorts, port)
			continue
		}
		if target.TargetPort != nil {
			if target.TargetPort.Type == intstr.Int {
				if target.TargetPort.IntVal == port.Port {
					svcPorts = append(svcPorts, port)
					continue
				}
			}
			if target.TargetPort.Type == intstr.String {
				if target.TargetPort.StrVal == port.Name {
					svcPorts = append(svcPorts, port)
					continue
				}
			}
		}
	}

	// logrus.Infof("got svcPorts : %v", svcPorts)

	// correlate to endpoints
	subsets := endpAndSvc.endp.Subsets

	for _, subset := range subsets {
		ports := subset.Ports
		actualPorts := []int{}
		for _, port := range ports {
			for _, svcPort := range svcPorts {
				if svcPort.Name != "" {
					if port.Name == svcPort.Name {
						actualPorts = append(actualPorts, int(port.Port))
					}
				}
				if svcPort.TargetPort.Type == intstr.Int {
					if port.Port == svcPort.TargetPort.IntVal {
						actualPorts = append(actualPorts, int(port.Port))
					}
				}
				if svcPort.TargetPort.Type == intstr.String {
					if port.Name == svcPort.TargetPort.StrVal {
						actualPorts = append(actualPorts, int(port.Port))
					}
				}
			}
		}
		actualPorts = lo.Uniq(actualPorts)
		// logrus.Infof("got actualPorts : %v", actualPorts)

		for _, ip := range subset.Addresses {
			for _, port := range actualPorts {
				addrWithoutSchemePath = append(addrWithoutSchemePath, directAddrAndFriendlyName{
					addr: fmt.Sprintf("%s:%d", ip.IP, port),
					// Note : do not include any '/' in the friendly name it will confuse the hacky storage implementation
					friendlyName: ip.TargetRef.Name,
				})
			}
		}
	}

	// logrus.Info("got addrWithoutSchemePath : ", addrWithoutSchemePath)
	ret := []directAddrAndFriendlyName{}
	for _, addrAndName := range addrWithoutSchemePath {
		newAddr := addrAndName.addr
		scheme := strings.TrimSpace(target.Scheme)
		if scheme == "" {
			scheme = "http"
		}
		newAddr = fmt.Sprintf("%s://%s", scheme, newAddr)
		path := strings.TrimSpace(target.Path)
		if path != "" {
			path = path + "/debug/pprof"
			newAddr = fmt.Sprintf("%s/%s", newAddr, path)
		}
		addrAndName.addr = newAddr
		ret = append(ret, addrAndName)
	}

	return ret
}

type MonitorAndAddresses struct {
	monitor      *v1alpha1.PprofMonitor
	addresses    []directAddrAndFriendlyName
	k8sname      string
	k8snamespace string
}

func (h *PprofHandler) OnPprofMonitorChange(_ string, monitor *v1alpha1.PprofMonitor) (*v1alpha1.PprofMonitor, error) {
	logger := h.Logger.With("handler", "onPprofMonitorChange")
	apply := h.apply.WithSetID(h.OperatorName + "-config")

	nsList, err := h.namespaceCache.List(labels.Everything())
	if err != nil {
		logger.With("err", err).Error("failed to list namespaces")
		return monitor, err
	}

	monitorList := []*v1alpha1.PprofMonitor{}
	for _, ns := range nsList {
		pprofs, err := h.monitorCache.List(ns.Name, labels.Everything())
		if err != nil {
			return monitor, err
		}
		monitorList = append(monitorList, pprofs...)
	}

	constructed := []MonitorAndAddresses{}

	for _, mon := range monitorList {
		selectedNs := nsSelectorToList(nsList, mon.Spec.NamespaceSelector)

		endpAndServiceList, err := endpSelectorToList(selectedNs, h.serviceCache, h.endpointCache, mon.Spec.Selector)
		if err != nil {
			return monitor, nil
		}

		logger.With("monitor", mon.Name, "endpoints", len(endpAndServiceList)).Debug("got endpoints to process")

		for _, endp := range endpAndServiceList {
			addresses := endpSubsetToAddresses(endp, mon.Spec.Endpoint)
			slices.SortFunc(addresses, func(a, b directAddrAndFriendlyName) int {
				if a.friendlyName < b.friendlyName {
					return -1
				}
				if a.friendlyName > b.friendlyName {
					return 1
				}
				return 0
			})
			// TODO : at some point in this pipeline we need to de-duplicate addresses : Monitors can point to the same address, pprof typically precious multiple concurrent
			// open requests.
			constructed = append(constructed, MonitorAndAddresses{
				monitor:      mon,
				addresses:    addresses,
				k8sname:      endp.endp.Name,
				k8snamespace: endp.endp.Namespace,
			})
		}
	}

	// TODO : verify this always results in a deterministic order
	slices.SortFunc(constructed, func(a, b MonitorAndAddresses) int {
		if a.monitor.Namespace > b.monitor.Namespace {
			return 1
		}
		if a.monitor.Namespace < b.monitor.Namespace {
			return -1
		}
		if a.monitor.Name < b.monitor.Name {
			return -1
		}
		if a.monitor.Name > b.monitor.Name {
			return 1
		}
		return 0
	})

	logger.With("len", len(constructed)).Debug("got total monitor configurations to process")

	cfg := config.CollectorConfig{
		SelfTelemetry: nil,
		Monitors:      []*config.MonitorConfig{},
	}

	for _, mon := range constructed {
		for _, addr := range mon.addresses {
			cfg.Monitors = append(cfg.Monitors, &config.MonitorConfig{
				Name:     addr.friendlyName,
				Endpoint: addr.addr,
				Labels: map[string]string{
					collabels.NamespaceLabel: mon.k8snamespace,
					collabels.NameLabel:      mon.k8sname,
				},
				GlobalSampling: mon.monitor.Spec.Config,
			})
		}
	}
	// FIXME: configurable
	cfg.SelfTelemetry = &config.SelfTelemetryConfig{
		PprofPort:       6060,
		IntervalSeconds: 120,
	}

	objs, err := h.Objects(cfg)
	logrus.Debug("got objects : ", objs)
	if err != nil {
		logrus.Errorf("failed to generate objects : %s", err)
		return monitor, err
	}
	if err := apply.ApplyObjects(objs...); err != nil {
		logrus.Errorf("failed to apply objects : %s", err)
		return monitor, err
	}

	return monitor, nil
}
