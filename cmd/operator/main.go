package main

import (
	"context"
	"log/slog"

	"github.com/rancher-sandbox/profiling/pkg/controllers/common"
	"github.com/rancher-sandbox/profiling/pkg/operator"
	"github.com/rancher-sandbox/profiling/pkg/operator/apis/v1alpha1"
	"github.com/rancher/wrangler/v3/pkg/crd"
	"github.com/rancher/wrangler/v3/pkg/kubeconfig"
	"github.com/rancher/wrangler/v3/pkg/signals"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
)

var (
	logger = slog.Default()
)

func BuildOperatorCmd() *cobra.Command {
	var kubeConfigPath string
	var controllerNamespace string
	var operatorName string
	cmd := &cobra.Command{
		Use: "poperator",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := signals.SetupSignalContext()
			restKubeConfig, err := kubeconfig.GetNonInteractiveClientConfig(kubeConfigPath).ClientConfig()
			if err != nil {
				return err
			}

			requiredCrds := []crd.CRD{
				crd.NamespacedType("PprofMonitor.resources.cattle.io/v1alpha1").WithSchemaFromStruct(&v1alpha1.PprofMonitor{}),
				crd.NamespacedType("PprofCollectorStack.resources.cattle.io/v1alpha1").WithSchemaFromStruct(&v1alpha1.PprofCollectorStack{}),
			}

			if err := createCrd(context.Background(), restKubeConfig, requiredCrds); err != nil {
				panic(err)
			}

			if err := operator.Run(ctx, common.OperatorOptions{
				OperatorName:        operatorName,
				ControllerNamespace: controllerNamespace,
				Logger:              logger,
			}, restKubeConfig); err != nil {
				return err
			}

			<-ctx.Done()
			return nil
		},
	}
	cmd.Flags().StringVarP(&kubeConfigPath, "kubeconfig", "", "", "Path to kubeconfig. Only required if running out of cluster")
	cmd.Flags().StringVarP(&controllerNamespace, "namespace", "n", "pprof-controller", "Namespace to watch for pprof monitors")
	cmd.Flags().StringVarP(&operatorName, "name", "", "pprof-operator", "Name of the operator")
	return cmd
}

func main() {
	cmd := BuildOperatorCmd()
	if err := cmd.Execute(); err != nil {
		logger.With("err", err).Error("failed to run pprof operator")
	}
}

func createCrd(ctx context.Context, cfg *rest.Config, crds []crd.CRD) error {
	factory, err := crd.NewFactoryFromClient(cfg)
	if err != nil {
		return err
	}
	return factory.BatchCreateCRDs(ctx, crds...).BatchWait()
}
