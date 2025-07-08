package store

import (
	"maps"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
)

func (s *Store) SetCluster(c *v1alpha1.Cluster) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clusters[helpers.NamespacedName{Namespace: c.Namespace, Name: c.Name}] = c
	s.updateSpecClusters()
}

func (s *Store) GetCluster(name helpers.NamespacedName) *v1alpha1.Cluster {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c := s.clusters[name]
	return c
}

func (s *Store) DeleteCluster(name helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clusters, name)
	s.updateSpecClusters()
}

func (s *Store) IsExistingCluster(name helpers.NamespacedName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.clusters[name]
	return ok
}

func (s *Store) MapClusters() map[helpers.NamespacedName]*v1alpha1.Cluster {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.clusters)
}

func (s *Store) GetSpecCluster(name string) *v1alpha1.Cluster {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.specClusters[name]
}

func (s *Store) MapSpecClusters() map[string]*v1alpha1.Cluster {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.specClusters)
}

func (s *Store) updateSpecClusters() {
	m := make(map[string]*v1alpha1.Cluster)

	for _, cluster := range s.clusters {
		clusterV3, _ := cluster.UnmarshalV3()
		m[clusterV3.Name] = cluster
	}

	s.specClusters = m
}

func (s *Store) updateClusterByUIDMap() {
	if len(s.clusters) == 0 {
		return
	}
	m := make(map[string]*v1alpha1.Cluster, len(s.clusters))
	for _, cl := range s.clusters {
		m[string(cl.UID)] = cl
	}
	s.clusterByUID = m
}
