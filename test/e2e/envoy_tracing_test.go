package e2e

import (
	"github.com/kaasops/envoy-xds-controller/test/e2e/fixtures"
	. "github.com/onsi/ginkgo/v2"
)

// tracingEnvoyContext contains e2e tests for Tracing (inline and by reference)
func tracingEnvoyContext() {
	var fixture *fixtures.EnvoyFixture

	BeforeEach(func() {
		fixture = fixtures.NewEnvoyFixture()
		fixture.Setup()
		DeferCleanup(fixture.Teardown)
	})

	It("should apply inline OTLP tracing and expose it in HCM", func() {
		// Prepare: HTTP listener + tracing cluster + VS with inline tracing
		fixture.ApplyManifests(
			"test/testdata/e2e/http_service/http-listener.yaml",
			"test/testdata/e2e/tracing/otel-cluster.yaml",
			"test/testdata/e2e/tracing/vs-inline-otel.yaml",
		)

		fixture.WaitEnvoyConfigChanged()

		// Verify that tracing provider name is set to opentelemetry in HCM
		expectations := map[string]string{
			// nolint: lll
			"configs.2.dynamic_listeners.0.active_state.listener.filter_chains.0.filters.0.typed_config.tracing.provider.name": "envoy.tracers.opentelemetry",
		}
		fixture.VerifyEnvoyConfig(expectations)
	})

	It("should apply tracingRef (OTLP) and expose it in HCM", func() {
		// Prepare: HTTP listener + tracing cluster + Tracing CR + VS with tracingRef
		fixture.ApplyManifests(
			"test/testdata/e2e/http_service/http-listener.yaml",
			"test/testdata/e2e/tracing/otel-cluster.yaml",
			"test/testdata/e2e/tracing/tracing-otlp.yaml",
			"test/testdata/e2e/tracing/vs-ref-otel.yaml",
		)

		fixture.WaitEnvoyConfigChanged()

		expectations := map[string]string{
			// nolint: lll
			"configs.2.dynamic_listeners.0.active_state.listener.filter_chains.0.filters.0.typed_config.tracing.provider.name": "envoy.tracers.opentelemetry",
		}
		fixture.VerifyEnvoyConfig(expectations)
	})

	It("should reject VirtualService with inline OTLP tracing referencing missing cluster", func() {
		// Prepare: only HTTP listener, the VS refers to missing cluster 'missing-otel'
		// Expect webhook validation error bubbled up from dry snapshot build
		fixture.ApplyManifests("test/testdata/e2e/http_service/http-listener.yaml")
		fixture.ApplyManifestsWithError("cluster missing-otel not found", "test/testdata/e2e/tracing/vs-inline-bad-otel.yaml")
	})
}
