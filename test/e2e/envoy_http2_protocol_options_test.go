package e2e

import (
	"github.com/kaasops/envoy-xds-controller/test/e2e/fixtures"
	. "github.com/onsi/ginkgo/v2"
)

// http2ProtocolOptionsEnvoyContext contains e2e tests for HTTP/2 protocol options
func http2ProtocolOptionsEnvoyContext() {
	var fixture *fixtures.EnvoyFixture

	BeforeEach(func() {
		fixture = fixtures.NewEnvoyFixture()
		fixture.Setup()
		DeferCleanup(fixture.Teardown)
	})

	It("should apply default HTTP/2 protocol options for TLS listeners", func() {
		// Prepare: HTTPS listener + TLS cert + VS without http2ProtocolOptions
		fixture.ApplyManifests(
			"test/testdata/e2e/basic_https_service/listener.yaml",
			"test/testdata/e2e/basic_https_service/tls-cert.yaml",
			"test/testdata/e2e/http2_protocol_options/vs-default-http2.yaml",
		)

		// Verify default HTTP/2 protocol options are applied
		// nolint: lll
		listenerPath := "configs.2.dynamic_listeners.#(name==\"default/https\").active_state.listener"
		hcmPath := listenerPath + ".filter_chains.0.filters.0.typed_config"

		expectations := map[string]string{
			hcmPath + ".http2_protocol_options.max_concurrent_streams":         "100",
			hcmPath + ".http2_protocol_options.initial_stream_window_size":     "65536",
			hcmPath + ".http2_protocol_options.initial_connection_window_size": "1048576",
		}
		fixture.WaitEnvoyConfigMatches(expectations)
	})

	It("should apply custom HTTP/2 protocol options when specified", func() {
		// Prepare: HTTPS listener + TLS cert + VS with custom http2ProtocolOptions
		fixture.ApplyManifests(
			"test/testdata/e2e/basic_https_service/listener.yaml",
			"test/testdata/e2e/basic_https_service/tls-cert.yaml",
			"test/testdata/e2e/http2_protocol_options/vs-custom-http2.yaml",
		)

		// Verify custom HTTP/2 protocol options are applied
		// nolint: lll
		listenerPath := "configs.2.dynamic_listeners.#(name==\"default/https\").active_state.listener"
		hcmPath := listenerPath + ".filter_chains.0.filters.0.typed_config"

		expectations := map[string]string{
			hcmPath + ".http2_protocol_options.max_concurrent_streams":         "50",
			hcmPath + ".http2_protocol_options.initial_stream_window_size":     "131072",
			hcmPath + ".http2_protocol_options.initial_connection_window_size": "2097152",
		}
		fixture.WaitEnvoyConfigMatches(expectations)
	})

	It("should NOT apply HTTP/2 protocol options for non-TLS listeners", func() {
		// Prepare: HTTP listener (no TLS) + VS without http2ProtocolOptions
		fixture.ApplyManifests(
			"test/testdata/e2e/http_service/http-listener.yaml",
			"test/testdata/e2e/http2_protocol_options/vs-http-no-defaults.yaml",
		)

		// Verify HTTP/2 protocol options are NOT applied (empty/null)
		// nolint: lll
		listenerPath := "configs.2.dynamic_listeners.#(name==\"default/http\").active_state.listener"
		hcmPath := listenerPath + ".filter_chains.0.filters.0.typed_config"

		// For non-TLS listeners, http2_protocol_options should be empty/null
		expectations := map[string]string{
			hcmPath + ".http2_protocol_options": "",
		}
		fixture.WaitEnvoyConfigMatches(expectations)
	})

	It("should reject invalid HTTP/2 protocol options", func() {
		// Prepare: HTTPS listener + TLS cert
		fixture.ApplyManifests(
			"test/testdata/e2e/basic_https_service/listener.yaml",
			"test/testdata/e2e/basic_https_service/tls-cert.yaml",
		)

		// Apply VS with invalid http2ProtocolOptions (initial_stream_window_size < 65535)
		// Expect webhook to reject it
		fixture.ApplyManifestsWithError(
			"failed to validate http2ProtocolOptions",
			"test/testdata/e2e/http2_protocol_options/vs-invalid-http2.yaml",
		)
	})

	It("should apply partial HTTP/2 protocol options without defaults for unset fields", func() {
		// Prepare: HTTPS listener + TLS cert + VS with only max_concurrent_streams set
		fixture.ApplyManifests(
			"test/testdata/e2e/basic_https_service/listener.yaml",
			"test/testdata/e2e/basic_https_service/tls-cert.yaml",
			"test/testdata/e2e/http2_protocol_options/vs-partial-http2.yaml",
		)

		// Verify only max_concurrent_streams is set, other fields are empty
		// nolint: lll
		listenerPath := "configs.2.dynamic_listeners.#(name==\"default/https\").active_state.listener"
		hcmPath := listenerPath + ".filter_chains.0.filters.0.typed_config"

		expectations := map[string]string{
			hcmPath + ".http2_protocol_options.max_concurrent_streams":         "200",
			hcmPath + ".http2_protocol_options.initial_stream_window_size":     "",
			hcmPath + ".http2_protocol_options.initial_connection_window_size": "",
		}
		fixture.WaitEnvoyConfigMatches(expectations)
	})

	It("should apply custom HTTP/2 protocol options for non-TLS listeners when explicitly set", func() {
		// Prepare: HTTP listener + VS with explicit http2ProtocolOptions
		fixture.ApplyManifests(
			"test/testdata/e2e/http_service/http-listener.yaml",
			"test/testdata/e2e/http2_protocol_options/vs-http-custom-http2.yaml",
		)

		// Verify custom HTTP/2 protocol options are applied even for non-TLS listener
		// nolint: lll
		listenerPath := "configs.2.dynamic_listeners.#(name==\"default/http\").active_state.listener"
		hcmPath := listenerPath + ".filter_chains.0.filters.0.typed_config"

		expectations := map[string]string{
			hcmPath + ".http2_protocol_options.max_concurrent_streams":         "75",
			hcmPath + ".http2_protocol_options.initial_stream_window_size":     "262144",
			hcmPath + ".http2_protocol_options.initial_connection_window_size": "4194304",
		}
		fixture.WaitEnvoyConfigMatches(expectations)
	})
}
