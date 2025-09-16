package store

// ReplaceNodeDomainsIndex replaces the nodeDomainsIndex with a deep copy of the provided index.
// Intended to be called by the updater after a successful snapshot rebuild.
func (s *Store) ReplaceNodeDomainsIndex(idx map[string]map[string]struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if idx == nil {
		s.nodeDomainsIndex = nil
		return
	}
	copyIdx := make(map[string]map[string]struct{}, len(idx))
	for node, set := range idx {
		inner := make(map[string]struct{}, len(set))
		for d := range set {
			inner[d] = struct{}{}
		}
		copyIdx[node] = inner
	}
	s.nodeDomainsIndex = copyIdx
}

// GetNodeDomainsIndex returns a deep copy of the nodeDomainsIndex.
func (s *Store) GetNodeDomainsIndex() map[string]map[string]struct{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.nodeDomainsIndex == nil {
		return nil
	}
	res := make(map[string]map[string]struct{}, len(s.nodeDomainsIndex))
	for node, set := range s.nodeDomainsIndex {
		inner := make(map[string]struct{}, len(set))
		for d := range set {
			inner[d] = struct{}{}
		}
		res[node] = inner
	}
	return res
}

// GetNodeDomainsForNodes returns a deep copy of the domain sets for the provided nodeIDs
// and a list of nodeIDs missing in the index.
func (s *Store) GetNodeDomainsForNodes(nodeIDs []string) (map[string]map[string]struct{}, []string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make(map[string]map[string]struct{}, len(nodeIDs))
	missing := make([]string, 0)
	for _, n := range nodeIDs {
		set, ok := s.nodeDomainsIndex[n]
		if !ok {
			missing = append(missing, n)
			continue
		}
		inner := make(map[string]struct{}, len(set))
		for d := range set {
			inner[d] = struct{}{}
		}
		res[n] = inner
	}
	return res, missing
}
