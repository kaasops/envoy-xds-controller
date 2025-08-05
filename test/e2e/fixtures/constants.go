package fixtures

import "time"

// Constants for all fixtures
const (
	// Namespace where the project is deployed
	Namespace = "envoy-xds-controller"

	// ServiceAccountName created for the project
	ServiceAccountName = "exc-e2e-envoy-xds-controller"

	// MetricsServiceName is the name of the metrics service
	MetricsServiceName = "exc-e2e-envoy-xds-controller-metrics"

	// MetricsRoleBindingName is the name of the RBAC for metrics access
	MetricsRoleBindingName = "envoy-xds-controller-metrics-binding"

	// Default timeouts
	DefaultTimeout         = 2 * time.Minute
	DefaultPollingInterval = 1 * time.Second
	LongTimeout            = 5 * time.Minute
	ShortTimeout           = 60 * time.Second
)
