package store

import (
	"maps"
	"strings"

	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	corev1 "k8s.io/api/core/v1"
)

func (s *Store) SetSecret(secret *corev1.Secret) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secrets[helpers.NamespacedName{Namespace: secret.Namespace, Name: secret.Name}] = secret
	s.updateDomainSecretsMap()
}

func (s *Store) GetSecret(name helpers.NamespacedName) *corev1.Secret {
	s.mu.RLock()
	defer s.mu.RUnlock()
	secret := s.secrets[name]
	return secret
}

func (s *Store) DeleteSecret(name helpers.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.secrets, name)
	s.updateDomainSecretsMap()
}

func (s *Store) IsExistingSecret(name helpers.NamespacedName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.secrets[name]
	return ok
}

func (s *Store) MapSecrets() map[helpers.NamespacedName]*corev1.Secret {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.secrets)
}

func (s *Store) MapDomainSecrets() map[string]corev1.Secret {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.domainToSecretMap)
}

func (s *Store) updateDomainSecretsMap() {
	m := make(map[string]corev1.Secret)

	for _, secret := range s.secrets {
		for _, domain := range strings.Split(secret.Annotations[v1alpha1.AnnotationSecretDomains], ",") {
			domain = strings.TrimSpace(domain)
			if domain == "" {
				continue
			}
			if _, ok := m[domain]; ok {
				// TODO domain already exist in another secret! Need create error case
				continue
			}
			m[domain] = *secret
		}
	}
	s.domainToSecretMap = m
}
