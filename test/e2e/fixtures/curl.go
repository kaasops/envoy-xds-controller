package fixtures

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/kaasops/envoy-xds-controller/test/utils"
	. "github.com/onsi/gomega"
)

// Pod logs combine stdout and stderr. Keep partial responses and retry errors out of a successful result.
const curlGet = `status=$(curl -fsS --retry 5 --retry-all-errors --retry-delay 1 --retry-max-time 30 --max-time 10 \
    -o /tmp/response -w '%{http_code}' "$@" 2>/tmp/curl-error) || {
    code=$?; cat /tmp/curl-error >&2; exit "$code"
}
[ "$status" = 200 ] || { echo "Unexpected HTTP status: $status" >&2; exit 1; }
cat /tmp/response`

func curlFromPod(name, namespace string, timeout time.Duration, args ...string) string {
	run := []string{"run", name, "-n", namespace, "--restart=Never", "--image=curlimages/curl:7.78.0"}
	_, err := utils.Run(exec.Command("kubectl", append(run, args...)...))
	Expect(err).NotTo(HaveOccurred(), "Failed to create curl pod %s", name)

	Eventually(func() (string, error) {
		cmd := exec.Command("kubectl", "get", "pod", name, "-n", namespace, "-o", "jsonpath={.status.phase}")
		phase, err := utils.Run(cmd)
		if err == nil && phase == "Failed" {
			logs, logErr := utils.Run(exec.Command("kubectl", "logs", name, "-n", namespace))
			return phase, StopTrying(fmt.Sprintf("Curl pod %s failed: %s", name, logs)).Wrap(logErr)
		}
		return phase, err
	}, timeout, 2*time.Second).Should(Equal("Succeeded"), "Waiting for curl pod %s", name)

	data, err := utils.Run(exec.Command("kubectl", "logs", name, "-n", namespace))
	Expect(err).NotTo(HaveOccurred(), "Failed to read curl pod %s", name)
	return data
}
