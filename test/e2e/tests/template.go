package tests

import (
	"github.com/kaasops/envoy-xds-controller/pkg/utils/k8s"
	"github.com/kaasops/envoy-xds-controller/test/utils"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"os"
	"testing"
	"time"
)

func init() {
	E2ETests = append(
		E2ETests,
		HTTPS_Template,
		HTTPS_Template_Merge,
		HTTPS_Template_Replace,
	)
}

var HTTPS_Template = utils.TestCase{
	ShortName:   "HTTPS_Template",
	Description: "Validation of the correct application of the template to the virtual service",
	Manifests:   nil,
	Test: func(t *testing.T, suite *utils.TestSuite) {
		firstSecretPath := "../testdata/certificates/exc-kaasops-io.yaml"
		domain := "exc.kaasops.io"
		vsPath := "../testdata/e2e/virtual-service-from-template.yaml"
		vsName := "virtual-service-from-template"
		validAnswer := `{"answer":"true"}`

		err := utils.CreateSecretInNamespace(
			suite,
			firstSecretPath, suite.Namespace,
		)
		require.NoError(t, err)
		defer func() {
			err := utils.CleanupManifest(suite.Client, firstSecretPath, suite.Namespace)
			require.NoError(t, err)
		}()

		err = utils.ApplyManifest(
			suite.Client,
			vsPath,
			suite.Namespace,
		)
		defer func() {
			err := utils.CleanupManifest(
				suite.Client,
				vsPath,
				suite.Namespace,
			)
			require.NoError(t, err)
		}()
		require.NoError(t, err)

		// TODO: change wait to check status.valid!
		time.Sleep(2 * time.Second)

		envoyWaitConnectToXDS(t)

		require.True(t, routeExistInxDS(t, k8s.ResourceName(suite.Namespace, vsName)))

		answer1 := curl(t, HTTPS_Method, &domain, "/")
		require.Equal(t, validAnswer, answer1)
		cfgDump := getEnvoyConfigDump(t)

		isEqual := func(path string, value string) {
			got := gjson.Get(cfgDump, path).String()
			require.Equal(t, value, got)
		}

		isEqual(
			"configs.0.bootstrap.node.id",
			"test",
		)
		isEqual(
			"configs.0.bootstrap.node.cluster",
			"test",
		)
		isEqual("configs.0.bootstrap.admin.address.socket_address.port_value",
			"19000",
		)
		isEqual(
			"configs.2.dynamic_listeners.0.name",
			"envoy-xds-controller-e2e/https",
		)
		isEqual(
			"configs.2.dynamic_listeners.0.active_state.listener.address.socket_address.port_value",
			"10443",
		)
		isEqual(
			"configs.2.dynamic_listeners.0.active_state.listener.listener_filters.0.name",
			"envoy.filters.listener.tls_inspector",
		)
		isEqual(
			"configs.2.dynamic_listeners.0.active_state.listener.filter_chains.0.filters.0.typed_config.http_filters.0.name",
			"envoy.filters.http.router",
		)
		isEqual(
			"configs.2.dynamic_listeners.0.active_state.listener.filter_chains.0.filters.0.typed_config.stat_prefix",
			"envoy-xds-controller-e2e/virtual-service-from-template",
		)
		isEqual(
			"configs.2.dynamic_listeners.0.active_state.listener.filter_chains.0.filters.0.typed_config.access_log.0.typed_config.@type",
			"type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog",
		)
	},
}

var HTTPS_Template_Merge = utils.TestCase{
	ShortName:   "HTTPS_Template_Merge",
	Description: "Validation of the correct application of the template to the virtual service",
	Manifests:   nil,
	Test: func(t *testing.T, suite *utils.TestSuite) {
		firstSecretPath := "../testdata/certificates/exc-kaasops-io.yaml"
		domain := "exc.kaasops.io"
		vsPath := "../testdata/e2e/virtual-service-from-template-merge.yaml"
		vsName := "virtual-service-from-template-merge"
		validAnswer := `{"answer":"true"}`

		err := utils.CreateSecretInNamespace(
			suite,
			firstSecretPath, suite.Namespace,
		)
		require.NoError(t, err)
		defer func() {
			err := utils.CleanupManifest(suite.Client, firstSecretPath, suite.Namespace)
			require.NoError(t, err)
		}()

		err = utils.ApplyManifest(
			suite.Client,
			vsPath,
			suite.Namespace,
		)
		defer func() {
			err := utils.CleanupManifest(
				suite.Client,
				vsPath,
				suite.Namespace,
			)
			require.NoError(t, err)
		}()
		require.NoError(t, err)

		// TODO: change wait to check status.valid!
		time.Sleep(2 * time.Second)

		envoyWaitConnectToXDS(t)

		require.True(t, routeExistInxDS(t, k8s.ResourceName(suite.Namespace, vsName)))

		answer1 := curl(t, HTTPS_Method, &domain, "/")
		require.Equal(t, validAnswer, answer1)
		cfgDump := getEnvoyConfigDump(t)

		err = os.WriteFile("/tmp/envoy-config-dump-merge.json", []byte(cfgDump), 0644)
		require.NoError(t, err)

		isEqual := func(path string, value string) {
			got := gjson.Get(cfgDump, path).String()
			require.Equal(t, value, got)
		}

		isEqual(
			"configs.0.bootstrap.node.id",
			"test",
		)
		isEqual(
			"configs.0.bootstrap.node.cluster",
			"test",
		)
		isEqual("configs.0.bootstrap.admin.address.socket_address.port_value",
			"19000",
		)
		isEqual(
			"configs.2.dynamic_listeners.0.name",
			"envoy-xds-controller-e2e/https",
		)
		isEqual(
			"configs.2.dynamic_listeners.0.active_state.listener.address.socket_address.port_value",
			"10443",
		)
		isEqual(
			"configs.2.dynamic_listeners.0.active_state.listener.listener_filters.0.name",
			"envoy.filters.listener.tls_inspector",
		)
		isEqual(
			"configs.2.dynamic_listeners.0.active_state.listener.filter_chains.0.filters.0.typed_config.http_filters.0.name",
			"envoy.filters.http.router",
		)
		isEqual(
			"configs.2.dynamic_listeners.0.active_state.listener.filter_chains.0.filters.0.typed_config.stat_prefix",
			"envoy-xds-controller-e2e/virtual-service-from-template-merge",
		)
		isEqual(
			"configs.2.dynamic_listeners.0.active_state.listener.filter_chains.0.filters.0.typed_config.access_log.0.typed_config.@type",
			"type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog",
		)
		isEqual(
			`configs.4.dynamic_route_configs.#(route_config.name="envoy-xds-controller-e2e/virtual-service-from-template-merge").route_config.virtual_hosts.0.routes.#`,
			"2",
		)
		isEqual(
			`configs.4.dynamic_route_configs.#(route_config.name="envoy-xds-controller-e2e/virtual-service-from-template-merge").route_config.virtual_hosts.0.routes.1.match.prefix`,
			"/health",
		)
		isEqual(
			`configs.4.dynamic_route_configs.#(route_config.name="envoy-xds-controller-e2e/virtual-service-from-template-merge").route_config.virtual_hosts.0.routes.1.direct_response.body.inline_string`,
			`{"status":"ok"}`,
		)
	},
}

var HTTPS_Template_Replace = utils.TestCase{
	ShortName:   "HTTPS_Template_Replace",
	Description: "Validation of the correct application of the template to the virtual service",
	Manifests:   nil,
	Test: func(t *testing.T, suite *utils.TestSuite) {
		firstSecretPath := "../testdata/certificates/exc-kaasops-io.yaml"
		domain := "exc.kaasops.io"
		vsPath := "../testdata/e2e/virtual-service-from-template-replace.yaml"
		vsName := "virtual-service-from-template-replace"
		validAnswer := `{"answer":"error"}`

		err := utils.CreateSecretInNamespace(
			suite,
			firstSecretPath, suite.Namespace,
		)
		require.NoError(t, err)
		defer func() {
			err := utils.CleanupManifest(suite.Client, firstSecretPath, suite.Namespace)
			require.NoError(t, err)
		}()

		err = utils.ApplyManifest(
			suite.Client,
			vsPath,
			suite.Namespace,
		)
		defer func() {
			err := utils.CleanupManifest(
				suite.Client,
				vsPath,
				suite.Namespace,
			)
			require.NoError(t, err)
		}()
		require.NoError(t, err)

		// TODO: change wait to check status.valid!
		time.Sleep(2 * time.Second)

		envoyWaitConnectToXDS(t)

		require.True(t, routeExistInxDS(t, k8s.ResourceName(suite.Namespace, vsName)))

		answer1 := curl(t, HTTPS_Method, &domain, "/")
		require.Equal(t, validAnswer, answer1)
		cfgDump := getEnvoyConfigDump(t)

		err = os.WriteFile("/tmp/envoy-config-dump-replace.json", []byte(cfgDump), 0644)
		require.NoError(t, err)

		isEqual := func(path string, value string) {
			got := gjson.Get(cfgDump, path).String()
			require.Equal(t, value, got)
		}

		isEqual(
			"configs.0.bootstrap.node.id",
			"test",
		)
		isEqual(
			"configs.0.bootstrap.node.cluster",
			"test",
		)
		isEqual("configs.0.bootstrap.admin.address.socket_address.port_value",
			"19000",
		)
		isEqual(
			"configs.2.dynamic_listeners.0.name",
			"envoy-xds-controller-e2e/https",
		)
		isEqual(
			"configs.2.dynamic_listeners.0.active_state.listener.address.socket_address.port_value",
			"10443",
		)
		isEqual(
			"configs.2.dynamic_listeners.0.active_state.listener.listener_filters.0.name",
			"envoy.filters.listener.tls_inspector",
		)
		isEqual(
			"configs.2.dynamic_listeners.0.active_state.listener.filter_chains.0.filters.0.typed_config.http_filters.0.name",
			"envoy.filters.http.router",
		)
		isEqual(
			"configs.2.dynamic_listeners.0.active_state.listener.filter_chains.0.filters.0.typed_config.stat_prefix",
			"envoy-xds-controller-e2e/virtual-service-from-template-replace",
		)
		isEqual(
			"configs.2.dynamic_listeners.0.active_state.listener.filter_chains.0.filters.0.typed_config.access_log.0.typed_config.@type",
			"type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog",
		)
		isEqual(
			`configs.4.dynamic_route_configs.#(route_config.name="envoy-xds-controller-e2e/virtual-service-from-template-replace").route_config.virtual_hosts.0.routes.0.match.prefix`,
			"/",
		)
		isEqual(
			`configs.4.dynamic_route_configs.#(route_config.name="envoy-xds-controller-e2e/virtual-service-from-template-replace").route_config.virtual_hosts.0.routes.#`,
			"1",
		)
		isEqual(
			`configs.4.dynamic_route_configs.#(route_config.name="envoy-xds-controller-e2e/virtual-service-from-template-replace").route_config.virtual_hosts.0.routes.0.direct_response.body.inline_string`,
			`{"answer":"error"}`,
		)
	},
}
