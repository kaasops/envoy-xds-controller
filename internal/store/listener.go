package store

import (
	"maps"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
)

func (s *LegacyStore) SetListener(l *v1alpha1.Listener) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners[helpers.NamespacedName{Namespace: l.Namespace, Name: l.Name}] = l
	s.updateListenerByUIDMap()
}

func (s *LegacyStore) GetListener(name helpers.NamespacedName) *v1alpha1.Listener {
	s.mu.RLock()
	defer s.mu.RUnlock()
	l := s.listeners[name]
	return l
}

func (s *LegacyStore) DeleteListener(name helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.listeners, name)
	s.updateListenerByUIDMap()
}

func (s *LegacyStore) IsExistingListener(name helpers.NamespacedName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.listeners[name]
	return ok
}

func (s *LegacyStore) MapListeners() map[helpers.NamespacedName]*v1alpha1.Listener {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.listeners)
}

func (s *LegacyStore) updateListenerByUIDMap() {
	if len(s.listeners) == 0 {
		return
	}
	m := make(map[string]*v1alpha1.Listener, len(s.listeners))
	for _, l := range s.listeners {
		m[string(l.UID)] = l
	}
	s.listenerByUID = m
}

func (s *LegacyStore) GetListenerByUID(uid string) *v1alpha1.Listener {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.listenerByUID[uid]
}
