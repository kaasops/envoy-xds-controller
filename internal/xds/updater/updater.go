package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	wrapped "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder"
	"go.uber.org/multierr"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/proto"
)

type CacheUpdater struct {
	mx            sync.RWMutex
	snapshotCache *wrapped.SnapshotCache
	store         *store.Store
	usedSecrets   map[helpers.NamespacedName]helpers.NamespacedName
}

// buildVSResources is a function variable to allow stubbing in tests.
// In production it points to resbuilder.BuildResources.
var buildVSResources = resbuilder.BuildResources

func NewCacheUpdater(wsc *wrapped.SnapshotCache, store *store.Store) *CacheUpdater {
	return &CacheUpdater{
		snapshotCache: wsc,
		usedSecrets:   make(map[helpers.NamespacedName]helpers.NamespacedName),
		store:         store,
	}
}

// ErrLightValidationInsufficientCoverage is returned by the light validator when
// the current cache doesn't have snapshots for all target nodeIDs and correctness
// cannot be guaranteed; callers should fallback to heavy dry-run.
var ErrLightValidationInsufficientCoverage = errors.New("light validation insufficient coverage")

func (c *CacheUpdater) RebuildSnapshots(ctx context.Context) error {
	c.mx.Lock()
	defer c.mx.Unlock()
	return c.rebuildSnapshots(ctx)
}

func (c *CacheUpdater) CopyStore() *store.Store {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return c.store.Copy()
}

func (c *CacheUpdater) DryBuildSnapshotsWithVirtualService(ctx context.Context, vs *v1alpha1.VirtualService) error {
	c.mx.RLock()
	storeCopy := c.store.Copy()
	c.mx.RUnlock()
	storeCopy.SetVirtualService(vs)
	err, _ := buildSnapshots(ctx, wrapped.NewSnapshotCache(), storeCopy)
	return err
}

// DryValidateVirtualServiceLight performs a lightweight validation for a VirtualService without rebuilding
// full snapshots. It builds resources only for the specified VS, checks for duplicate domains within the VS,
// verifies listener address uniqueness across all Listeners in the store, and validates that the VS domains
// do not collide with existing domains present in the current snapshot cache for the relevant nodeIDs.
// Note: For updates of an existing VS, comparing to the current cache may lead to false positives if
// the same VS already contributes the same domains. Prefer using this on Create, or fall back to heavy dry-run on Update.
func (c *CacheUpdater) DryValidateVirtualServiceLight(ctx context.Context, vs *v1alpha1.VirtualService, prevVS *v1alpha1.VirtualService, validationIndices bool) error {
	// Honor cancellation upfront
	if err := ctx.Err(); err != nil {
		return err
	}

	// Work on a copy of the store and overlay the candidate VS
	c.mx.RLock()
	storeCopy := c.store.Copy()
	c.mx.RUnlock()
	storeCopy.SetVirtualService(vs)

	// Build resources for the candidate VS only
	vsRes, err := buildVSResources(vs, storeCopy)
	if err != nil {
		log.FromContext(ctx).Error(err, "buildVSResources failed for candidate VS", "vs", vs.GetLabelName())
		return fmt.Errorf("failed to build resources for VS: %w", err)
	}

	// Quick self-duplication check inside the same VS
	if err := validateDomainsWithinVS(vsRes.Domains); err != nil {
		return err
	}

	// Validate that listener addresses are unique across all listeners
	if err := validateListenerAddresses(storeCopy, c.snapshotCache, validationIndices); err != nil {
		return err
	}

	// Determine relevant nodeIDs for domain collision checks
	targetNodeIDs := resolveTargetNodeIDs(vs.GetNodeIDs(), c.snapshotCache)

	if len(targetNodeIDs) == 0 || len(vsRes.Domains) == 0 {
		return nil
	}

	// Build a map of existing domains per node
	existingByNode, err := c.buildExistingDomainsMap(ctx, targetNodeIDs, storeCopy, validationIndices)
	if err != nil {
		return err
	}

	// If prevVS is provided (update case), subtract its domains from existing sets on intersecting nodeIDs
	if prevVS != nil {
		if err := c.excludePreviousVSDomains(ctx, prevVS, targetNodeIDs, existingByNode); err != nil {
			return err
		}
	}

	// Compare candidate domains with existing ones per node
	if err := checkDomainCollisions(vsRes.Domains, targetNodeIDs, existingByNode); err != nil {
		return err
	}

	return nil
}

func (c *CacheUpdater) DryBuildSnapshotsWithVirtualServiceTemplate(ctx context.Context, vst *v1alpha1.VirtualServiceTemplate) error {
	c.mx.RLock()
	storeCopy := c.store.Copy()
	c.mx.RUnlock()
	storeCopy.SetVirtualServiceTemplate(vst)
	err, _ := buildSnapshots(ctx, wrapped.NewSnapshotCache(), storeCopy)
	return err
}

func (c *CacheUpdater) rebuildSnapshots(ctx context.Context) error {
	rlog := log.FromContext(ctx).WithName("cache-updater")
	rlog.Info("rebuild snapshots started")
	start := time.Now()
	err, usedSecrets := buildSnapshots(ctx, c.snapshotCache, c.store)
	if err != nil {
		rlog.Error(err, "rebuild snapshots with errors", "duration", time.Since(start).String())
		return err
	}
	c.usedSecrets = usedSecrets
	rlog.Info("rebuild snapshots done", "duration", time.Since(start).String())
	return nil
}

// nolint: gocyclo
func buildSnapshots(
	ctx context.Context,
	snapshotCache *wrapped.SnapshotCache,
	store *store.Store,
) (error, map[helpers.NamespacedName]helpers.NamespacedName) {
	errs := make([]error, 0)

	// Honor context cancellation early
	if err := ctx.Err(); err != nil {
		return err, nil
	}

	mixer := NewMixer()

	// ---------------------------------------------

	usedSecrets := make(map[helpers.NamespacedName]helpers.NamespacedName)
	nodeIDDomainsSet := make(map[string]struct{})

	// Optional index of domains per node for light validation (feature-flagged)
	buildDomainsIndex := getValidationIndicesEnabled()
	var nodeDomainsIndex map[string]map[string]struct{}
	if buildDomainsIndex {
		nodeDomainsIndex = make(map[string]map[string]struct{})
	}

	nodeIDsForCleanup := snapshotCache.GetNodeIDsAsMap()
	var commonVirtualServices []*v1alpha1.VirtualService

	for _, vs := range store.MapVirtualServices() {
		// Check cancellation between iterations
		if err := ctx.Err(); err != nil {
			return err, usedSecrets
		}

		vs.UpdateStatus(false, "")

		vsNodeIDs := vs.GetNodeIDs()
		if len(vsNodeIDs) == 0 {
			err := fmt.Errorf("virtual service %s/%s has no node IDs", vs.Namespace, vs.Name)
			vs.UpdateStatus(true, err.Error())
			errs = append(errs, err)
			continue
		}

		if isCommonVirtualService(vsNodeIDs) {
			commonVirtualServices = append(commonVirtualServices, vs)
			continue
		}

		// Build resources for VS (may be heavy); check ctx before and after
		if err := ctx.Err(); err != nil {
			return err, usedSecrets
		}
		vsRes, err := resbuilder.BuildResources(vs, store)
		if err != nil {
			vs.UpdateStatus(true, err.Error())
			errs = append(errs, err)
			continue
		}
		if err := ctx.Err(); err != nil {
			return err, usedSecrets
		}

		for _, secret := range vsRes.UsedSecrets {
			usedSecrets[secret] = helpers.NamespacedName{Name: vs.Name, Namespace: vs.Namespace}
		}

		for _, nodeID := range vsNodeIDs {
			// Check ctx inside nested loops too
			if err := ctx.Err(); err != nil {
				return err, usedSecrets
			}

			for _, domain := range vsRes.Domains {
				if err := ctx.Err(); err != nil { // cheap check
					return err, usedSecrets
				}
				nodeDom := nodeIDDomain(nodeID, domain)
				if _, ok := nodeIDDomainsSet[nodeDom]; ok {
					return fmt.Errorf("duplicate domain %s for node %s", domain, nodeID), nil
				}
				nodeIDDomainsSet[nodeDom] = struct{}{}
				if buildDomainsIndex {
					set, ok := nodeDomainsIndex[nodeID]
					if !ok {
						set = make(map[string]struct{})
						nodeDomainsIndex[nodeID] = set
					}
					set[domain] = struct{}{}
				}
			}

			if vsRes.RouteConfig != nil {
				mixer.Add(nodeID, resource.RouteType, vsRes.RouteConfig)
			}
			for _, cl := range vsRes.Clusters {
				mixer.Add(nodeID, resource.ClusterType, cl)
			}
			for _, secret := range vsRes.Secrets {
				mixer.Add(nodeID, resource.SecretType, secret)
			}
			mixer.AddListenerParams(vsRes.Listener, vsRes.FilterChain, nodeID)
		}
	}

	if len(commonVirtualServices) > 0 {
		for _, vs := range commonVirtualServices {
			if err := ctx.Err(); err != nil {
				return err, usedSecrets
			}
			vsRes, err := resbuilder.BuildResources(vs, store)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if err := ctx.Err(); err != nil {
				return err, usedSecrets
			}
			for _, secret := range vsRes.UsedSecrets {
				usedSecrets[secret] = helpers.NamespacedName{Name: vs.Name, Namespace: vs.Namespace}
			}

			for nodeID := range mixer.nodeIDs {
				if err := ctx.Err(); err != nil {
					return err, usedSecrets
				}

				for _, domain := range vsRes.Domains {
					if err := ctx.Err(); err != nil {
						return err, usedSecrets
					}
					nodeDom := nodeIDDomain(nodeID, domain)
					if _, ok := nodeIDDomainsSet[nodeDom]; ok {
						return fmt.Errorf("duplicate domain %s for node %s", domain, nodeID), nil
					}
					nodeIDDomainsSet[nodeDom] = struct{}{}
					if buildDomainsIndex {
						set, ok := nodeDomainsIndex[nodeID]
						if !ok {
							set = make(map[string]struct{})
							nodeDomainsIndex[nodeID] = set
						}
						set[domain] = struct{}{}
					}
				}

				if vsRes.RouteConfig != nil {
					mixer.Add(nodeID, resource.RouteType, vsRes.RouteConfig)
				}
				for _, cl := range vsRes.Clusters {
					mixer.Add(nodeID, resource.ClusterType, cl)
				}
				for _, secret := range vsRes.Secrets {
					mixer.Add(nodeID, resource.SecretType, secret)
				}
				mixer.AddListenerParams(vsRes.Listener, vsRes.FilterChain, nodeID)
			}
		}
	}

	if err := ctx.Err(); err != nil {
		return err, usedSecrets
	}
	tmp, err := mixer.Mix(store)
	if err != nil {
		errs = append(errs, err)
		return multierr.Combine(errs...), usedSecrets
	}

	// Stage snapshots to avoid partial updates on cancellation or errors
	staged := make(map[string]cache.ResourceSnapshot)

	for nodeID, resMap := range tmp {
		if err := ctx.Err(); err != nil {
			return err, usedSecrets
		}
		var snapshot *cache.Snapshot
		var err error
		var hasChanges bool
		prevSnapshot, _ := snapshotCache.GetSnapshot(nodeID)
		if prevSnapshot != nil {
			snapshot, hasChanges, err = updateSnapshot(prevSnapshot, resMap)
			if err != nil {
				errs = append(errs, err)
				continue
			}
		} else {
			hasChanges = true
			snapshot, err = cache.NewSnapshot("1", resMap)
			if err != nil {
				errs = append(errs, err)
				continue
			}
		}
		if hasChanges {
			if err := snapshot.Consistent(); err != nil {
				errs = append(errs, err)
				continue
			}
			// Stage for commit later
			staged[nodeID] = snapshot
		}
		// Mark as not needing cleanup since we have a plan for this node
		delete(nodeIDsForCleanup, nodeID)
	}

	// If any error occurred during staging, abort without mutating cache
	if len(errs) > 0 {
		return multierr.Combine(errs...), usedSecrets
	}
	// Abort on cancellation prior to commit to avoid partial state
	if err := ctx.Err(); err != nil {
		return err, usedSecrets
	}

	// Commit phase: ensure we are not interrupted mid-commit
	commitCtx := context.WithoutCancel(ctx)
	for nodeID, snapshot := range staged {
		if err := snapshotCache.SetSnapshot(commitCtx, nodeID, snapshot); err != nil {
			errs = append(errs, err)
		}
	}
	for nodeID := range nodeIDsForCleanup {
		if err := snapshotCache.SetSnapshot(commitCtx, nodeID, &cache.Snapshot{}); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return multierr.Combine(errs...), usedSecrets
	}

	// Update Store domain index after successful commit (feature-flagged)
	if buildDomainsIndex {
		// Ensure index has entries (possibly empty) for all nodes we touched this rebuild
		allNodes := make(map[string]struct{}, len(staged)+len(nodeIDsForCleanup))
		for nodeID := range staged {
			allNodes[nodeID] = struct{}{}
		}
		for nodeID := range nodeIDsForCleanup {
			allNodes[nodeID] = struct{}{}
		}
		for nodeID := range allNodes {
			if _, ok := nodeDomainsIndex[nodeID]; !ok {
				nodeDomainsIndex[nodeID] = make(map[string]struct{})
			}
		}
		store.ReplaceNodeDomainsIndex(nodeDomainsIndex)
	}

	return nil, usedSecrets
}

func (c *CacheUpdater) GetUsedSecrets() map[helpers.NamespacedName]helpers.NamespacedName {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return maps.Clone(c.usedSecrets)
}

func (c *CacheUpdater) GetMarshaledStore() ([]byte, error) {
	c.mx.RLock()
	defer c.mx.RUnlock()
	data := make(map[string]map[string]any)
	data["virtualServices"] = make(map[string]any)
	data["clusters"] = make(map[string]any)
	data["specClusters"] = make(map[string]any)
	data["secrets"] = make(map[string]any)
	data["routes"] = make(map[string]any)
	data["listeners"] = make(map[string]any)
	data["virtualServiceTemplates"] = make(map[string]any)
	data["accessLogConfigs"] = make(map[string]any)
	data["httpFilters"] = make(map[string]any)
	data["policies"] = make(map[string]any)
	data["domainToSecret"] = make(map[string]any)

	for key, vs := range c.store.MapVirtualServices() {
		data["virtualServices"][key.String()] = vs
	}
	for key, cl := range c.store.MapClusters() {
		data["clusters"][key.String()] = cl
	}
	for key, secret := range c.store.MapSecrets() {
		data["secrets"][key.String()] = secret
	}
	for key, route := range c.store.MapRoutes() {
		data["routes"][key.String()] = route
	}
	for key, listener := range c.store.MapListeners() {
		data["listeners"][key.String()] = listener
	}
	for key, vst := range c.store.MapVirtualServiceTemplates() {
		data["virtualServiceTemplates"][key.String()] = vst
	}
	for key, alc := range c.store.MapAccessLogs() {
		data["accessLogConfigs"][key.String()] = alc
	}
	for key, httpFilter := range c.store.MapHTTPFilters() {
		data["httpFilters"][key.String()] = httpFilter
	}
	for key, policy := range c.store.MapPolicies() {
		data["policies"][key.String()] = policy
	}
	for ds, s := range c.store.MapDomainSecrets() {
		data["domainToSecret"][ds] = s
	}
	for specCluster, cl := range c.store.MapSpecClusters() {
		data["specClusters"][specCluster] = cl
	}
	return json.MarshalIndent(data, "", "\t")
}

func updateSnapshot(prevSnapshot cache.ResourceSnapshot, resources map[resource.Type][]types.Resource) (*cache.Snapshot, bool, error) {
	if prevSnapshot == nil {
		return nil, false, errors.New("snapshot is nil")
	}

	hasChanges := false

	snapshot := cache.Snapshot{}
	for typ, res := range resources {
		index := cache.GetResponseType(typ)
		if index == types.UnknownType {
			return nil, false, errors.New("unknown resource type: " + typ)
		}

		version := prevSnapshot.GetVersion(typ)
		if version == "" {
			version = "0"
		}

		if checkResourcesChanged(prevSnapshot.GetResources(typ), res) {
			hasChanges = true
			vInt, _ := strconv.Atoi(version)
			vInt++
			version = strconv.Itoa(vInt)
		}

		snapshot.Resources[index] = cache.NewResources(version, res)
	}
	return &snapshot, hasChanges, nil
}

func checkResourcesChanged(prevRes map[string]types.Resource, newRes []types.Resource) bool {
	if len(prevRes) != len(newRes) {
		return true
	}
	for _, newR := range newRes {
		if val, ok := prevRes[getName(newR)]; ok {
			if !proto.Equal(val, newR) {
				return true
			}
		} else {
			return true
		}
	}
	return false
}

func getName(msg proto.Message) string {
	msgDesc := msg.ProtoReflect().Descriptor()
	for i := 0; i < msgDesc.Fields().Len(); i++ {
		if msgDesc.Fields().Get(i).Name() == "name" {
			return msg.ProtoReflect().Get(msgDesc.Fields().Get(i)).String()
		}
	}
	return ""
}

func isCommonVirtualService(nodeIDs []string) bool {
	return len(nodeIDs) == 1 && nodeIDs[0] == "*"
}

func resolveTargetNodeIDs(nodeIDs []string, snapshotCache *wrapped.SnapshotCache) []string {
	if isCommonVirtualService(nodeIDs) {
		// common VS affects all known nodes
		return snapshotCache.GetNodeIDs()
	}
	return nodeIDs
}

func nodeIDDomain(nodeID, domain string) string {
	return nodeID + ":" + domain
}

// getValidationIndicesEnabled returns true if Store-backed indices should be used for validation shortcuts.
// Controlled by WEBHOOK_VALIDATION_INDICES env var (true/1/yes/on).
func getValidationIndicesEnabled() bool {
	v := strings.ToLower(os.Getenv("WEBHOOK_VALIDATION_INDICES"))
	switch v {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// validateDomainsWithinVS checks for duplicate domains within the same VirtualService.
func validateDomainsWithinVS(domains []string) error {
	seen := make(map[string]struct{}, len(domains))
	for _, d := range domains {
		if _, ok := seen[d]; ok {
			return fmt.Errorf("duplicate domain '%s' within VirtualService", d)
		}
		seen[d] = struct{}{}
	}
	return nil
}

// validateListenerAddresses validates that listener addresses are unique within each nodeID (snapshot).
// This replaces the previous global validation approach which incorrectly prevented
// the same address:port from being used across different nodeIDs (different snapshots).
func validateListenerAddresses(storeCopy *store.Store, snapshotCache *wrapped.SnapshotCache, validationIndices bool) error {
	// Build mapping of listeners to nodeIDs
	listenerToNodeIDs := buildListenerToNodeIDsMapping(storeCopy, snapshotCache)

	if validationIndices {
		// Use the fast index-based approach but check per nodeID
		if err := validateListenerAddressesPerNodeID(storeCopy, listenerToNodeIDs); err != nil {
			return err
		}
	}
	// Always run the thorough check to catch structural issues (missing address, etc.)
	if err := validateListenerAddressesUniquePerNodeID(storeCopy, listenerToNodeIDs); err != nil {
		return err
	}
	return nil
}

// buildListenerToNodeIDsMapping creates a mapping of listener names to the nodeIDs that use them.
// This is determined by analyzing which VirtualServices reference each listener and what nodeIDs
// those VirtualServices target. For common VirtualServices (nodeID = "*"), expands to all actual nodeIDs.
func buildListenerToNodeIDsMapping(storeCopy *store.Store, snapshotCache *wrapped.SnapshotCache) map[helpers.NamespacedName][]string {
	mapping := make(map[helpers.NamespacedName][]string)

	// Iterate through all VirtualServices to find which listeners they use
	for _, vs := range storeCopy.MapVirtualServices() {
		vsNodeIDs := vs.GetNodeIDs()
		if len(vsNodeIDs) == 0 {
			continue // Skip VirtualServices without nodeIDs
		}

		// Get the listener referenced by this VirtualService
		if vs.Spec.Listener != nil && vs.Spec.Listener.Name != "" {
			listenerNN := helpers.NamespacedName{
				Namespace: vs.Namespace, // Listener is in the same namespace as VirtualService
				Name:      vs.Spec.Listener.Name,
			}

			// Resolve nodeIDs, expanding "*" to actual nodeIDs for common VirtualServices
			resolvedNodeIDs := resolveTargetNodeIDs(vsNodeIDs, snapshotCache)

			// Add nodeIDs to this listener's mapping
			existing := mapping[listenerNN]
			for _, nodeID := range resolvedNodeIDs {
				// Check if nodeID is already in the list to avoid duplicates
				found := false
				for _, existingNodeID := range existing {
					if existingNodeID == nodeID {
						found = true
						break
					}
				}
				if !found {
					existing = append(existing, nodeID)
				}
			}
			mapping[listenerNN] = existing
		}
	}

	return mapping
}

// validateListenerAddressesPerNodeID validates listener addresses using indices, but per nodeID.
func validateListenerAddressesPerNodeID(storeCopy *store.Store, listenerToNodeIDs map[helpers.NamespacedName][]string) error {
	// Group listeners by nodeID
	nodeIDToListeners := make(map[string][]helpers.NamespacedName)
	for listenerNN, nodeIDs := range listenerToNodeIDs {
		for _, nodeID := range nodeIDs {
			nodeIDToListeners[nodeID] = append(nodeIDToListeners[nodeID], listenerNN)
		}
	}

	// Check for duplicates within each nodeID
	for nodeID, listeners := range nodeIDToListeners {
		addrToListener := make(map[string]helpers.NamespacedName)

		for _, listenerNN := range listeners {
			listener := storeCopy.MapListeners()[listenerNN]
			if listener == nil {
				continue // Listener not found, skip
			}

			lv3, err := listener.UnmarshalV3()
			if err != nil {
				continue // Skip malformed listeners, validation will catch this later
			}

			addr := lv3.GetAddress()
			if addr == nil || addr.GetSocketAddress() == nil {
				continue // Skip incomplete addresses
			}

			host := addr.GetSocketAddress().GetAddress()
			port := addr.GetSocketAddress().GetPortValue()
			hostPort := fmt.Sprintf("%s:%d", host, port)

			if existingListener, exists := addrToListener[hostPort]; exists {
				return fmt.Errorf("listener '%s' has duplicate address '%s' as existing listener '%s' within nodeID '%s'",
					listenerNN.String(), hostPort, existingListener.String(), nodeID)
			}
			addrToListener[hostPort] = listenerNN
		}
	}

	return nil
}

// validateListenerAddressesUniquePerNodeID performs thorough validation of listener addresses per nodeID.
func validateListenerAddressesUniquePerNodeID(storeCopy *store.Store, listenerToNodeIDs map[helpers.NamespacedName][]string) error {
	// Group listeners by nodeID
	nodeIDToListeners := make(map[string][]helpers.NamespacedName)
	for listenerNN, nodeIDs := range listenerToNodeIDs {
		for _, nodeID := range nodeIDs {
			nodeIDToListeners[nodeID] = append(nodeIDToListeners[nodeID], listenerNN)
		}
	}

	// Check for duplicates within each nodeID with thorough validation
	for nodeID, listeners := range nodeIDToListeners {
		addrToListener := make(map[string]helpers.NamespacedName)

		for _, listenerNN := range listeners {
			listener := storeCopy.MapListeners()[listenerNN]
			if listener == nil {
				return fmt.Errorf("listener %s referenced by nodeID %s not found", listenerNN.String(), nodeID)
			}

			lv3, err := listener.UnmarshalV3()
			if err != nil {
				return fmt.Errorf("failed to unmarshal listener %s to v3: %w", listenerNN.String(), err)
			}

			addr := lv3.GetAddress()
			if addr == nil {
				return fmt.Errorf("listener %s has no address configured", listenerNN.String())
			}

			sa := addr.GetSocketAddress()
			if sa == nil {
				return fmt.Errorf("listener %s has no socket address configured", listenerNN.String())
			}

			host := sa.GetAddress()
			port := sa.GetPortValue()
			hostPort := fmt.Sprintf("%s:%d", host, port)

			if existingListener, exists := addrToListener[hostPort]; exists {
				return fmt.Errorf("listener '%s' has duplicate address '%s' as existing listener '%s' within nodeID '%s'",
					listenerNN.String(), hostPort, existingListener.String(), nodeID)
			}
			addrToListener[hostPort] = listenerNN
		}
	}

	return nil
}

// buildExistingDomainsMap builds a map of existing domains per node from either indices or snapshot cache.
func (c *CacheUpdater) buildExistingDomainsMap(ctx context.Context, targetNodeIDs []string, storeCopy *store.Store, validationIndices bool) (map[string]map[string]struct{}, error) {
	existingByNode := make(map[string]map[string]struct{})
	if validationIndices {
		// Prefer Store-backed index when available; ensures O(1) lookup
		idx, missing := storeCopy.GetNodeDomainsForNodes(targetNodeIDs)
		if len(missing) > 0 {
			return nil, ErrLightValidationInsufficientCoverage
		}
		existingByNode = idx
	} else {
		missing := 0
		for _, nodeID := range targetNodeIDs {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			rcs, err := c.snapshotCache.GetRouteConfigurations(nodeID)
			if err != nil {
				// If snapshot for nodeID not found yet, we cannot reliably validate against existing domains
				missing++
				continue
			}
			set := make(map[string]struct{})
			for _, rc := range rcs {
				for _, vh := range rc.GetVirtualHosts() {
					for _, dom := range vh.GetDomains() {
						set[dom] = struct{}{}
					}
				}
			}
			existingByNode[nodeID] = set
		}
		// If any target node lacks snapshot coverage, ask caller to fallback to heavy dry-run
		if missing > 0 {
			return nil, ErrLightValidationInsufficientCoverage
		}
	}
	return existingByNode, nil
}

// excludePreviousVSDomains removes previous VirtualService domains from the existing domains map.
func (c *CacheUpdater) excludePreviousVSDomains(ctx context.Context, prevVS *v1alpha1.VirtualService, targetNodeIDs []string, existingByNode map[string]map[string]struct{}) error {
	// Build resources for previous VS using the original store state
	c.mx.RLock()
	origStoreCopy := c.store.Copy()
	c.mx.RUnlock()
	prevRes, err := buildVSResources(prevVS, origStoreCopy)
	if err != nil {
		log.FromContext(ctx).Error(err, "buildVSResources failed for previous VS", "prevVS", prevVS.GetLabelName())
		return fmt.Errorf("failed to build resources for previous VS: %w", err)
	}
	prevNodeIDs := resolveTargetNodeIDs(prevVS.GetNodeIDs(), c.snapshotCache)
	// Build a quick set of prev domains for faster deletion
	prevDomains := make(map[string]struct{}, len(prevRes.Domains))
	for _, d := range prevRes.Domains {
		prevDomains[d] = struct{}{}
	}
	// Build a set for prev nodeIDs for O(1) membership checks
	prevSet := make(map[string]struct{}, len(prevNodeIDs))
	for _, p := range prevNodeIDs {
		prevSet[p] = struct{}{}
	}
	// Delete prev domains only from nodes that are both in targetNodeIDs and prevNodeIDs
	for _, nodeID := range targetNodeIDs {
		if set, ok := existingByNode[nodeID]; ok {
			if _, had := prevSet[nodeID]; had {
				for d := range prevDomains {
					delete(set, d)
				}
			}
		}
	}
	return nil
}

// checkDomainCollisions validates that candidate domains don't collide with existing ones per node.
func checkDomainCollisions(candidateDomains []string, targetNodeIDs []string, existingByNode map[string]map[string]struct{}) error {
	for _, nodeID := range targetNodeIDs {
		set := existingByNode[nodeID]
		if len(set) == 0 {
			continue
		}
		for _, d := range candidateDomains {
			if _, ok := set[d]; ok {
				return fmt.Errorf("duplicate domain '%s' for node %s", d, nodeID)
			}
		}
	}
	return nil
}
