package updater

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"

	wrapped "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
)

// DryBuildNode builds the snapshots of all nodes from the current objects without publishing any, and
// returns the one of nodeID: what the node would be served now if it were not paused. The error tells
// why it would not be served: a listener address conflict of this node, or a failure that stops every
// node, in which case there is no snapshot. A node no object targets gets an empty snapshot.
func (c *CacheUpdater) DryBuildNode(ctx context.Context, nodeID string) (cache.ResourceSnapshot, error) {
	c.mx.RLock()
	storeCopy := c.store.Copy()
	c.mx.RUnlock()

	dry := wrapped.NewSnapshotCache()
	err, _, _, _ := buildSnapshots(ctx, dry, storeCopy, nil)
	snapshot, built := dry.Offered(nodeID)
	if !built {
		if err != nil && !onlyListenerConflicts(err) {
			return nil, err
		}
		return &cache.Snapshot{}, nil
	}
	if err == nil || !onlyListenerConflicts(err) {
		return snapshot, err
	}
	// Only listener address conflicts: each held node has its own, the others were published.
	if conflict := conflictOf(err, nodeID); conflict != nil {
		return snapshot, conflict
	}
	return snapshot, nil
}

// conflictOf returns the listener address conflict of nodeID found in err, or nil.
func conflictOf(err error, nodeID string) *wrapped.DuplicateListenerAddress {
	byNode := make(map[string]*wrapped.DuplicateListenerAddress)
	collectConflicts(err, nil, byNode)
	return byNode[nodeID]
}

// PausedNodeIDs returns the nodes paused on the snapshot they are served, sorted. The webhooks accept
// listener address conflicts on them: nothing reaches their Envoys until they are resumed, and resuming
// is refused while their build fails.
func (c *CacheUpdater) PausedNodeIDs() []string {
	paused := c.snapshotCache.PausedNodes()
	nodeIDs := make([]string, 0, len(paused))
	for nodeID := range paused {
		nodeIDs = append(nodeIDs, nodeID)
	}
	sort.Strings(nodeIDs)
	return nodeIDs
}

// maxPauseReason bounds the reason an operator gave, as it is copied into service statuses.
const maxPauseReason = 200

// withPauseNote adds to the status of a service a note per paused node it goes to: the service may be
// valid and still not served there. Who paused the node is left to the API.
func withPauseNote(status VSStatus, nodeIDs []string, paused map[string]wrapped.Pause) VSStatus {
	notes := make([]string, 0, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		pause, ok := paused[nodeID]
		if !ok {
			continue
		}
		notes = append(notes, fmt.Sprintf("updates to node %s are paused since %s (%s): changes are not pushed",
			boundedStatusText(nodeID, maxDisplayedNodeBytes), pause.At.UTC().Format(time.RFC3339),
			boundedStatusText(pause.Reason, maxPauseReason)))
	}
	if len(notes) == 0 {
		return status
	}
	if status.Message != "" {
		notes = append([]string{status.Message}, notes...)
	}
	status.Message = strings.Join(notes, "; ")
	return status
}
