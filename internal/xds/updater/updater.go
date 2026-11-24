package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	wrapped "github.com/kaasops/envoy-xds-controller/internal/xds/cache"
	"go.uber.org/multierr"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/proto"
)

// CacheUpdater manages the xDS snapshot cache and coordinates updates between
// Kubernetes resources and Envoy configuration.
//
// INVARIANT: All resources stored in the store are normalized via their respective
// NormalizeSpec() methods before being saved. This ensures that IsEqual() comparisons
// between stored resources and incoming resources are valid without re-normalizing
// the stored copy. Apply methods must call NormalizeSpec() on incoming resources
// before comparison and storage.
type CacheUpdater struct {
	publications  map[helpers.NamespacedName]publicationObservation
	contributors  publishedContributors
	enqueueStatus func(helpers.NamespacedName)
	mx            sync.RWMutex
	snapshotCache *wrapped.SnapshotCache
	store         store.Store
	usedSecrets   map[helpers.NamespacedName]helpers.NamespacedName
}

// VSStatus represents the status of a VirtualService after processing
type VSStatus struct {
	Invalid       bool
	Message       string
	AffectedNodes []string
}

// buildMetrics tracks timing for different phases of snapshot building
type buildMetrics struct {
	initDuration           time.Duration
	vsProcessingDuration   time.Duration
	commonVSsDuration      time.Duration
	listenerBuildDuration  time.Duration
	snapshotCreateDuration time.Duration
	totalVSCount           int
	processedVSCount       int
	commonVSCount          int
}

var buildVSResources = buildResourcesAdapter

func NewCacheUpdater(wsc *wrapped.SnapshotCache, store store.Store) *CacheUpdater {
	return &CacheUpdater{
		publications:  make(map[helpers.NamespacedName]publicationObservation),
		contributors:  make(publishedContributors),
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

func (c *CacheUpdater) CopyStore() store.Store {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return c.store.Copy()
}

func (c *CacheUpdater) DryBuildSnapshotsWithVirtualService(ctx context.Context, vs *v1alpha1.VirtualService) error {
	c.mx.RLock()
	storeCopy := c.store.Copy()
	c.mx.RUnlock()
	storeCopy.SetVirtualService(vs)
	err, _, _, _ := buildSnapshots(ctx, wrapped.NewSnapshotCache(), storeCopy, nil)
	return err
}

// DryValidateVirtualServiceLight performs a lightweight validation for a VirtualService without rebuilding
// full snapshots. It builds resources only for the specified VS, checks for duplicate domains within the VS,
// verifies listener address uniqueness across all Listeners in the store, and validates that the VS domains
// do not collide with existing domains present in the current snapshot cache for the relevant nodeIDs.
// Note: For updates of an existing VS, comparing to the current cache may lead to false positives
// if the same VS already contributes the same domains. Prefer using this on Create, or fall back
// to heavy dry-run on Update.
// A listener address conflict on one of toleratedNodeIDs is not reported: such a node keeps its
// previous snapshot until the conflict is gone, and the change itself is checked for other nodes.
func (c *CacheUpdater) DryValidateVirtualServiceLight(
	ctx context.Context,
	vs *v1alpha1.VirtualService,
	prevVS *v1alpha1.VirtualService,
	validationIndices bool,
	toleratedNodeIDs []string,
) error {
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
	if err := validateListenerAddresses(storeCopy, c.snapshotCache, validationIndices,
		nodeSet(toleratedNodeIDs)); err != nil {
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
	if err := checkDomainCollisions(vsRes.Domains, vsRes.Listener, targetNodeIDs, existingByNode); err != nil {
		return err
	}

	return nil
}

// DryValidateListenerAddresses is the listener part of DryValidateVirtualServiceLight: with vs applied,
// every listener a service refers to exists and has a socket address, and no two listeners of a node
// share an address, except on toleratedNodeIDs.
func (c *CacheUpdater) DryValidateListenerAddresses(
	ctx context.Context,
	vs *v1alpha1.VirtualService,
	validationIndices bool,
	toleratedNodeIDs []string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mx.RLock()
	storeCopy := c.store.Copy()
	c.mx.RUnlock()
	storeCopy.SetVirtualService(vs)
	return validateListenerAddresses(storeCopy, c.snapshotCache, validationIndices, nodeSet(toleratedNodeIDs))
}

func nodeSet(nodeIDs []string) map[string]struct{} {
	set := make(map[string]struct{}, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		set[nodeID] = struct{}{}
	}
	return set
}

func (c *CacheUpdater) DryBuildSnapshotsWithVirtualServiceTemplate(
	ctx context.Context,
	vst *v1alpha1.VirtualServiceTemplate,
) error {
	rlog := log.FromContext(ctx).WithName("cache-updater")

	// Always log webhook validation start for diagnostics
	validationStart := time.Now()

	// Extract validation ID from context if present (for log correlation)
	validationID := ""
	if val := ctx.Value("validationID"); val != nil {
		if id, ok := val.(string); ok {
			validationID = id
		}
	}

	logFields := []interface{}{"template", vst.Name}
	if validationID != "" {
		logFields = append(logFields, "validationID", validationID)
	}
	rlog.Info("DryBuildSnapshots: started", logFields...)

	// Log before attempting to acquire lock to detect lock contention
	lockStart := time.Now()
	c.mx.RLock()
	lockAcquireDuration := time.Since(lockStart)
	if lockAcquireDuration > 100*time.Millisecond {
		rlog.Info("DryBuildSnapshots: lock contention detected",
			"template", vst.Name, "lockWaitDuration", lockAcquireDuration.String())
	}

	start := time.Now()
	storeCopy := c.store.Copy()
	c.mx.RUnlock()
	copyDuration := time.Since(start)

	if copyDuration > 500*time.Millisecond {
		rlog.Info("DryBuildSnapshots: slow store copy", "template", vst.Name, "duration", copyDuration.String())
	}

	// Get resource counts for diagnostics
	vsCount := len(storeCopy.MapVirtualServices())
	secretCount := len(storeCopy.MapDomainSecrets())
	clusterCount := len(storeCopy.MapClusters())
	listenerCount := len(storeCopy.MapListeners())
	routeCount := len(storeCopy.MapRoutes())

	storeCopy.SetVirtualServiceTemplate(vst)

	buildStart := time.Now()
	err, usedSecrets, vsStatuses, metrics := buildSnapshots(ctx, wrapped.NewSnapshotCache(), storeCopy, nil)
	buildDuration := time.Since(buildStart)
	totalDuration := time.Since(validationStart)

	// Always log completion for webhook validations (critical for timeout diagnosis)
	completionLogFields := []interface{}{
		"template", vst.Name,
		"lockWaitDuration", lockAcquireDuration.String(),
		"storeCopyDuration", copyDuration.String(),
		"buildDuration", buildDuration.String(),
		"totalDuration", totalDuration.String(),
		"vsCount", vsCount,
		"secretCount", secretCount,
		"clusterCount", clusterCount,
		"listenerCount", listenerCount,
		"routeCount", routeCount,
		"usedSecretsCount", len(usedSecrets),
		"vsStatusesCount", len(vsStatuses),
		"hasError", err != nil,
		// Detailed build phase timings
		"build.initDuration", metrics.initDuration.String(),
		"build.vsProcessingDuration", metrics.vsProcessingDuration.String(),
		"build.commonVSsDuration", metrics.commonVSsDuration.String(),
		"build.listenerBuildDuration", metrics.listenerBuildDuration.String(),
		"build.snapshotCreateDuration", metrics.snapshotCreateDuration.String(),
		"build.totalVSCount", metrics.totalVSCount,
		"build.processedVSCount", metrics.processedVSCount,
		"build.commonVSCount", metrics.commonVSCount,
	}
	if validationID != "" {
		completionLogFields = append(completionLogFields, "validationID", validationID)
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			rlog.Error(err, "DryBuildSnapshots: timeout exceeded", completionLogFields...)
		} else {
			rlog.Error(err, "DryBuildSnapshots: completed with error", completionLogFields...)
		}
	} else {
		rlog.Info("DryBuildSnapshots: completed successfully", completionLogFields...)
	}

	return err
}

func (c *CacheUpdater) rebuildSnapshots(ctx context.Context) error {
	rlog := log.FromContext(ctx).WithName("cache-updater")
	rlog.Info("rebuild snapshots started")
	start := time.Now()
	err, usedSecrets, vsStatuses, _ := buildSnapshots(ctx, c.snapshotCache, c.store, c.contributors)

	paused := c.snapshotCache.PausedNodes()
	for key, status := range vsStatuses {
		if len(paused) > 0 {
			if vs := c.store.GetVirtualService(key); vs != nil {
				status = withPauseNote(status, resolveTargetNodeIDs(vs.GetNodeIDs(), c.snapshotCache), paused)
			}
		}
		c.recordPublication(key, status)
	}

	if err != nil {
		rlog.Error(err, "rebuild snapshots with errors", "duration", time.Since(start).String())
		if onlyListenerConflicts(err) {
			// Nodes held by a listener address conflict still serve their previous snapshots: keep
			// the secrets those use next to the secrets of the nodes published now.
			c.usedSecrets = mergeUsedSecrets(c.usedSecrets, usedSecrets)
		}
		return err
	}
	c.usedSecrets = usedSecrets

	rlog.Info("rebuild snapshots done", "duration", time.Since(start).String())
	return nil
}

// getRootCause unwraps an error chain to extract the innermost (root) error.
// This is used to provide concise error messages to users while preserving
// the full error chain in logs.
func getRootCause(err error) error {
	if err == nil {
		return nil
	}
	for {
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			return err
		}
		err = unwrapped
	}
}

// nolint: gocyclo
func buildSnapshots(
	ctx context.Context,
	snapshotCache *wrapped.SnapshotCache,
	store store.Store,
	published publishedContributors,
) (error, map[helpers.NamespacedName]helpers.NamespacedName, map[helpers.NamespacedName]VSStatus, buildMetrics) {
	metrics := buildMetrics{}
	initStart := time.Now()

	errs := make([]error, 0)
	vsStatuses := make(map[helpers.NamespacedName]VSStatus)

	// Honor context cancellation early
	if err := ctx.Err(); err != nil {
		return err, nil, nil, metrics
	}

	mixer := NewMixer()

	// ---------------------------------------------

	usedSecrets := make(map[helpers.NamespacedName]helpers.NamespacedName)
	domainOwners := make(map[string]helpers.NamespacedName)

	// Optional index of domains per node for light validation (feature-flagged)
	buildDomainsIndex := getValidationIndicesEnabled()
	var nodeDomainsIndex map[string]map[string]struct{}
	if buildDomainsIndex {
		nodeDomainsIndex = make(map[string]map[string]struct{})
	}

	nodeIDsForCleanup := snapshotCache.GetNodeIDsAsMap()
	var commonVirtualServices []*v1alpha1.VirtualService

	metrics.initDuration = time.Since(initStart)
	vsProcessingStart := time.Now()

	allVSs := store.MapVirtualServices()
	metrics.totalVSCount = len(allVSs)

	firstBlocker := make(map[string]string)
	recordBuildFailure := func(nn helpers.NamespacedName, reason string) {
		vsStatuses[nn] = VSStatus{Invalid: true, Message: boundedStatusText(reason, maxPublicationDetail)}
		if first, ok := firstBlocker[nn.Namespace]; !ok || nn.Name < first {
			firstBlocker[nn.Namespace] = nn.Name
		}
	}
	globalFailure := func() map[helpers.NamespacedName]VSStatus {
		return publicationResults(allVSs, vsStatuses, firstBlocker, nil, published.retained(allVSs, nil))
	}
	checkDomain := func(vsNN helpers.NamespacedName, nodeID string, listener helpers.NamespacedName, domain string) error {
		listenerKey := listenerDomain(listener, domain)
		key := nodeIDDomain(nodeID, listenerKey)
		if owner, exists := domainOwners[key]; exists {
			err := fmt.Errorf("duplicate domain '%s' for node %s on listener %s", domain, nodeID, listener.String())
			detail := fmt.Sprintf("duplicate domain %q on node %s", domain, nodeID)
			if owner.Namespace == vsNN.Namespace {
				detail += fmt.Sprintf(" (VirtualServices %s and %s)", owner.String(), vsNN.String())
			}
			recordBuildFailure(owner, detail)
			recordBuildFailure(vsNN, detail)
			return err
		}
		domainOwners[key] = vsNN
		if buildDomainsIndex {
			if nodeDomainsIndex[nodeID] == nil {
				nodeDomainsIndex[nodeID] = make(map[string]struct{})
			}
			nodeDomainsIndex[nodeID][listenerKey] = struct{}{}
		}
		return nil
	}

	keys := maps.Keys(allVSs)
	slices.SortFunc(keys, func(a, b helpers.NamespacedName) int {
		if c := strings.Compare(a.Namespace, b.Namespace); c != 0 {
			return c
		}
		return strings.Compare(a.Name, b.Name)
	})
	for _, vsNN := range keys {
		vs := allVSs[vsNN]
		// Check cancellation between iterations
		if err := ctx.Err(); err != nil {
			return err, usedSecrets, nil, metrics
		}

		vsNodeIDs := vs.GetNodeIDs()
		if len(vsNodeIDs) == 0 {
			err := fmt.Errorf("virtual service %s/%s has no node IDs", vs.Namespace, vs.Name)
			recordBuildFailure(vsNN, getRootCause(err).Error())
			errs = append(errs, err)
			continue
		}

		if isCommonVirtualService(vsNodeIDs) {
			commonVirtualServices = append(commonVirtualServices, vs)
			metrics.commonVSCount++
			continue
		}

		// Build resources for VS (may be heavy); check ctx before and after
		if err := ctx.Err(); err != nil {
			return err, usedSecrets, nil, metrics
		}
		vsRes, err := buildVSResources(vs, store)
		if err != nil {
			recordBuildFailure(vsNN, getRootCause(err).Error())
			errs = append(errs, err)
			continue
		}
		if err := ctx.Err(); err != nil {
			return err, usedSecrets, nil, metrics
		}
		vsStatuses[vsNN] = resourceWarnings(vsRes.Warnings)
		metrics.processedVSCount++

		for _, secret := range vsRes.UsedSecrets {
			usedSecrets[secret] = helpers.NamespacedName{Name: vs.Name, Namespace: vs.Namespace}
		}

		for _, nodeID := range vsNodeIDs {
			// Check ctx inside nested loops too
			if err := ctx.Err(); err != nil {
				return err, usedSecrets, nil, metrics
			}

			for _, domain := range vsRes.Domains {
				if err := ctx.Err(); err != nil { // cheap check
					return err, usedSecrets, nil, metrics
				}
				if err := checkDomain(vsNN, nodeID, vsRes.Listener, domain); err != nil {
					return err, nil, globalFailure(), metrics
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

	metrics.vsProcessingDuration = time.Since(vsProcessingStart)

	// Process common VirtualServices
	commonVSsStart := time.Now()
	if len(commonVirtualServices) > 0 {
		nodes := maps.Keys(mixer.nodeIDs)
		slices.Sort(nodes)
		for _, vs := range commonVirtualServices {
			vsNN := helpers.NamespacedName{Namespace: vs.Namespace, Name: vs.Name}
			if err := ctx.Err(); err != nil {
				return err, usedSecrets, nil, metrics
			}
			vsRes, err := buildVSResources(vs, store)
			if err != nil {
				recordBuildFailure(vsNN, getRootCause(err).Error())
				errs = append(errs, err)
				continue
			}
			if err := ctx.Err(); err != nil {
				return err, usedSecrets, nil, metrics
			}
			vsStatuses[vsNN] = resourceWarnings(vsRes.Warnings)
			for _, secret := range vsRes.UsedSecrets {
				usedSecrets[secret] = helpers.NamespacedName{Name: vs.Name, Namespace: vs.Namespace}
			}

			for _, nodeID := range nodes {
				if err := ctx.Err(); err != nil {
					return err, usedSecrets, nil, metrics
				}

				for _, domain := range vsRes.Domains {
					if err := ctx.Err(); err != nil {
						return err, usedSecrets, nil, metrics
					}
					if err := checkDomain(vsNN, nodeID, vsRes.Listener, domain); err != nil {
						return err, nil, globalFailure(), metrics
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
	metrics.commonVSsDuration = time.Since(commonVSsStart)

	// Build listeners
	listenerBuildStart := time.Now()
	if err := ctx.Err(); err != nil {
		return err, usedSecrets, nil, metrics
	}
	tmp, err := mixer.Mix(store)
	if err != nil {
		errs = append(errs, err)
		return multierr.Combine(errs...), usedSecrets, globalFailure(), metrics
	}
	metrics.listenerBuildDuration = time.Since(listenerBuildStart)

	// Stage snapshots to avoid partial updates on cancellation or errors
	snapshotCreateStart := time.Now()
	staged := make(map[string]cache.ResourceSnapshot)

	for nodeID, resMap := range tmp {
		if err := ctx.Err(); err != nil {
			return err, usedSecrets, nil, metrics
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
		return multierr.Combine(errs...), usedSecrets, globalFailure(), metrics
	}
	// Abort on cancellation prior to commit to avoid partial state
	if err := ctx.Err(); err != nil {
		return err, usedSecrets, nil, metrics
	}

	// Commit phase: ensure we are not interrupted mid-commit
	nodeFailures := make(map[string]error)
	commitCtx := context.WithoutCancel(ctx)
	for nodeID, snapshot := range staged {
		if err := snapshotCache.SetSnapshot(commitCtx, nodeID, snapshot); err != nil {
			nodeFailures[nodeID] = err
			errs = append(errs, err)
		}
	}
	for nodeID := range nodeIDsForCleanup {
		if err := snapshotCache.SetSnapshot(commitCtx, nodeID, &cache.Snapshot{}); err != nil {
			nodeFailures[nodeID] = err
			errs = append(errs, err)
		}
	}

	retained := published.retained(allVSs, nodeFailures)
	published.update(allVSs, mixer.nodeIDs, nodeFailures)
	vsStatuses = publicationResults(allVSs, vsStatuses, nil, nodeFailures, retained)
	if len(errs) > 0 && !onlyListenerConflicts(multierr.Combine(errs...)) {
		return multierr.Combine(errs...), usedSecrets, vsStatuses, metrics
	}

	// Update Store domain index after the commit (feature-flagged). A node held by a listener address
	// conflict keeps the domains it serves next to the ones it is about to serve.
	if buildDomainsIndex {
		if len(nodeFailures) > 0 {
			previous := store.GetNodeDomainsIndex()
			for nodeID := range nodeFailures {
				merged := previous[nodeID]
				if merged == nil {
					merged = make(map[string]struct{})
				}
				for key := range nodeDomainsIndex[nodeID] {
					merged[key] = struct{}{}
				}
				nodeDomainsIndex[nodeID] = merged
			}
		}
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

	metrics.snapshotCreateDuration = time.Since(snapshotCreateStart)
	return multierr.Combine(errs...), usedSecrets, vsStatuses, metrics
}

// mergeUsedSecrets returns the secrets used before and now, with the owners seen now.
func mergeUsedSecrets(
	before, now map[helpers.NamespacedName]helpers.NamespacedName,
) map[helpers.NamespacedName]helpers.NamespacedName {
	merged := make(map[helpers.NamespacedName]helpers.NamespacedName, len(before)+len(now))
	for secret, owner := range before {
		merged[secret] = owner
	}
	for secret, owner := range now {
		merged[secret] = owner
	}
	return merged
}

func resourceWarnings(warnings []string) VSStatus {
	if len(warnings) == 0 {
		return VSStatus{}
	}
	messages := slices.Clone(warnings)
	slices.Sort(messages)
	messages = slices.Compact(messages)
	shown := min(len(messages), maxPublicationItems)
	for i := range shown {
		messages[i] = boundedStatusText(messages[i], maxPublicationDetail)
	}
	var suffix string
	if shown < len(messages) {
		suffix = fmt.Sprintf("; %d more warnings", len(messages)-shown)
	}
	detail := boundedStatusText(strings.Join(messages[:shown], "; "), maxPublicationDetail-len(suffix))
	return VSStatus{Message: detail + suffix}
}

// publicationResults reconciles per-service build results with the actual cache writes.
// A failure before commit holds the entire build. A failed cache write only holds its node.
func publicationResults(
	services map[helpers.NamespacedName]*v1alpha1.VirtualService,
	statuses map[helpers.NamespacedName]VSStatus,
	blockerByNamespace map[string]string,
	nodeFailures map[string]error,
	retained map[helpers.NamespacedName][]string,
) map[helpers.NamespacedName]VSStatus {
	buildHeld := blockerByNamespace != nil
	var knownNodes []string
	for nn, vs := range services {
		status := statuses[nn]
		if !buildHeld && len(nodeFailures) == 0 && !status.Invalid {
			continue
		}
		nodes := vs.GetNodeIDs()
		common := isCommonVirtualService(nodes)
		if common {
			if knownNodes == nil {
				knownNodes = publicationNodes(services)
			}
			nodes = knownNodes
		}
		if previous := retained[nn]; len(previous) > 0 {
			nodes = slices.Concat(nodes, previous)
			slices.Sort(nodes)
			nodes = slices.Compact(nodes)
		} else if !common {
			slices.Sort(nodes)
		}
		var reasons []string
		if status.Message != "" {
			reasons = append(reasons, boundedStatusText(status.Message, maxPublicationDetail))
		}
		if buildHeld {
			held := "snapshot publication held"
			if blocker := blockerByNamespace[nn.Namespace]; blocker != "" && !status.Invalid {
				held += " by VirtualService " + blocker
			}
			reasons = append(reasons, held+"; see controller log")
		}
		var affected int
		if (status.Invalid || buildHeld) && len(nodes) > maxPublicationItems {
			affected = len(nodes) - maxPublicationItems
			nodes = nodes[:maxPublicationItems]
		}
		for _, node := range nodes {
			failure := nodeFailures[node]
			if !status.Invalid && !buildHeld && failure == nil {
				continue
			}
			affected++
			if len(status.AffectedNodes) == maxPublicationItems {
				continue
			}
			status.AffectedNodes = append(status.AffectedNodes, node)
			if failure != nil {
				reason := "snapshot update rejected; see controller log"
				var conflict *wrapped.DuplicateListenerAddress
				if errors.As(failure, &conflict) {
					reason = "snapshot update rejected: " + conflict.DetailForNamespace(nn.Namespace)
				}
				reasons = append(reasons, fmt.Sprintf("node %s: %s", boundedStatusText(node, maxDisplayedNodeBytes), reason))
			}
		}
		status.Invalid = status.Invalid || buildHeld || affected > 0
		status.Message = strings.Join(reasons, "; ")
		if affected > len(status.AffectedNodes) {
			status.Message += fmt.Sprintf("; %d more affected nodes", affected-len(status.AffectedNodes))
		}
		statuses[nn] = status
	}
	return statuses
}

func publicationNodes(services map[helpers.NamespacedName]*v1alpha1.VirtualService) []string {
	nodeSet := make(map[string]struct{})
	for _, vs := range services {
		nodes := vs.GetNodeIDs()
		if isCommonVirtualService(nodes) {
			continue
		}
		for _, node := range nodes {
			nodeSet[node] = struct{}{}
		}
	}
	nodes := maps.Keys(nodeSet)
	slices.Sort(nodes)
	return nodes
}

func (c *CacheUpdater) GetUsedSecrets() map[helpers.NamespacedName]helpers.NamespacedName {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return maps.Clone(c.usedSecrets)
}

// GetVirtualServiceWithStatus retrieves a VirtualService with its status applied
// This is used by controllers to get the current status for syncing to Kubernetes
func (c *CacheUpdater) GetVirtualServiceWithStatus(nn helpers.NamespacedName) *v1alpha1.VirtualService {
	c.mx.RLock()
	defer c.mx.RUnlock()

	// Check if OptimizedStore (has status storage)
	if optimizedStore, ok := c.store.(interface {
		GetVirtualServiceWithStatus(helpers.NamespacedName) *v1alpha1.VirtualService
	}); ok {
		return optimizedStore.GetVirtualServiceWithStatus(nn)
	}

	// Fallback for LegacyStore - status is already in VS object
	return c.store.GetVirtualService(nn)
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

func updateSnapshot(
	prevSnapshot cache.ResourceSnapshot,
	resources map[resource.Type][]types.Resource,
) (*cache.Snapshot, bool, error) {
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

// listenerDomain builds the key used to detect domain collisions.
// Domains only have to be unique within a single listener: Envoy resolves an incoming
// connection to a listener first, and only then picks a filter chain inside it. Two
// listeners are independent even when they share a port but differ by address, so the
// listener's namespaced name is the right scope - not the address:port pair.
func listenerDomain(listener helpers.NamespacedName, domain string) string {
	return listener.String() + "|" + domain
}

func nodeIDDomain(nodeID, listenerDomainKey string) string {
	return nodeID + ":" + listenerDomainKey
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
// Duplicate addresses on the nodes in tolerated are skipped.
func validateListenerAddresses(
	storeCopy store.Store,
	snapshotCache *wrapped.SnapshotCache,
	validationIndices bool,
	tolerated map[string]struct{},
) error {
	// Build mapping of listeners to nodeIDs
	listenerToNodeIDs := buildListenerToNodeIDsMapping(storeCopy, snapshotCache)

	if validationIndices {
		// Use the fast index-based approach but check per nodeID
		if err := validateListenerAddressesPerNodeID(storeCopy, listenerToNodeIDs, tolerated); err != nil {
			return err
		}
	}
	// Always run the thorough check to catch structural issues (missing address, etc.)
	if err := validateListenerAddressesUniquePerNodeID(storeCopy, listenerToNodeIDs, tolerated); err != nil {
		return err
	}
	return nil
}

// buildListenerToNodeIDsMapping creates a mapping of listener names to the nodeIDs that use them.
// This is determined by analyzing which VirtualServices reference each listener and what nodeIDs
// those VirtualServices target. For common VirtualServices (nodeID = "*"), expands to all actual nodeIDs.
func buildListenerToNodeIDsMapping(
	storeCopy store.Store,
	snapshotCache *wrapped.SnapshotCache,
) map[helpers.NamespacedName][]string {
	mapping := make(map[helpers.NamespacedName][]string)

	// Iterate through all VirtualServices to find which listeners they use
	for _, vs := range storeCopy.MapVirtualServices() {
		vsNodeIDs := vs.GetNodeIDs()
		if len(vsNodeIDs) == 0 {
			continue // Skip VirtualServices without nodeIDs
		}

		// Get the listener referenced by this VirtualService
		if vs.Spec.Listener != nil && vs.Spec.Listener.Name != "" {
			// The same namespace as the builder uses: spec.listener.namespace, else the one of the service
			listenerNN, err := vs.GetListenerNamespacedName()
			if err != nil {
				continue
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
func validateListenerAddressesPerNodeID(
	storeCopy store.Store,
	listenerToNodeIDs map[helpers.NamespacedName][]string,
	tolerated map[string]struct{},
) error {
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
				if _, ok := tolerated[nodeID]; ok {
					continue
				}
				return fmt.Errorf("listener '%s' has duplicate address '%s' as existing listener '%s' within nodeID '%s'",
					listenerNN.String(), hostPort, existingListener.String(), nodeID)
			}
			addrToListener[hostPort] = listenerNN
		}
	}

	return nil
}

// validateListenerAddressesUniquePerNodeID performs thorough validation of listener addresses per nodeID.
func validateListenerAddressesUniquePerNodeID(
	storeCopy store.Store,
	listenerToNodeIDs map[helpers.NamespacedName][]string,
	tolerated map[string]struct{},
) error {
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
				if _, ok := tolerated[nodeID]; ok {
					continue
				}
				return fmt.Errorf("listener '%s' has duplicate address '%s' as existing listener '%s' within nodeID '%s'",
					listenerNN.String(), hostPort, existingListener.String(), nodeID)
			}
			addrToListener[hostPort] = listenerNN
		}
	}

	return nil
}

// buildExistingDomainsMap builds a map of existing domains per node from either indices or snapshot cache.
func (c *CacheUpdater) buildExistingDomainsMap(
	ctx context.Context,
	targetNodeIDs []string,
	storeCopy store.Store,
	validationIndices bool,
) (map[string]map[string]struct{}, error) {
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
			// Both resource types have to come from the same snapshot, otherwise route
			// configurations could be attributed to listeners of a different generation.
			rcs, listeners, err := c.snapshotCache.GetRouteConfigurationsAndListeners(nodeID)
			if err != nil {
				// If snapshot for nodeID not found yet, we cannot reliably validate against existing domains
				missing++
				continue
			}
			// Route configurations and filter chains are both named after the VirtualService
			// they come from, so the listener owning a filter chain also owns the route
			// configuration with the same name. That gives us the listener scope domains
			// have to be unique within.
			rcListener, err := routeConfigToListener(listeners)
			if err != nil {
				missing++
				continue
			}
			set := make(map[string]struct{})
			for _, rc := range rcs {
				listener, ok := rcListener[rc.GetName()]
				if !ok {
					// Cannot attribute this route configuration to a listener - fall back
					// to the heavy dry-run rather than validating against a wrong scope.
					missing++
					break
				}
				for _, vh := range rc.GetVirtualHosts() {
					for _, dom := range vh.GetDomains() {
						set[listenerDomain(listener, dom)] = struct{}{}
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

// routeConfigToListener maps route configuration names to the listener that serves them.
// Both filter chains and route configurations are named after their VirtualService, so a
// filter chain name is enough to attribute a route configuration to its listener.
// The listeners must come from the same snapshot as the route configurations they are
// matched against.
func routeConfigToListener(listeners []*listenerv3.Listener) (map[string]helpers.NamespacedName, error) {
	result := make(map[string]helpers.NamespacedName, len(listeners))
	for _, l := range listeners {
		nn, err := helpers.NamespacedNameFromString(l.GetName())
		if err != nil {
			return nil, err
		}
		for _, fc := range l.GetFilterChains() {
			// Listeners come from the snapshot in map order, so silently keeping one of
			// two same-named filter chains would vary between runs.
			if previous, ok := result[fc.GetName()]; ok {
				return nil, fmt.Errorf("filter chain %q found on both listener %s and %s",
					fc.GetName(), previous.String(), nn.String())
			}
			result[fc.GetName()] = nn
		}
	}
	return result, nil
}

// excludePreviousVSDomains removes previous VirtualService domains from the existing domains map.
func (c *CacheUpdater) excludePreviousVSDomains(
	ctx context.Context,
	prevVS *v1alpha1.VirtualService,
	targetNodeIDs []string,
	existingByNode map[string]map[string]struct{},
) error {
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
	// Build a quick set of prev domains for faster deletion, scoped to the listener
	// the previous VS was attached to.
	prevDomains := make(map[string]struct{}, len(prevRes.Domains))
	for _, d := range prevRes.Domains {
		prevDomains[listenerDomain(prevRes.Listener, d)] = struct{}{}
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
func checkDomainCollisions(
	candidateDomains []string,
	candidateListener helpers.NamespacedName,
	targetNodeIDs []string,
	existingByNode map[string]map[string]struct{},
) error {
	for _, nodeID := range targetNodeIDs {
		set := existingByNode[nodeID]
		if len(set) == 0 {
			continue
		}
		for _, d := range candidateDomains {
			if _, ok := set[listenerDomain(candidateListener, d)]; ok {
				return fmt.Errorf("duplicate domain '%s' for node %s on listener %s",
					d, nodeID, candidateListener.String())
			}
		}
	}
	return nil
}
