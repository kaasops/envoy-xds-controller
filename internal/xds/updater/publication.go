package updater

import (
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"k8s.io/apimachinery/pkg/types"
)

// Status is a per-object diagnostic. The controller log retains the complete build errors.
const (
	maxPublicationDetail  = 2048
	maxPublicationMessage = 4096
	maxPublicationItems   = 8
	maxDisplayedNodeBytes = 96
)

func boundedStatusText(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	end := limit - len("...")
	for end > 0 && !utf8.RuneStart(text[end]) {
		end--
	}
	return text[:end] + "..."
}

type publicationObservation struct {
	uid        types.UID
	generation int64
	invalid    bool
	message    string
}

type publishedService struct {
	name helpers.NamespacedName
	uid  types.UID
}

type nodeContributors struct {
	explicit []publishedService
	common   *publishedCommonServices
}

type publishedCommonServices struct{ services []publishedService }

// A rejected write leaves the previous node snapshot and its contributors in place.
// Common services share one slice across all nodes published by the same build.
type publishedContributors map[string]nodeContributors

func (p publishedContributors) retained(
	services map[helpers.NamespacedName]*v1alpha1.VirtualService, failures map[string]error,
) map[helpers.NamespacedName][]string {
	if len(p) == 0 || (failures != nil && len(failures) == 0) {
		return nil
	}
	retained := make(map[helpers.NamespacedName][]string)
	var activeNodes []string
	movedCommon := make(map[*publishedCommonServices][]publishedService)
	collect := func(node string) {
		owners := p[node]
		var shared []publishedService
		if owners.common != nil {
			if activeNodes == nil {
				activeNodes = publicationNodes(services)
			}
			if _, active := slices.BinarySearch(activeNodes, node); active {
				// Unchanged wildcard targets are already covered by publicationResults.
				// Check each shared cohort once even when an entire build is held.
				var seen bool
				if shared, seen = movedCommon[owners.common]; !seen {
					for _, owner := range owners.common.services {
						if vs := services[owner.name]; vs != nil && !isCommonVirtualService(vs.GetNodeIDs()) {
							shared = append(shared, owner)
						}
					}
					movedCommon[owners.common] = shared
				}
			} else {
				shared = owners.common.services
			}
		}
		for _, list := range [][]publishedService{owners.explicit, shared} {
			for _, owner := range list {
				if vs := services[owner.name]; vs != nil && vs.UID == owner.uid {
					retained[owner.name] = append(retained[owner.name], node)
				}
			}
		}
	}
	if failures == nil {
		for node := range p {
			collect(node)
		}
	} else {
		for node := range failures {
			collect(node)
		}
	}
	return retained
}

func (p publishedContributors) update(
	services map[helpers.NamespacedName]*v1alpha1.VirtualService, nodes map[string]struct{}, failures map[string]error,
) {
	if p == nil {
		return
	}
	explicit := make(map[string][]publishedService, len(nodes))
	common := &publishedCommonServices{}
	for key, vs := range services {
		owner := publishedService{name: key, uid: vs.UID}
		targets := vs.GetNodeIDs()
		if isCommonVirtualService(targets) {
			common.services = append(common.services, owner)
			continue
		}
		for _, node := range targets {
			if failures[node] == nil {
				explicit[node] = append(explicit[node], owner)
			}
		}
	}
	if len(common.services) == 0 {
		common = nil
	}
	for node := range p {
		if _, active := nodes[node]; !active && failures[node] == nil {
			delete(p, node)
		}
	}
	for node := range nodes {
		if failures[node] == nil {
			p[node] = nodeContributors{explicit: explicit[node], common: common}
		}
	}
}

func (c *CacheUpdater) recordPublication(key helpers.NamespacedName, status VSStatus) {
	vs := c.store.GetVirtualService(key)
	if vs == nil {
		return
	}
	message := c.publicationMessage(status)
	if store, ok := c.store.(interface {
		SetVirtualServiceStatus(helpers.NamespacedName, bool, string)
	}); ok {
		store.SetVirtualServiceStatus(key, status.Invalid, message)
	} else {
		copy := vs.DeepCopy()
		copy.UpdateStatus(status.Invalid, message)
		c.store.SetVirtualService(copy)
	}
	next := publicationObservation{uid: vs.UID, generation: vs.Generation, invalid: status.Invalid, message: message}
	if previous, exists := c.publications[key]; exists && previous == next {
		return
	}
	c.publications[key] = next
	if c.enqueueStatus != nil {
		c.enqueueStatus(key)
	}
}

// WatchPublicationStatus registers the controller work queue. Its Add operation
// must not block or call back into the updater while a build holds the lock.
func (c *CacheUpdater) WatchPublicationStatus(enqueue func(helpers.NamespacedName)) {
	c.mx.Lock()
	defer c.mx.Unlock()
	c.enqueueStatus = enqueue
	for key := range c.publications {
		enqueue(key)
	}
}

// PublicationStatusFor ignores a result for a deleted or older incarnation.
func (c *CacheUpdater) PublicationStatusFor(vs *v1alpha1.VirtualService) (VSStatus, bool) {
	c.mx.RLock()
	defer c.mx.RUnlock()
	result, ok := c.publications[helpers.NamespacedName{Namespace: vs.Namespace, Name: vs.Name}]
	if !ok || result.uid != vs.UID || result.generation != vs.Generation {
		return VSStatus{}, false
	}
	return VSStatus{Invalid: result.invalid, Message: result.message}, true
}

func (c *CacheUpdater) publicationMessage(status VSStatus) string {
	if !status.Invalid || len(status.AffectedNodes) == 0 {
		return boundedStatusText(status.Message, maxPublicationMessage)
	}
	var message strings.Builder
	message.WriteString(status.Message)
	message.WriteString("; last snapshot publication: ")
	for i, node := range status.AffectedNodes {
		if i != 0 {
			message.WriteString(", ")
		}
		last := c.snapshotCache.LastPublication(node)
		name := boundedStatusText(node, maxDisplayedNodeBytes)
		if last.IsZero() {
			fmt.Fprintf(&message, "%s=none since startup", name)
		} else {
			fmt.Fprintf(&message, "%s=%s", name, last.Format(time.RFC3339))
		}
	}
	return boundedStatusText(message.String(), maxPublicationMessage)
}
