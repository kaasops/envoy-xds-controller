package store

import (
	"maps"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
)

func (s *LegacyStore) SetPolicy(p *v1alpha1.Policy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies[helpers.NamespacedName{Namespace: p.Namespace, Name: p.Name}] = p
}

func (s *LegacyStore) GetPolicy(name helpers.NamespacedName) *v1alpha1.Policy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p := s.policies[name]
	return p
}

func (s *LegacyStore) DeletePolicy(name helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.policies, name)
}

func (s *LegacyStore) IsExistingPolicy(name helpers.NamespacedName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.policies[name]
	return ok
}

func (s *LegacyStore) MapPolicies() map[helpers.NamespacedName]*v1alpha1.Policy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.policies)
}
