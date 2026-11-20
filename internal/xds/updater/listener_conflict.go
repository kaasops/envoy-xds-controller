package updater

import (
	"sort"

	wrapped "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
)

// ToleratedListenerConflicts returns the listener address conflicts held in err when every error in
// it is such a conflict on one of nodeIDs, one conflict per node, sorted by node. It returns nil when
// err is nil, when nodeIDs is empty or when err holds any other error, so callers keep rejecting those.
//
// A node with such a conflict keeps its previous snapshot: the snapshot cache refuses a snapshot with
// two listeners on one address, and the other nodes are still published.
func ToleratedListenerConflicts(err error, nodeIDs []string) []*wrapped.DuplicateListenerAddress {
	if err == nil || len(nodeIDs) == 0 {
		return nil
	}
	tolerated := make(map[string]struct{}, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		tolerated[nodeID] = struct{}{}
	}
	byNode := make(map[string]*wrapped.DuplicateListenerAddress)
	if !collectConflicts(err, tolerated, byNode) {
		return nil
	}
	conflicts := make([]*wrapped.DuplicateListenerAddress, 0, len(byNode))
	for _, conflict := range byNode {
		conflicts = append(conflicts, conflict)
	}
	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].NodeID() < conflicts[j].NodeID() })
	return conflicts
}

// onlyListenerConflicts reports whether err holds nothing but listener address conflicts, on any node.
// The nodes with such a conflict keep their previous snapshots; everything else was published.
func onlyListenerConflicts(err error) bool {
	return err != nil && collectConflicts(err, nil, make(map[string]*wrapped.DuplicateListenerAddress))
}

// collectConflicts walks the whole error tree: wrappers are looked through, and every error at the
// bottom has to be a conflict on a tolerated node, or on any node when tolerated is nil. errors.As
// would stop at the first match.
func collectConflicts(
	err error,
	tolerated map[string]struct{},
	byNode map[string]*wrapped.DuplicateListenerAddress,
) bool {
	if conflict, ok := err.(*wrapped.DuplicateListenerAddress); ok {
		if _, ok := tolerated[conflict.NodeID()]; tolerated != nil && !ok {
			return false
		}
		if _, seen := byNode[conflict.NodeID()]; !seen {
			byNode[conflict.NodeID()] = conflict
		}
		return true
	}
	switch wrapper := err.(type) {
	case interface{ Unwrap() []error }:
		inner := wrapper.Unwrap()
		if len(inner) == 0 {
			return false
		}
		for _, e := range inner {
			if e == nil || !collectConflicts(e, tolerated, byNode) {
				return false
			}
		}
		return true
	case interface{ Unwrap() error }:
		inner := wrapper.Unwrap()
		return inner != nil && collectConflicts(inner, tolerated, byNode)
	}
	return false
}
