package store

import (
	"fmt"
)

// listener address index and duplicate tracking
// guarded via updateListenerAddressIndex and read via GetListenerAddressDuplicate.
// This enables O(1) duplicate detection in light validators.

type listenerAddrDup struct {
	Address string
	First   string // first listener (namespaced name string)
	Second  string // duplicate listener (namespaced name string)
}

func (s *Store) updateListenerAddressIndex() {
	if s == nil {
		return
	}
	// Build address -> first listener mapping and capture the first duplicate (if any)
	addr2name := make(map[string]string, len(s.listeners))
	var dup *listenerAddrDup
	for nn, l := range s.listeners {
		lv3, err := l.UnmarshalV3()
		if err != nil {
			// On malformed listener spec, skip indexing to avoid panics; validation path will catch it later
			continue
		}
		addr := lv3.GetAddress()
		if addr == nil || addr.GetSocketAddress() == nil {
			// Incomplete address - skip; validation will handle the error with precise message if needed
			continue
		}
		host := addr.GetSocketAddress().GetAddress()
		port := addr.GetSocketAddress().GetPortValue()
		hostPort := fmt.Sprintf("%s:%d", host, port)
		if exist, ok := addr2name[hostPort]; ok && dup == nil {
			// record the first duplicate encountered
			dup = &listenerAddrDup{
				Address: hostPort,
				First:   exist,
				Second:  nn.String(),
			}
		}
		addr2name[hostPort] = nn.String()
	}
	s.listenerAddrIndex = addr2name
	s.listenerAddrDup = dup
}

// GetListenerAddressDuplicate returns the first detected duplicate listener address, if any.
// addr is host:port, first is existing listener name, second is conflicting listener name.
func (s *Store) GetListenerAddressDuplicate() (addr, first, second string, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.listenerAddrDup == nil {
		return "", "", "", false
	}
	return s.listenerAddrDup.Address, s.listenerAddrDup.First, s.listenerAddrDup.Second, true
}

// GetListenerAddressIndex returns a shallow copy of the address->listener map for diagnostics.
func (s *Store) GetListenerAddressIndex() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make(map[string]string, len(s.listenerAddrIndex))
	for k, v := range s.listenerAddrIndex {
		res[k] = v
	}
	return res
}

// --- Node domains index (nodeID -> set(domains)) ---

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
