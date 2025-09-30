package store

import (
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
)

func (s *LegacyStore) SetTracing(t *v1alpha1.Tracing) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tracings[helpers.NamespacedName{Namespace: t.Namespace, Name: t.Name}] = t
}

func (s *LegacyStore) GetTracing(name helpers.NamespacedName) *v1alpha1.Tracing {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t := s.tracings[name]
	return t
}

func (s *LegacyStore) DeleteTracing(name helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tracings, name)
}

func (s *LegacyStore) IsExistingTracing(name helpers.NamespacedName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.tracings[name]
	return ok
}

func (s *LegacyStore) MapTracings() map[helpers.NamespacedName]*v1alpha1.Tracing {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tracings
}
