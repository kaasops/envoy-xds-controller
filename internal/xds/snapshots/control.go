// Package snapshots lets an operator see what a node is served and what the controller has built for it,
// freeze the node on what it is served, and roll it back to a snapshot it was served before.
package snapshots

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	wrapped "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
)

var (
	ErrUnknownNode     = errors.New("unknown node")
	ErrUnknownResource = errors.New("unknown resource")
	ErrAlreadyFrozen   = errors.New("node is frozen already")
	ErrNotFrozen       = errors.New("node is not frozen")
	ErrNoReason        = errors.New("a reason is required")
	// ErrStateChanged means the node changed since the operator read it: read again and decide again.
	ErrStateChanged = errors.New("the node changed since it was read")
	// ErrBuildFails means what is built for the node now could not be published.
	ErrBuildFails = errors.New("the build for the node fails")
)

// Snapshot names accepted where a snapshot is chosen; anything else is the ID of a snapshot in the history.
const (
	Served = "served"
	Built  = "built"
)

// Builder is the part of the cache updater the control needs.
type Builder interface {
	// DryBuildNode builds what the node would be served now, without publishing anything.
	DryBuildNode(ctx context.Context, nodeID string) (cache.ResourceSnapshot, error)
	// RebuildSnapshots builds from the current objects and publishes to every node that is not frozen.
	RebuildSnapshots(ctx context.Context) error
}

// FreezeStore keeps the frozen nodes, and the highest version a freeze or rollback has served, across
// controller restarts.
type FreezeStore interface {
	Save(ctx context.Context, frozen map[string]wrapped.Freeze, versionFloor int64) error
}

// Control reads and changes what nodes are served.
type Control struct {
	Cache   *wrapped.SnapshotCache
	Builder Builder
	// Store keeps the frozen nodes; nil keeps them in memory only.
	Store FreezeStore
	Now   func() time.Time

	// writes runs freeze, unfreeze and rollback one at a time, each until the frozen nodes are kept or
	// restored: an operation that fails restores the map it read, which must not undo another one.
	writes sync.Mutex
}

// SnapshotInfo describes one snapshot of a node.
type SnapshotInfo struct {
	ID          string            `json:"id"`
	PublishedAt *time.Time        `json:"publishedAt,omitempty"`
	BuiltAt     *time.Time        `json:"builtAt,omitempty"`
	Versions    map[string]string `json:"versions"`
	Counts      map[string]int    `json:"counts"`
	Error       string            `json:"error,omitempty"`
}

// State is what a node is served, what was built for it and what it was served before.
type State struct {
	NodeID  string          `json:"nodeId"`
	Frozen  bool            `json:"frozen"`
	Freeze  *wrapped.Freeze `json:"freeze"`
	Served  *SnapshotInfo   `json:"served"`
	Built   *SnapshotInfo   `json:"built"`
	History []SnapshotInfo  `json:"history"`
}

// State returns the state of the node.
func (c *Control) State(ctx context.Context, nodeID string) (State, error) {
	if !c.known(nodeID) {
		return State{}, fmt.Errorf("%w: %s", ErrUnknownNode, nodeID)
	}
	state := State{NodeID: nodeID, History: []SnapshotInfo{}}
	if freeze, frozen := c.Cache.Frozen(nodeID); frozen {
		state.Frozen, state.Freeze = true, &freeze
	}
	if served, ok := c.Cache.Served(nodeID); ok {
		info := describe(served.Snapshot, true)
		info.PublishedAt = &served.PublishedAt
		state.Served = &info
	}
	built, err := c.Builder.DryBuildNode(ctx, nodeID)
	if built != nil {
		info := describe(built, false)
		at := c.Now().UTC()
		info.BuiltAt = &at
		state.Built = &info
	} else {
		state.Built = &SnapshotInfo{Versions: map[string]string{}, Counts: map[string]int{}}
	}
	if err != nil {
		state.Built.Error = err.Error()
	}
	for _, published := range c.Cache.History(nodeID) {
		info := describe(published.Snapshot, true)
		info.PublishedAt = &published.PublishedAt
		state.History = append(state.History, info)
	}
	return state, nil
}

// Freeze stops publishing to the node; the node keeps being served what it has. The freeze is kept in
// the store first: a freeze lost on a restart would publish what it held back.
func (c *Control) Freeze(ctx context.Context, nodeID, by, reason string) (State, error) {
	c.writes.Lock()
	defer c.writes.Unlock()
	if reason == "" {
		return State{}, ErrNoReason
	}
	if !c.known(nodeID) {
		return State{}, fmt.Errorf("%w: %s", ErrUnknownNode, nodeID)
	}
	if !c.Cache.Freeze(nodeID, wrapped.Freeze{By: by, At: c.Now().UTC(), Reason: reason}) {
		return State{}, ErrAlreadyFrozen
	}
	// kept with the freeze: the versions the node was served must not come back after a restart
	c.Cache.RaiseVersionFloor(nodeID)
	if err := c.save(ctx); err != nil {
		c.Cache.Unfreeze(nodeID)
		return State{}, err
	}
	// The services of the node get the note that it is frozen; nothing is published to it.
	_ = c.Builder.RebuildSnapshots(ctx)
	return c.State(ctx, nodeID)
}

// Unfreeze serves the node what is built from the current objects, then lets every build publish to it
// again. It refuses while that build fails, and keeps the freeze: the node would only be held for
// another reason. With expectBuilt, the ID of the built snapshot the operator compared, it refuses when
// the build differs now; otherwise exactly that snapshot is served first, then whatever the objects
// build at that moment.
func (c *Control) Unfreeze(ctx context.Context, nodeID, _, expectBuilt string) (State, error) {
	c.writes.Lock()
	defer c.writes.Unlock()
	freeze, frozen := c.Cache.Frozen(nodeID)
	if !frozen {
		return State{}, ErrNotFrozen
	}
	built, err := c.Builder.DryBuildNode(ctx, nodeID)
	if err != nil {
		return State{}, fmt.Errorf("%w: %w", ErrBuildFails, err)
	}
	if expectBuilt != "" && (built == nil || wrapped.SnapshotID(built) != expectBuilt) {
		return State{}, ErrStateChanged
	}
	// The version is kept before it is served: after a restart the next one is higher.
	version := c.Cache.NextVersion(nodeID)
	if c.Store != nil {
		if err := c.Store.Save(ctx, withoutNode(c.Cache.FrozenNodes(), nodeID), version); err != nil {
			return State{}, err
		}
	}
	if err := c.Cache.PublishFrozen(ctx, nodeID, built, version); err != nil {
		if c.Store != nil {
			_ = c.Store.Save(ctx, withNode(c.Cache.FrozenNodes(), nodeID, freeze), version)
		}
		return State{}, err
	}
	// The objects may have changed since the compared build: publish what they build now. A failure
	// shows in the state, the node is held by the builder then.
	_ = c.Builder.RebuildSnapshots(ctx)
	return c.State(ctx, nodeID)
}

// Rollback serves the node a snapshot from its history and freezes it there. With expectServed, the ID
// of the snapshot the operator saw served, it refuses when the node is served something else now.
func (c *Control) Rollback(ctx context.Context, nodeID, by, id, expectServed string) (State, error) {
	c.writes.Lock()
	defer c.writes.Unlock()
	if !c.inHistory(nodeID, id) {
		return State{}, fmt.Errorf("%w: %s", wrapped.ErrUnknownSnapshot, id)
	}
	freeze := wrapped.Freeze{By: by, At: c.Now().UTC(), Reason: "rollback to " + id}
	before := c.Cache.FrozenNodes()
	// The version is kept before it is served: after a restart the next one is higher.
	version := c.Cache.NextVersion(nodeID)
	if c.Store != nil {
		if err := c.Store.Save(ctx, withNode(before, nodeID, freeze), version); err != nil {
			return State{}, err
		}
	}
	if err := c.Cache.Rollback(ctx, nodeID, id, expectServed, freeze, version); err != nil {
		if errors.Is(err, wrapped.ErrServedChanged) {
			err = fmt.Errorf("%w: %w", ErrStateChanged, err)
		}
		if c.Store != nil {
			_ = c.Store.Save(ctx, before, version)
		}
		return State{}, err
	}
	// The services of the node get the note that it is frozen; nothing is published to it.
	_ = c.Builder.RebuildSnapshots(ctx)
	return c.State(ctx, nodeID)
}

// Summary counts the resources of one type by change.
type Summary struct {
	Same        int `json:"same"`
	SameDecoded int `json:"sameDecoded"`
	Changed     int `json:"changed"`
	Added       int `json:"added"`
	Removed     int `json:"removed"`
}

// Change is the difference of one resource between two snapshots.
type Change struct {
	TypeURL    string `json:"typeUrl"`
	Name       string `json:"name"`
	Change     string `json:"change"`
	FromSHA256 string `json:"fromSha256"`
	ToSHA256   string `json:"toSha256"`
}

// Diff compares two snapshots of a node resource by resource.
type Diff struct {
	From      SnapshotInfo       `json:"from"`
	To        SnapshotInfo       `json:"to"`
	Summary   map[string]Summary `json:"summary"`
	Resources []Change           `json:"resources"`
}

// Diff compares two snapshots of the node: served, built or the ID of one from the history. Bytes are
// compared first; resources with different bytes are decoded and compared again, so a different order
// of fields inside packed Any values counts as sameDecoded. Secrets are compared by bytes only.
// Without all, only resources that are not the same are listed.
func (c *Control) Diff(ctx context.Context, nodeID, from, to string, all bool) (Diff, error) {
	a, fromTime, err := c.snapshot(ctx, nodeID, from)
	if err != nil {
		return Diff{}, err
	}
	b, toTime := a, fromTime
	// A served inventory must describe one publication even if a rebuild publishes during this request.
	if from != Served || to != Served {
		b, toTime, err = c.snapshot(ctx, nodeID, to)
		if err != nil {
			return Diff{}, err
		}
	}
	diff := Diff{
		From:      describe(a, from != Built),
		To:        describe(b, to != Built),
		Summary:   map[string]Summary{},
		Resources: []Change{},
	}
	diff.From.PublishedAt, diff.To.PublishedAt = fromTime, toTime
	for _, typeURL := range typeURLs() {
		before, after := a.GetResources(typeURL), b.GetResources(typeURL)
		if len(before) == 0 && len(after) == 0 {
			continue
		}
		summary := Summary{}
		for _, name := range unionNames(before, after) {
			change := Change{TypeURL: typeURL, Name: name}
			if r, ok := before[name]; ok {
				change.FromSHA256 = hash(r)
			}
			if r, ok := after[name]; ok {
				change.ToSHA256 = hash(r)
			}
			switch {
			case change.FromSHA256 == "":
				change.Change, summary.Added = "added", summary.Added+1
			case change.ToSHA256 == "":
				change.Change, summary.Removed = "removed", summary.Removed+1
			case change.FromSHA256 == change.ToSHA256:
				change.Change, summary.Same = "same", summary.Same+1
			case typeURL != resource.SecretType && sameDecoded(before[name], after[name]):
				change.Change, summary.SameDecoded = "sameDecoded", summary.SameDecoded+1
			default:
				change.Change, summary.Changed = "changed", summary.Changed+1
			}
			if all || change.Change != "same" {
				diff.Resources = append(diff.Resources, change)
			}
		}
		diff.Summary[typeURL] = summary
	}
	return diff, nil
}

// Resource is one resource of a snapshot.
type Resource struct {
	TypeURL string          `json:"typeUrl"`
	Name    string          `json:"name"`
	SHA256  string          `json:"sha256"`
	JSON    json.RawMessage `json:"json,omitempty"`
}

// Resource returns one resource of a snapshot of the node, decoded. A secret is never decoded: it holds
// private keys, so only its hash is returned.
func (c *Control) Resource(ctx context.Context, nodeID, which, typeURL, name string) (Resource, error) {
	s, _, err := c.snapshot(ctx, nodeID, which)
	if err != nil {
		return Resource{}, err
	}
	r, ok := s.GetResources(typeURL)[name]
	if !ok {
		return Resource{}, fmt.Errorf("%w: %s %s", ErrUnknownResource, typeURL, name)
	}
	result := Resource{TypeURL: typeURL, Name: name, SHA256: hash(r)}
	if typeURL == resource.SecretType {
		return result, nil
	}
	data, err := protojson.Marshal(r)
	if err != nil {
		return Resource{}, fmt.Errorf("decode %s %s: %w", typeURL, name, err)
	}
	result.JSON = data
	return result, nil
}

func (c *Control) snapshot(ctx context.Context, nodeID, which string) (cache.ResourceSnapshot, *time.Time, error) {
	if !c.known(nodeID) {
		return nil, nil, fmt.Errorf("%w: %s", ErrUnknownNode, nodeID)
	}
	switch which {
	case Served:
		served, ok := c.Cache.Served(nodeID)
		if !ok {
			return &cache.Snapshot{}, nil, nil
		}
		return served.Snapshot, &served.PublishedAt, nil
	case Built:
		built, err := c.Builder.DryBuildNode(ctx, nodeID)
		if built == nil {
			return nil, nil, err
		}
		return built, nil, nil
	}
	for _, published := range c.Cache.History(nodeID) {
		if published.ID() == which {
			return published.Snapshot, &published.PublishedAt, nil
		}
	}
	return nil, nil, fmt.Errorf("%w: %s", wrapped.ErrUnknownSnapshot, which)
}

func (c *Control) known(nodeID string) bool {
	if _, ok := c.Cache.GetNodeIDsAsMap()[nodeID]; ok {
		return true
	}
	_, frozen := c.Cache.Frozen(nodeID)
	return frozen
}

func (c *Control) inHistory(nodeID, id string) bool {
	for _, published := range c.Cache.History(nodeID) {
		if published.ID() == id {
			return true
		}
	}
	return false
}

func (c *Control) save(ctx context.Context) error {
	if c.Store == nil {
		return nil
	}
	return c.Store.Save(ctx, c.Cache.FrozenNodes(), c.Cache.VersionFloor())
}

func withoutNode(nodes map[string]wrapped.Freeze, nodeID string) map[string]wrapped.Freeze {
	result := make(map[string]wrapped.Freeze, len(nodes))
	for id, f := range nodes {
		if id != nodeID {
			result[id] = f
		}
	}
	return result
}

func withNode(nodes map[string]wrapped.Freeze, nodeID string, freeze wrapped.Freeze) map[string]wrapped.Freeze {
	result := make(map[string]wrapped.Freeze, len(nodes)+1)
	for id, f := range nodes {
		result[id] = f
	}
	result[nodeID] = freeze
	return result
}

// describe summarizes a snapshot; versions are left out for a dry build, they mean nothing there.
func describe(s cache.ResourceSnapshot, withVersions bool) SnapshotInfo {
	info := SnapshotInfo{ID: wrapped.SnapshotID(s), Versions: map[string]string{}, Counts: map[string]int{}}
	for _, typeURL := range typeURLs() {
		if n := len(s.GetResources(typeURL)); n > 0 {
			info.Counts[typeURL] = n
			if withVersions {
				info.Versions[typeURL] = s.GetVersion(typeURL)
			}
		}
	}
	return info
}

func typeURLs() []string {
	urls := make([]string, 0, types.UnknownType)
	for i := types.ResponseType(0); i < types.UnknownType; i++ {
		if url, err := cache.GetResponseTypeURL(i); err == nil {
			urls = append(urls, url)
		}
	}
	return urls
}

func unionNames(a, b map[string]types.Resource) []string {
	names := make([]string, 0, len(a)+len(b))
	for name := range a {
		names = append(names, name)
	}
	for name := range b {
		if _, ok := a[name]; !ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func hash(r types.Resource) string {
	data, err := proto.MarshalOptions{Deterministic: true}.Marshal(r)
	if err != nil {
		return "error"
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// sameDecoded compares two resources decoded, packed Any values included.
func sameDecoded(a, b types.Resource) bool {
	ja, errA := protojson.Marshal(a)
	jb, errB := protojson.Marshal(b)
	if errA != nil || errB != nil {
		return false
	}
	var va, vb any
	if json.Unmarshal(ja, &va) != nil || json.Unmarshal(jb, &vb) != nil {
		return false
	}
	return fmt.Sprint(va) == fmt.Sprint(vb)
}
