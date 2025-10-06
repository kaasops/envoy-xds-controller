package store

import (
	"sync"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
)

// VirtualServiceStatus represents the validation/processing status of a VirtualService
type VirtualServiceStatus struct {
	Invalid bool
	Message string
}

// StatusStorage stores VirtualService statuses separately from the VS objects themselves.
// This allows immutable VS objects while still tracking mutable status information.
type StatusStorage struct {
	mu       sync.RWMutex
	statuses map[helpers.NamespacedName]VirtualServiceStatus
}

// NewStatusStorage creates a new status storage
func NewStatusStorage() *StatusStorage {
	return &StatusStorage{
		statuses: make(map[helpers.NamespacedName]VirtualServiceStatus),
	}
}

// SetStatus sets the status for a VirtualService
func (s *StatusStorage) SetStatus(nn helpers.NamespacedName, invalid bool, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statuses[nn] = VirtualServiceStatus{
		Invalid: invalid,
		Message: message,
	}
}

// GetStatus retrieves the status for a VirtualService
// Returns zero value if status not found (Invalid=false, Message="")
func (s *StatusStorage) GetStatus(nn helpers.NamespacedName) VirtualServiceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.statuses[nn]
}

// ApplyStatusToVS applies the stored status to a VirtualService object
// This is used when we need to sync status to Kubernetes
func (s *StatusStorage) ApplyStatusToVS(vs *v1alpha1.VirtualService) {
	if vs == nil {
		return
	}
	nn := helpers.NamespacedName{Namespace: vs.Namespace, Name: vs.Name}
	status := s.GetStatus(nn)
	vs.Status.Invalid = status.Invalid
	vs.Status.Message = status.Message
}

// DeleteStatus removes status for a VirtualService
func (s *StatusStorage) DeleteStatus(nn helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.statuses, nn)
}

// SetStatuses sets multiple statuses at once
func (s *StatusStorage) SetStatuses(statuses map[helpers.NamespacedName]VirtualServiceStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for nn, status := range statuses {
		s.statuses[nn] = status
	}
}

// GetAllStatuses returns a copy of all statuses
func (s *StatusStorage) GetAllStatuses() map[helpers.NamespacedName]VirtualServiceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[helpers.NamespacedName]VirtualServiceStatus, len(s.statuses))
	for k, v := range s.statuses {
		result[k] = v
	}
	return result
}

// Copy creates a shallow copy of the status storage
// Statuses are value types (struct with bool+string), so shallow copy is safe
func (s *StatusStorage) Copy() *StatusStorage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	newStorage := &StatusStorage{
		statuses: make(map[helpers.NamespacedName]VirtualServiceStatus, len(s.statuses)),
	}

	for k, v := range s.statuses {
		newStorage.statuses[k] = v
	}

	return newStorage
}
