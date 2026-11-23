package v1alpha1

import (
	"fmt"
	"slices"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	envoyv1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/xds/updater"
)

// toleratedListenerConflict returns one warning per node when every error in err is a listener address
// conflict on a node from nodeIDs, and nil otherwise. Such a node gets no new configuration until the
// conflict is gone; the other nodes are published as usual. Listener names are shown only when both
// are in namespace, the one of the object being checked.
func toleratedListenerConflict(err error, nodeIDs []string, namespace string) admission.Warnings {
	conflicts := updater.ToleratedListenerConflicts(err, nodeIDs)
	if conflicts == nil {
		return nil
	}
	warnings := make(admission.Warnings, 0, len(conflicts))
	for _, conflict := range conflicts {
		detail := "two of its listeners share an address"
		if conflict.ListenersIn(namespace) {
			listener, previous := conflict.Listeners()
			detail = fmt.Sprintf("listeners %s and %s share %s", previous, listener, conflict.Address())
		}
		warning := fmt.Sprintf("node %q gets no new configuration while %s", conflict.NodeID(), detail)
		if len(warning) > maxWarningLength {
			warning = warning[:maxWarningLength-3] + "..."
		}
		warnings = append(warnings, warning)
	}
	return warnings
}

// maxWarningLength keeps a warning whole: the API server may cut longer ones.
const maxWarningLength = 250

// targetsNodes reports whether vs is published to one of nodeIDs. A common service goes to every node.
func targetsNodes(vs *envoyv1alpha1.VirtualService, nodeIDs []string) bool {
	if len(nodeIDs) == 0 {
		return false
	}
	for _, nodeID := range vs.GetNodeIDs() {
		if nodeID == "*" || slices.Contains(nodeIDs, nodeID) {
			return true
		}
	}
	return false
}
