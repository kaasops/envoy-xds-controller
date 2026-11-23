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
// returns the one of nodeID: what the node would be served now if it were not frozen. The error tells
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

// FrozenNodeIDs returns the nodes frozen on the snapshot they are served, sorted. The webhooks accept
// listener address conflicts on them: nothing reaches their Envoys until they are unfrozen, and unfreezing
// is refused while their build fails.
func (c *CacheUpdater) FrozenNodeIDs() []string {
	frozen := c.snapshotCache.FrozenNodes()
	nodeIDs := make([]string, 0, len(frozen))
	for nodeID := range frozen {
		nodeIDs = append(nodeIDs, nodeID)
	}
	sort.Strings(nodeIDs)
	return nodeIDs
}

// maxFreezeReason bounds the reason an operator gave, as it is copied into service statuses.
const maxFreezeReason = 200

// withFreezeNote adds to the status of a service a note per frozen node it goes to: the service may be
// valid and still not served there. Who froze the node is left to the API.
func withFreezeNote(status VSStatus, nodeIDs []string, frozen map[string]wrapped.Freeze) VSStatus {
	notes := make([]string, 0, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		freeze, ok := frozen[nodeID]
		if !ok {
			continue
		}
		notes = append(notes, fmt.Sprintf("node %s is frozen since %s (%s): changes are not published to it",
			boundedStatusText(nodeID, maxDisplayedNodeBytes), freeze.At.UTC().Format(time.RFC3339),
			boundedStatusText(freeze.Reason, maxFreezeReason)))
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
