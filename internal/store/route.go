package store

import (
	"maps"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
)

func (s *Store) SetRoute(r *v1alpha1.Route) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.routes[helpers.NamespacedName{Namespace: r.Namespace, Name: r.Name}] = r
	s.updateRouteByUIDMap()
}

func (s *Store) GetRoute(name helpers.NamespacedName) *v1alpha1.Route {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r := s.routes[name]
	return r
}

func (s *Store) DeleteRoute(name helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.routes, name)
	s.updateRouteByUIDMap()
}

func (s *Store) IsExistingRoute(name helpers.NamespacedName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.routes[name]
	return ok
}

func (s *Store) MapRoutes() map[helpers.NamespacedName]*v1alpha1.Route {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.routes)
}

func (s *Store) updateRouteByUIDMap() {
	if len(s.routes) == 0 {
		return
	}
	m := make(map[string]*v1alpha1.Route, len(s.routes))
	for _, r := range s.routes {
		m[string(r.UID)] = r
	}
	s.routeByUID = m
}

func (s *Store) GetRouteByUID(uid string) *v1alpha1.Route {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.routeByUID[uid]
}
