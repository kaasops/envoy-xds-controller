package store

import (
	"maps"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
)

func (s *Store) SetAccessLog(a *v1alpha1.AccessLogConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accessLogs[helpers.NamespacedName{Namespace: a.Namespace, Name: a.Name}] = a
	s.updateAccessLogByUIDMap()
}

func (s *Store) GetAccessLog(name helpers.NamespacedName) *v1alpha1.AccessLogConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a := s.accessLogs[name]
	return a
}

func (s *Store) DeleteAccessLog(name helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.accessLogs, name)
	s.updateAccessLogByUIDMap()
}

func (s *Store) IsExistingAccessLog(name helpers.NamespacedName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.accessLogs[name]
	return ok
}

func (s *Store) MapAccessLogs() map[helpers.NamespacedName]*v1alpha1.AccessLogConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.accessLogs)
}

func (s *Store) updateAccessLogByUIDMap() {
	if len(s.accessLogs) == 0 {
		return
	}
	m := make(map[string]*v1alpha1.AccessLogConfig, len(s.accessLogs))
	for _, al := range s.accessLogs {
		m[string(al.UID)] = al
	}
	s.accessLogByUID = m
}

func (s *Store) GetAccessLogByUID(uid string) *v1alpha1.AccessLogConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.accessLogByUID[uid]
}
