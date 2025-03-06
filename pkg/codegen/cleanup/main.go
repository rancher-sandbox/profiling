package main

import (
	"log/slog"
	"os"

	"github.com/rancher/wrangler/v3/pkg/cleanup"
)

const apiPackage = "./pkg/operator/apis"
const generatedPackage = "./pkg/operator/generated"

func main() {
	logger := slog.Default()
	if err := cleanup.Cleanup(apiPackage); err != nil {
		logger.With("err", err, "path", apiPackage).Error("codegen failed to cleanup")
		return
	}

	if err := os.RemoveAll(generatedPackage); err != nil {
		logger.With("err", err, "path", generatedPackage).Error("codegen failed to cleanup generated pkg")
		return
	}
}
