package common

import (
	"fmt"
	"log/slog"
)

type OperatorOptions struct {
	OperatorName        string
	ControllerNamespace string
	Logger              *slog.Logger
}

func NamespacedConfigName(opts OperatorOptions) string {
	return fmt.Sprintf("%s-config", opts.OperatorName)
}

func NamespacedCollectorName(opts OperatorOptions) string {
	return fmt.Sprintf("%s-collector", opts.OperatorName)
}
