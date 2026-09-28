package e2e

import (
	"strings"

	"github.com/kaasops/envoy-xds-controller/test/e2e/fixtures"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const dplTestdata = "test/testdata/e2e/domain_per_listener/"

// domainPerListenerContext covers domain uniqueness being scoped to a listener rather
// than to the whole node. Envoy resolves a connection to a listener first and only then
// picks a filter chain inside it, so the same domain on two listeners is legitimate -
// while a duplicate within one listener still is not.
//
// Every VirtualService here gets its own listener on purpose: on a non-TLS listener the
// filter chain carries no match criteria, so two VirtualServices on one such listener
// collide in Envoy regardless of their domains.
func domainPerListenerContext() {
	var fixture *fixtures.EnvoyFixture

	BeforeEach(func() {
		fixture = fixtures.NewEnvoyFixture()
		fixture.Setup()
		DeferCleanup(fixture.Teardown)
	})

	It("should serve the same domain from two listeners on different ports", func() {
		By("applying two listeners and two virtual services sharing one domain")
		fixture.ApplyManifests(
			dplTestdata+"listener-dpl-a.yaml",
			dplTestdata+"listener-dpl-b.yaml",
			dplTestdata+"vs-a.yaml",
			dplTestdata+"vs-b.yaml",
		)
		fixture.WaitEnvoyConfigChanged()

		By("verifying each port answers with its own virtual service")
		// FetchDataFromEnvoy sends the bare domain in the Host header. That matters:
		// on a non-standard port a client would otherwise send "domain:port", which
		// does not match a virtual host declared without the port.
		Expect(strings.TrimSpace(fixture.FetchDataFromEnvoy("http://dup.kaasops.io:10081/"))).
			To(Equal("from-listener-a"))
		Expect(strings.TrimSpace(fixture.FetchDataFromEnvoy("http://dup.kaasops.io:10082/"))).
			To(Equal("from-listener-b"))
	})

	It("should serve a wildcard domain from two listeners on different ports", func() {
		By("applying two listeners and two virtual services both using '*'")
		fixture.ApplyManifests(
			dplTestdata+"listener-dpl-wc-a.yaml",
			dplTestdata+"listener-dpl-wc-b.yaml",
			dplTestdata+"vs-wc-a.yaml",
			dplTestdata+"vs-wc-b.yaml",
		)
		fixture.WaitEnvoyConfigChanged()

		By("verifying each port answers with its own catch-all virtual service")
		Expect(strings.TrimSpace(fixture.FetchDataFromEnvoy("http://anything.kaasops.io:10083/"))).
			To(Equal("wildcard-a"))
		Expect(strings.TrimSpace(fixture.FetchDataFromEnvoy("http://anything.kaasops.io:10084/"))).
			To(Equal("wildcard-b"))
	})

	It("should still reject the same domain twice on one listener", func() {
		By("applying a listener and a virtual service that claims the domain")
		fixture.ApplyManifests(
			dplTestdata+"listener-dpl-a.yaml",
			dplTestdata+"vs-a.yaml",
		)
		fixture.WaitEnvoyConfigChanged()

		By("expecting the webhook to reject a second virtual service with the same domain")
		// The quoted form is what both the heavy and the light validation path report,
		// so this holds regardless of how the webhook is configured.
		fixture.ApplyManifestsWithError(
			"duplicate domain 'dup.kaasops.io'",
			dplTestdata+"vs-dup.yaml",
		)
	})
}
