package v1alpha1

import (
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	ctrmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	vsValidateDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "exc",
			Subsystem: "webhook",
			Name:      "virtualservice_validate_duration_seconds",
			Help:      "Duration of VirtualService validation, seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"mode", "outcome"}, // mode: skipped|light|heavy|heavy_fallback; outcome: ok|error|timeout|coverage_miss
	)

	vsValidateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "exc",
			Subsystem: "webhook",
			Name:      "virtualservice_validate_total",
			Help:      "Total VirtualService validations.",
		},
		[]string{"mode", "outcome"},
	)

	vsLightFallbacks = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "exc",
			Subsystem: "webhook",
			Name:      "virtualservice_light_fallback_total",
			Help:      "Number of light validations that fell back to heavy due to insufficient coverage.",
		},
	)

	featureFlagsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "exc",
			Subsystem: "webhook",
			Name:      "feature_flags",
			Help:      "Feature flags state for webhook (0/1).",
		},
		[]string{"flag"},
	)
)

func init() {
	ctrmetrics.Registry.MustRegister(vsValidateDuration, vsValidateTotal, vsLightFallbacks, featureFlagsGauge)
	// Set gauges once at process start (env vars are static for pod lifetime)
	setFeatureFlagGauge("light_dryrun_enabled", envBool("EXC_WEBHOOK_LIGHT_DRYRUN"))
	setFeatureFlagGauge("validation_indices_enabled", envBool("EXC_VALIDATION_INDICES"))
}

func envBool(name string) bool {
	v := strings.ToLower(os.Getenv(name))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func setFeatureFlagGauge(flag string, on bool) {
	if on {
		featureFlagsGauge.WithLabelValues(flag).Set(1)
	} else {
		featureFlagsGauge.WithLabelValues(flag).Set(0)
	}
}

func observeVSValidation(mode, outcome string, since time.Time) {
	d := time.Since(since)
	vsValidateDuration.WithLabelValues(mode, outcome).Observe(d.Seconds())
	vsValidateTotal.WithLabelValues(mode, outcome).Inc()
}

func incVSValidationFallback() {
	vsLightFallbacks.Inc()
}
