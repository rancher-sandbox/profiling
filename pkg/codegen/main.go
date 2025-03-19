package main

import (
	"os"

	"github.com/rancher-sandbox/profiling/pkg/operator/apis/v1alpha1"
	controllergen "github.com/rancher/wrangler/v3/pkg/controller-gen"
	"github.com/rancher/wrangler/v3/pkg/controller-gen/args"
)

func main() {
	os.Unsetenv("GOPATH")
	controllergen.Run(args.Options{
		OutputPackage: "github.com/rancher-sandbox/profiling/pkg/operator/generated",
		Boilerplate:   "gen/boilerplate.go.txt",
		Groups: map[string]args.Group{
			"resources.cattle.io": {
				Types: []interface{}{
					v1alpha1.PprofMonitor{},
					v1alpha1.PprofCollectorStack{},
				},
				GenerateTypes: true,
			},
		},
	})
}
