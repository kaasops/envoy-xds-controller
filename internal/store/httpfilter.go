package store

import (
	"maps"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
)

func (s *Store) SetHTTPFilter(hf *v1alpha1.HttpFilter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.httpFilters[helpers.NamespacedName{Namespace: hf.Namespace, Name: hf.Name}] = hf
	s.updateHTTPFilterByUIDMap()
}

func (s *Store) GetHTTPFilter(name helpers.NamespacedName) *v1alpha1.HttpFilter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hf := s.httpFilters[name]
	return hf
}

func (s *Store) DeleteHTTPFilter(name helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.httpFilters, name)
	s.updateHTTPFilterByUIDMap()
}

func (s *Store) IsExistingHTTPFilter(name helpers.NamespacedName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.httpFilters[name]
	return ok
}

func (s *Store) MapHTTPFilters() map[helpers.NamespacedName]*v1alpha1.HttpFilter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.httpFilters)
}

func (s *Store) updateHTTPFilterByUIDMap() {
	if len(s.httpFilters) == 0 {
		return
	}
	m := make(map[string]*v1alpha1.HttpFilter, len(s.httpFilters))
	for _, hf := range s.httpFilters {
		m[string(hf.UID)] = hf
	}
	s.httpFilterByUID = m
}

func (s *Store) GetHTTPFilterByUID(uid string) *v1alpha1.HttpFilter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hf := s.httpFilterByUID[uid]
	return hf
}
