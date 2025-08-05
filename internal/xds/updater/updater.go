package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"

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

func NewCacheUpdater(wsc *wrapped.SnapshotCache, store *store.Store) *CacheUpdater {
	return &CacheUpdater{
		snapshotCache: wsc,
		usedSecrets:   make(map[helpers.NamespacedName]helpers.NamespacedName),
		store:         store,
	}
}

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

func (c *CacheUpdater) rebuildSnapshots(ctx context.Context) error {
	err, usedSecrets := buildSnapshots(ctx, c.snapshotCache, c.store)
	c.usedSecrets = usedSecrets
	return err
}

// nolint: gocyclo
func buildSnapshots(
	ctx context.Context,
	snapshotCache *wrapped.SnapshotCache,
	store *store.Store,
) (error, map[helpers.NamespacedName]helpers.NamespacedName) {
	errs := make([]error, 0)

	mixer := NewMixer()

	// ---------------------------------------------

	usedSecrets := make(map[helpers.NamespacedName]helpers.NamespacedName)
	nodeIDDomainsSet := make(map[string]struct{})

	nodeIDsForCleanup := snapshotCache.GetNodeIDsAsMap()
	var commonVirtualServices []*v1alpha1.VirtualService

	for _, vs := range store.MapVirtualServices() {
		if vs.IsStatusInvalid() {
			continue
		}

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

		vsRes, err := resbuilder.BuildResources(vs, store)
		if err != nil {
			vs.UpdateStatus(true, err.Error())
			errs = append(errs, err)
			continue
		}
		vs.UpdateStatus(false, "")

		for _, secret := range vsRes.UsedSecrets {
			usedSecrets[secret] = helpers.NamespacedName{Name: vs.Name, Namespace: vs.Namespace}
		}

		for _, nodeID := range vsNodeIDs {

			for _, domain := range vsRes.Domains {
				nodeDom := nodeIDDomain(nodeID, domain)
				if _, ok := nodeIDDomainsSet[nodeDom]; ok {
					return fmt.Errorf("duplicate domain %s for node %s", domain, nodeID), nil
				}
				nodeIDDomainsSet[nodeDom] = struct{}{}
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
			vsRes, err := resbuilder.BuildResources(vs, store)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			for _, secret := range vsRes.UsedSecrets {
				usedSecrets[secret] = helpers.NamespacedName{Name: vs.Name, Namespace: vs.Namespace}
			}

			for nodeID := range mixer.nodeIDs {

				for _, domain := range vsRes.Domains {
					nodeDom := nodeIDDomain(nodeID, domain)
					if _, ok := nodeIDDomainsSet[nodeDom]; ok {
						return fmt.Errorf("duplicate domain %s for node %s", domain, nodeID), nil
					}
					nodeIDDomainsSet[nodeDom] = struct{}{}
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

	tmp, err := mixer.Mix(store)
	if err != nil {
		errs = append(errs, err)
		return multierr.Combine(errs...), usedSecrets
	}

	for nodeID, resMap := range tmp {
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
			err = snapshotCache.SetSnapshot(ctx, nodeID, snapshot)
			if err != nil {
				errs = append(errs, err)
				continue
			}
		}
		delete(nodeIDsForCleanup, nodeID)
	}

	for nodeID := range nodeIDsForCleanup {
		err = snapshotCache.SetSnapshot(ctx, nodeID, &cache.Snapshot{})
		if err != nil {
			errs = append(errs, err)
			continue
		}
	}

	if len(errs) > 0 {
		return multierr.Combine(errs...), usedSecrets
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

func nodeIDDomain(nodeID, domain string) string {
	return nodeID + ":" + domain
}
