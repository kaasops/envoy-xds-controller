package store

import (
	"maps"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
)

func (s *LegacyStore) SetCluster(c *v1alpha1.Cluster) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clusters[helpers.NamespacedName{Namespace: c.Namespace, Name: c.Name}] = c
	s.updateSpecClusters()
}

func (s *LegacyStore) GetCluster(name helpers.NamespacedName) *v1alpha1.Cluster {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c := s.clusters[name]
	return c
}

func (s *LegacyStore) DeleteCluster(name helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clusters, name)
	s.updateSpecClusters()
}

func (s *LegacyStore) IsExistingCluster(name helpers.NamespacedName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.clusters[name]
	return ok
}

func (s *LegacyStore) MapClusters() map[helpers.NamespacedName]*v1alpha1.Cluster {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.clusters)
}

func (s *LegacyStore) GetSpecCluster(name string) *v1alpha1.Cluster {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.specClusters[name]
}

func (s *LegacyStore) MapSpecClusters() map[string]*v1alpha1.Cluster {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.specClusters)
}

func (s *LegacyStore) updateSpecClusters() {
	m := make(map[string]*v1alpha1.Cluster)

	for _, cluster := range s.clusters {
		clusterV3, _ := cluster.UnmarshalV3()
		m[clusterV3.Name] = cluster
	}

	s.specClusters = m
}

func (s *LegacyStore) updateClusterByUIDMap() {
	if len(s.clusters) == 0 {
		return
	}
	m := make(map[string]*v1alpha1.Cluster, len(s.clusters))
	for _, cl := range s.clusters {
		m[string(cl.UID)] = cl
	}
	s.clusterByUID = m
}
