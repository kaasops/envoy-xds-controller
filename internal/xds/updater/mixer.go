package updater

import (
	"sort"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/store"
)

type Mixer struct {
	listeners map[helpers.NamespacedName]map[string][]*listenerv3.FilterChain
	data      map[string]map[resource.Type][]types.Resource
	nodeIDs   map[string]struct{}
}

func NewMixer() *Mixer {
	return &Mixer{
		data:      make(map[string]map[resource.Type][]types.Resource),
		listeners: make(map[helpers.NamespacedName]map[string][]*listenerv3.FilterChain),
		nodeIDs:   make(map[string]struct{}),
	}
}

func (m *Mixer) Add(nodeID string, resourceType resource.Type, resource types.Resource) {
	if resources, ok := m.data[nodeID]; ok {
		resources[resourceType] = append(resources[resourceType], resource)
	} else {
		m.data[nodeID] = map[string][]types.Resource{
			resourceType: {resource},
		}
		m.nodeIDs[nodeID] = struct{}{}
	}
}

func (m *Mixer) AddListenerParams(
	listenerNamespacedName helpers.NamespacedName,
	fcs []*listenerv3.FilterChain,
	nodeID string,
) {
	if m.listeners[listenerNamespacedName] == nil {
		m.listeners[listenerNamespacedName] = make(map[string][]*listenerv3.FilterChain)
	}
	m.listeners[listenerNamespacedName][nodeID] = append(m.listeners[listenerNamespacedName][nodeID], fcs...)
	m.nodeIDs[nodeID] = struct{}{}
}

func (m *Mixer) Mix(store store.Store) (map[string]map[resource.Type][]types.Resource, error) {
	result := make(map[string]map[resource.Type][]types.Resource)

	for listenerNamespacedName, data := range m.listeners {
		listener := store.GetListener(listenerNamespacedName)
		for nodeID, fcs := range data {
			lv3, err := listener.UnmarshalV3()
			if err != nil {
				return nil, err
			}
			// Sort filter chains by name to ensure deterministic order.
			// This prevents spurious snapshot version increments caused by
			// non-deterministic map iteration order in Go.
			sortFilterChains(fcs)
			lv3.FilterChains = fcs
			lv3.Name = listenerNamespacedName.String()
			if resources, ok := result[nodeID]; ok {
				result[nodeID][resource.ListenerType] = append(resources[resource.ListenerType], lv3)
			} else {
				result[nodeID] = map[resource.Type][]types.Resource{
					resource.ListenerType: {lv3},
				}
			}
		}
	}

	for nodeID, resources := range m.data {
		if result[nodeID] == nil {
			result[nodeID] = make(map[resource.Type][]types.Resource)
		}
		result[nodeID][resource.SecretType] = resources[resource.SecretType]
		result[nodeID][resource.ClusterType] = resources[resource.ClusterType]
		result[nodeID][resource.RouteType] = resources[resource.RouteType]
	}

	// Sort all resources by name to ensure deterministic order.
	// This prevents spurious snapshot version increments caused by
	// non-deterministic map iteration order in Go.
	for nodeID := range result {
		sortResources(result[nodeID][resource.ListenerType])
		sortResources(result[nodeID][resource.ClusterType])
		sortResources(result[nodeID][resource.SecretType])
		sortResources(result[nodeID][resource.RouteType])
	}

	return result, nil
}

// sortFilterChains sorts filter chains by name for deterministic ordering.
// This prevents spurious snapshot version increments caused by
// non-deterministic map iteration order in Go.
// Note: Filter chain names should be unique; duplicates are a configuration error.
func sortFilterChains(fcs []*listenerv3.FilterChain) {
	if len(fcs) < 2 {
		return
	}
	sort.Slice(fcs, func(i, j int) bool {
		return fcs[i].GetName() < fcs[j].GetName()
	})
}

// sortResources sorts xDS resources by name for deterministic ordering.
// Uses cachev3.GetResourceName which uses type assertion (fast) instead of reflection.
// This prevents spurious snapshot version increments caused by
// non-deterministic map iteration order in Go.
// Note: Resource names should be unique within their type; duplicates are a configuration error.
func sortResources(resources []types.Resource) {
	if len(resources) < 2 {
		return
	}
	sort.Slice(resources, func(i, j int) bool {
		return cachev3.GetResourceName(resources[i]) < cachev3.GetResourceName(resources[j])
	})
}
