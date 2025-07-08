package store

import (
	"maps"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
)

func (s *Store) SetVirtualService(vs *v1alpha1.VirtualService) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.virtualServices[helpers.NamespacedName{Namespace: vs.Namespace, Name: vs.Name}] = vs
	s.updateVirtualServiceByUIDMap()
}

func (s *Store) GetVirtualService(name helpers.NamespacedName) *v1alpha1.VirtualService {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vs := s.virtualServices[name]
	return vs
}

func (s *Store) DeleteVirtualService(name helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.virtualServices, name)
	s.updateVirtualServiceByUIDMap()
}

func (s *Store) IsExistingVirtualService(name helpers.NamespacedName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.virtualServices[name]
	return ok
}

func (s *Store) MapVirtualServices() map[helpers.NamespacedName]*v1alpha1.VirtualService {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.virtualServices)
}

func (s *Store) updateVirtualServiceByUIDMap() {
	if len(s.virtualServices) == 0 {
		return
	}
	m := make(map[string]*v1alpha1.VirtualService, len(s.virtualServices))
	for _, vs := range s.virtualServices {
		m[string(vs.UID)] = vs
	}
	s.virtualServiceByUID = m
}

func (s *Store) GetVirtualServiceByUID(uid string) *v1alpha1.VirtualService {
	s.mu.RLock()
	defer s.mu.RUnlock()
	vs := s.virtualServiceByUID[uid]
	if vs == nil {
		return nil
	}
	vs = vs.DeepCopy()
	return vs
}
