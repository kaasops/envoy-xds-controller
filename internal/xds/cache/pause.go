package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

// maxHistory is the most previously served snapshots a node keeps for a rollback.
const maxHistory = 10

// SetHistoryLimit sets how many previously served snapshots each node keeps, at most maxHistory.
// The default is none: the history costs memory and serves only a rollback.
func (c *SnapshotCache) SetHistoryLimit(limit int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.historyLimit = min(max(limit, 0), maxHistory)
}

var (
	// ErrUnknownSnapshot is returned when a rollback names a snapshot the node has not served.
	ErrUnknownSnapshot = errors.New("snapshot is not in the history of the node")
	// ErrServedChanged is returned when the node is served something else than the caller expects.
	ErrServedChanged = errors.New("the node is served another snapshot now")
)

// Pause records who paused a node and why.
type Pause struct {
	By     string    `json:"by"`
	At     time.Time `json:"at"`
	Reason string    `json:"reason"`
}

// Published is a snapshot as it was served to a node.
type Published struct {
	Snapshot    cache.ResourceSnapshot
	PublishedAt time.Time
}

// ID identifies the content of the snapshot, see SnapshotID.
func (p Published) ID() string { return SnapshotID(p.Snapshot) }

// SnapshotID identifies the content of a snapshot: the first 12 hex digits of a sha256 over the type,
// name and content hash of every resource. Versions do not count, so the same content built twice, or
// served again after a rollback, keeps its ID.
func SnapshotID(snapshot cache.ResourceSnapshot) string {
	h := sha256.New()
	for i := types.ResponseType(0); i < types.UnknownType; i++ {
		typeURL, err := cache.GetResponseTypeURL(i)
		if err != nil {
			continue
		}
		resources := snapshot.GetResources(typeURL)
		names := make([]string, 0, len(resources))
		for name := range resources {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			_, _ = fmt.Fprintf(h, "%s\x00%s\x00%s\n", typeURL, name, computeResourceHash(resources[name]))
		}
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
}

// Pause stops publishing new snapshots to the node: builds go on and the last one is kept as offered,
// but the node is served what it has. It returns false when the node is paused already.
func (c *SnapshotCache) Pause(nodeID string, pause Pause) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, paused := c.paused[nodeID]; paused {
		return false
	}
	c.paused[nodeID] = pause
	return true
}

// Resume lets the next build publish to the node again. It returns false when the node is not paused.
func (c *SnapshotCache) Resume(nodeID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, paused := c.paused[nodeID]; !paused {
		return false
	}
	delete(c.paused, nodeID)
	return true
}

// Paused reports whether the node is paused, and by whom.
func (c *SnapshotCache) Paused(nodeID string) (Pause, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	pause, paused := c.paused[nodeID]
	return pause, paused
}

// PausedNodes returns the paused nodes.
func (c *SnapshotCache) PausedNodes() map[string]Pause {
	c.mu.RLock()
	defer c.mu.RUnlock()
	nodes := make(map[string]Pause, len(c.paused))
	for nodeID, pause := range c.paused {
		nodes[nodeID] = pause
	}
	return nodes
}

// RestorePaused replaces the paused nodes, for a controller that starts with nothing served: a paused
// node gets no snapshot until it is resumed, its running Envoys keep the configuration they have.
func (c *SnapshotCache) RestorePaused(nodes map[string]Pause) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paused = make(map[string]Pause, len(nodes))
	for nodeID, pause := range nodes {
		c.paused[nodeID] = pause
	}
}

// Served returns the snapshot the node is served.
func (c *SnapshotCache) Served(nodeID string) (Published, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	published, ok := c.served[nodeID]
	return published, ok
}

// History returns the snapshots the node was served before the current one, newest first.
func (c *SnapshotCache) History(nodeID string) []Published {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]Published(nil), c.history[nodeID]...)
}

// Offered returns the last snapshot a build gave for the node, served or not.
func (c *SnapshotCache) Offered(nodeID string) (cache.ResourceSnapshot, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	snapshot, ok := c.offered[nodeID]
	return snapshot, ok
}

// Rollback serves the node a snapshot from its history and pauses the node, so the next build does
// not replace it. The content goes out under version, see NextVersion: an Envoy that reconnects with a
// version it got before must not be taken as up to date. With expectServed, the rollback happens only
// while the node is served the snapshot of that ID.
func (c *SnapshotCache) Rollback(
	ctx context.Context,
	nodeID, id, expectServed string,
	pause Pause,
	version int64,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if expectServed != "" {
		if served, ok := c.served[nodeID]; !ok || served.ID() != expectServed {
			return ErrServedChanged
		}
	}
	var target cache.ResourceSnapshot
	for _, published := range c.history[nodeID] {
		if published.ID() == id {
			target = published.Snapshot
			break
		}
	}
	if target == nil {
		return fmt.Errorf("%w: %s", ErrUnknownSnapshot, id)
	}
	snapshot, err := withVersion(target, version)
	if err != nil {
		return err
	}
	if err := c.publish(ctx, nodeID, snapshot); err != nil {
		return err
	}
	c.versionFloor = max(c.versionFloor, version)
	c.paused[nodeID] = pause
	return nil
}

// PublishPaused serves a paused node the given snapshot, the one an operator compared, and resumes it:
// the next builds publish to it as usual. The snapshot goes out under version, see NextVersion; when its
// content is what the node is served already, nothing is sent.
func (c *SnapshotCache) PublishPaused(
	ctx context.Context,
	nodeID string,
	snapshot cache.ResourceSnapshot,
	version int64,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, paused := c.paused[nodeID]; !paused {
		return fmt.Errorf("node %s is not paused", nodeID)
	}
	if served, ok := c.served[nodeID]; !ok || served.ID() != SnapshotID(snapshot) {
		if err := c.validateCache(nodeID, snapshot); err != nil {
			return fmt.Errorf("snapshot is invalid: %w", err)
		}
		next, err := withVersion(snapshot, version)
		if err != nil {
			return err
		}
		if err := c.publish(ctx, nodeID, next); err != nil {
			return err
		}
		c.versionFloor = max(c.versionFloor, version)
	}
	delete(c.paused, nodeID)
	return nil
}

// publish serves the snapshot to the node and moves the one served before into the history.
// The caller holds c.mu.
func (c *SnapshotCache) publish(ctx context.Context, nodeID string, snapshot cache.ResourceSnapshot) error {
	if err := c.SnapshotCache.SetSnapshot(ctx, nodeID, snapshot); err != nil {
		return err
	}
	now := time.Now().UTC()
	// The same content again, an empty snapshot for a node without services on every build for one,
	// is not a new entry: it would push out the configurations a rollback needs.
	if previous, ok := c.served[nodeID]; ok && c.historyLimit > 0 && previous.ID() != SnapshotID(snapshot) {
		history := append([]Published{previous}, c.history[nodeID]...)
		if len(history) > c.historyLimit {
			history = history[:c.historyLimit]
		}
		c.history[nodeID] = history
	}
	c.served[nodeID] = Published{Snapshot: snapshot, PublishedAt: now}
	c.nodeIDs[nodeID] = now
	return nil
}

// SetVersionFloor sets the highest version a pause or rollback has served, as kept across restarts.
func (c *SnapshotCache) SetVersionFloor(version int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.versionFloor = max(c.versionFloor, version)
}

// RaiseVersionFloor raises the floor to the highest version the node is served and returns the floor.
// Called on a pause: builds go on counting from what the node was served, and nothing is served to a
// paused node until a pause or rollback publication, so the kept floor covers every version its Envoys
// may hold, whatever the clock does.
func (c *SnapshotCache) RaiseVersionFloor(nodeID string) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if served, ok := c.served[nodeID]; ok {
		for i := types.ResponseType(0); i < types.UnknownType; i++ {
			typeURL, err := cache.GetResponseTypeURL(i)
			if err != nil {
				continue
			}
			if version, err := strconv.ParseInt(served.Snapshot.GetVersion(typeURL), 10, 64); err == nil {
				c.versionFloor = max(c.versionFloor, version)
			}
		}
	}
	return c.versionFloor
}

// VersionFloor returns the highest version a pause or rollback has served.
func (c *SnapshotCache) VersionFloor() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.versionFloor
}

// NextVersion returns the version for the next pause or rollback publication on the node: above every
// version the node was served or offered in this process, above the floor, and not below the clock in
// nanoseconds. An SotW stream that holds an equal version string would not get the update. Build
// counters start again after a controller restart; builds after a pause count on from its version, one
// per build, while the clock gains a billion per second of the restart. The caller keeps the version
// before serving it and gives it back with SetVersionFloor after a restart, so only a wall clock that
// steps back further than the restart took could repeat a version.
func (c *SnapshotCache) NextVersion(nodeID string) int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	next := max(c.now().UnixNano(), c.versionFloor+1)
	for _, seen := range []cache.ResourceSnapshot{c.served[nodeID].Snapshot, c.offered[nodeID]} {
		if seen == nil {
			continue
		}
		for i := types.ResponseType(0); i < types.UnknownType; i++ {
			typeURL, err := cache.GetResponseTypeURL(i)
			if err != nil {
				continue
			}
			if version, err := strconv.ParseInt(seen.GetVersion(typeURL), 10, 64); err == nil && version >= next {
				next = version + 1
			}
		}
	}
	return next
}

// withVersion returns the resources of snapshot under one version for every type.
func withVersion(snapshot cache.ResourceSnapshot, version int64) (*cache.Snapshot, error) {
	source, ok := snapshot.(*cache.Snapshot)
	if !ok {
		return nil, fmt.Errorf("unsupported snapshot type %T", snapshot)
	}
	result := &cache.Snapshot{}
	for i := range source.Resources {
		result.Resources[i] = cache.Resources{
			Version: strconv.FormatInt(version, 10),
			Items:   source.Resources[i].Items,
		}
	}
	return result, nil
}
