package clients

import (
	"sort"
	"sync"
)

type Registry struct {
	mu   sync.RWMutex
	data map[int64]*Info
}

func NewRegistry() *Registry {
	return &Registry{
		data: make(map[int64]*Info),
	}
}

func (r *Registry) Set(id int64, ci *Info) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[id] = ci
}

func (r *Registry) Update(id int64, ci *Info) {
	r.mu.Lock()
	defer r.mu.Unlock()
	prev, ok := r.data[id]
	if !ok {
		r.data[id] = ci
		return
	}
	prev.Version = ci.Version
	prev.NodeID = ci.NodeID
}

func (r *Registry) Delete(id int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.data, id)
}

func (r *Registry) List() []Info {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Info, 0, len(r.data))
	for _, c := range r.data {
		result = append(result, *c)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}
