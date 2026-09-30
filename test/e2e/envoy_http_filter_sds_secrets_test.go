package e2e

import (
	"fmt"
	"net/http"
	"time"

	"github.com/kaasops/envoy-xds-controller/test/e2e/fixtures"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"
)

// httpFilterSDSSecretsContext contains tests for secrets that HTTP filters request over SDS.
//
// The OAuth2 filter references its token and hmac secrets as namespace/name/key.
// Envoy keeps such a secret in the warming state until the controller delivers it.
func httpFilterSDSSecretsContext() {
	var fixture *fixtures.EnvoyFixture

	const (
		listenerPath       = "test/testdata/e2e/http_filter_sds_secrets/listener.yaml"
		clusterPath        = "test/testdata/e2e/http_filter_sds_secrets/cluster.yaml"
		secretPath         = "test/testdata/e2e/http_filter_sds_secrets/secret.yaml"
		virtualServicePath = "test/testdata/e2e/http_filter_sds_secrets/virtual-service.yaml"
		requestURL         = "http://oauth2.kaasops.io:8082/"
	)

	BeforeEach(func() {
		By("setting up EnvoyFixture")
		fixture = fixtures.NewEnvoyFixture()
		fixture.Setup()
		DeferCleanup(fixture.Teardown)
	})

	It("should deliver the secrets requested by the OAuth2 filter", func() {
		By("applying listener, cluster and secret")
		fixture.ApplyManifests(listenerPath, clusterPath, secretPath)

		// Give the controller time to index the secret
		time.Sleep(2 * time.Second)

		By("applying VirtualService with the OAuth2 filter")
		fixture.ApplyManifests(virtualServicePath)

		By("waiting for the listener to become active")
		listenerJsonPath := "configs.2.dynamic_listeners." +
			"#(name==\"default/http-oauth2\").active_state.listener"
		fixture.WaitEnvoyConfigMatches(map[string]string{
			listenerJsonPath + ".name": "default/http-oauth2",
			listenerJsonPath + ".filter_chains.0.filters.0.typed_config.http_filters.0.name": "envoy.filters.http.oauth2",
		})

		// Recorded before the secrets check: without the secrets that check fails,
		// and the log still has to show what Envoy answers in that state
		By("sending an unauthenticated request before the secrets check")
		status, headers, body := fixture.FetchResponseFromEnvoy(requestURL)
		By(fmt.Sprintf("response before the secrets check: status=%d location=%q body=%q",
			status, headers.Get("Location"), body))

		By("verifying the secrets are active in Envoy")
		secretNames := []string{
			"default/oauth2-credentials/token",
			"default/oauth2-credentials/hmac",
		}
		Eventually(func(g Gomega) {
			active := string(fixture.GetEnvoyConfigDump("resource=dynamic_active_secrets"))
			warming := string(fixture.GetEnvoyConfigDump("resource=dynamic_warming_secrets"))
			for _, name := range secretNames {
				path := fmt.Sprintf("configs.#(name==%q).name", name)
				g.Expect(gjson.Get(active, path).String()).To(Equal(name), "secret should be active: "+name)
				g.Expect(gjson.Get(warming, path).Exists()).To(BeFalse(), "secret should not be warming: "+name)
			}
		}, fixtures.ShortTimeout, fixtures.DefaultPollingInterval).Should(Succeed())

		By("verifying an unauthenticated request is redirected to the authorization endpoint")
		status, headers, body = fixture.FetchResponseFromEnvoy(requestURL)
		By(fmt.Sprintf("response after the secrets check: status=%d location=%q body=%q",
			status, headers.Get("Location"), body))
		Expect(status).To(Equal(http.StatusFound))
		Expect(headers.Get("Location")).To(HavePrefix("http://oauth2.kaasops.io/auth"))
	})
}
