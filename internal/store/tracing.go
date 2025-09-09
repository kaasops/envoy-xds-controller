package store

import (
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
)

func (s *Store) SetTracing(t *v1alpha1.Tracing) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tracings[helpers.NamespacedName{Namespace: t.Namespace, Name: t.Name}] = t
}

func (s *Store) GetTracing(name helpers.NamespacedName) *v1alpha1.Tracing {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t := s.tracings[name]
	return t
}

func (s *Store) DeleteTracing(name helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tracings, name)
}

func (s *Store) IsExistingTracing(name helpers.NamespacedName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.tracings[name]
	return ok
}
