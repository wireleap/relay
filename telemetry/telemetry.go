package telemetry

import (
	"github.com/cabify/gotoprom"

	"os"
)

const defaultNamespace = "wl_relay"

func setupPrometheusMetrics(metrics interface{}, namespace string) {
	gotoprom.MustAddBuilder(TimeHistogramType, registerTimeHistogram)
	gotoprom.MustInit(metrics, namespace)
}

func getNamespace(defaultNS string) (ns string) {
	if ns = os.Getenv("TELEMETRY_NAMESPACE"); ns == "" {
		ns = defaultNS
	}
	return
}

func init() {
	setupPrometheusMetrics(
		&Metrics,
		getNamespace(defaultNamespace))
}
