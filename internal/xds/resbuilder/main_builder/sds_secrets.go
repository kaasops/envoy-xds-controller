package main_builder

import (
	"fmt"
	"strings"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// buildSDSSecrets builds the secrets that HTTP filters request over SDS,
// for example the token and hmac secrets of the OAuth2 filter.
// A secret is referenced as namespace/name for a TLS secret
// or as namespace/name/key for a single key of an Opaque secret.
// A reference that does not resolve is logged and skipped: the VirtualService
// is still served and Envoy keeps waiting for the secret.
func (b *Builder) buildSDSSecrets(
	vsName helpers.NamespacedName,
	httpFilters []*hcmv3.HttpFilter,
) ([]*tlsv3.Secret, []helpers.NamespacedName) {
	var refs []*tlsv3.SdsSecretConfig
	for _, httpFilter := range httpFilters {
		collectSDSSecretRefs(httpFilter.ProtoReflect(), &refs)
	}

	var secrets []*tlsv3.Secret
	var usedSecrets []helpers.NamespacedName
	seenNames := make(map[string]struct{}, len(refs))
	seenSecrets := make(map[helpers.NamespacedName]struct{}, len(refs))

	for _, ref := range refs {
		name := ref.GetName()
		if _, ok := seenNames[name]; ok {
			continue
		}
		seenNames[name] = struct{}{}

		secretNN, secret, err := b.buildSDSSecret(name)
		if err != nil {
			log.Log.WithName("main-builder").Info("SDS secret requested by HTTP filter is not served",
				"virtualService", vsName.String(), "secret", name, "reason", err.Error())
			continue
		}
		secrets = append(secrets, secret)

		if _, ok := seenSecrets[secretNN]; !ok {
			seenSecrets[secretNN] = struct{}{}
			usedSecrets = append(usedSecrets, secretNN)
		}
	}

	return secrets, usedSecrets
}

// buildSDSSecret builds a secret by the name an HTTP filter requests it with
func (b *Builder) buildSDSSecret(name string) (helpers.NamespacedName, *tlsv3.Secret, error) {
	parts := strings.Split(name, "/")
	if len(parts) != 2 && len(parts) != 3 {
		return helpers.NamespacedName{}, nil,
			fmt.Errorf("invalid SDS secret name %q: expected namespace/name or namespace/name/key", name)
	}
	secretNN := helpers.NamespacedName{Namespace: parts[0], Name: parts[1]}

	if len(parts) == 2 {
		secret, err := b.buildSecret(secretNN)
		return secretNN, secret, err
	}
	secret, err := b.buildGenericSecret(secretNN, parts[2])
	return secretNN, secret, err
}

// buildGenericSecret builds a generic secret from one key of an Opaque Kubernetes secret
func (b *Builder) buildGenericSecret(secretName helpers.NamespacedName, key string) (*tlsv3.Secret, error) {
	k8sSecret := b.store.GetSecret(secretName)
	if k8sSecret == nil {
		return nil, fmt.Errorf("Kubernetes secret %s not found", secretName.String())
	}
	if k8sSecret.Type != corev1.SecretTypeOpaque {
		return nil, fmt.Errorf("Kubernetes secret %s has type %s, expected %s",
			secretName.String(), k8sSecret.Type, corev1.SecretTypeOpaque)
	}

	name := secretName.String() + "/" + key
	value, exists := k8sSecret.Data[key]
	if !exists {
		return nil, fmt.Errorf("SDS secret %s: key %q not found in Kubernetes secret %s", name, key, secretName.String())
	}

	return &tlsv3.Secret{
		Name: name,
		Type: &tlsv3.Secret_GenericSecret{
			GenericSecret: &tlsv3.GenericSecret{
				Secret: &corev3.DataSource{
					Specifier: &corev3.DataSource_InlineBytes{
						InlineBytes: value,
					},
				},
			},
		},
	}, nil
}

// collectSDSSecretRefs walks the message, including packed Any values,
// and collects the references to secrets that Envoy requests from an xDS server.
func collectSDSSecretRefs(m protoreflect.Message, refs *[]*tlsv3.SdsSecretConfig) {
	switch msg := m.Interface().(type) {
	case *tlsv3.SdsSecretConfig:
		// Without sds_config the secret is static, with a path it is read from a file:
		// neither is served by the controller.
		source := msg.GetSdsConfig()
		if source.GetAds() != nil || source.GetApiConfigSource() != nil || source.GetSelf() != nil {
			*refs = append(*refs, msg)
		}
		return
	case *anypb.Any:
		// A type that is not linked into the controller cannot be inspected.
		if inner, err := msg.UnmarshalNew(); err == nil {
			collectSDSSecretRefs(inner.ProtoReflect(), refs)
		}
		return
	}

	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.IsMap():
			if fd.MapValue().Message() != nil {
				v.Map().Range(func(_ protoreflect.MapKey, mv protoreflect.Value) bool {
					collectSDSSecretRefs(mv.Message(), refs)
					return true
				})
			}
		case fd.Message() == nil:
		case fd.IsList():
			for i := 0; i < v.List().Len(); i++ {
				collectSDSSecretRefs(v.List().Get(i).Message(), refs)
			}
		default:
			collectSDSSecretRefs(v.Message(), refs)
		}
		return true
	})
}
