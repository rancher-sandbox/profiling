package operator

import (
	"context"

	"github.com/rancher-sandbox/profiling/pkg/controllers/collector"
	"github.com/rancher-sandbox/profiling/pkg/controllers/common"
	"github.com/rancher-sandbox/profiling/pkg/controllers/monitor"
	"github.com/rancher-sandbox/profiling/pkg/operator/generated/controllers/resources.cattle.io"

	"github.com/rancher/wrangler/v3/pkg/apply"
	"github.com/rancher/wrangler/v3/pkg/generated/controllers/apps"
	core "github.com/rancher/wrangler/v3/pkg/generated/controllers/core"

	// v1core "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/v3/pkg/start"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type AppContext struct {
	PprofFactory *resources.Factory
	ClientSet    *kubernetes.Clientset
	Apply        apply.Apply
	Core         core.Interface
	Apps         apps.Interface

	starters []start.Starter
}

func Setup(cfg *rest.Config) (*AppContext, error) {
	pp, err := resources.NewFactoryFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	core, err := core.NewFactoryFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	apps, err := apps.NewFactoryFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	discovery, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}

	applier := apply.New(discovery, apply.NewClientFactory(cfg))
	return &AppContext{
		PprofFactory: pp,
		Core:         core.Core(),
		Apps:         apps.Apps(),
		ClientSet:    clientSet,
		Apply:        applier,
		starters: []start.Starter{
			core,
			apps,
			pp,
		},
	}, nil
}

func (a *AppContext) Start(ctx context.Context, threadiness int) error {
	return start.All(ctx, threadiness, a.starters...)
}

func Run(ctx context.Context, operatorOpts common.OperatorOptions, cfg *rest.Config) error {
	operatorOpts.Logger.Info("setting up pprof controller...")
	appCtx, err := Setup(cfg)
	if err != nil {
		return err
	}
	monitor.Register(
		ctx,
		operatorOpts,
		appCtx.Core,
		appCtx.ClientSet,
		appCtx.PprofFactory,
		appCtx.Apply)

	collector.Register(
		ctx,
		operatorOpts,
		appCtx.Core,
		appCtx.Apps,
		appCtx.PprofFactory,
		appCtx.Apply,
	)

	operatorOpts.Logger.Info("starting pprof controller")

	return appCtx.Start(ctx, 10)
}
