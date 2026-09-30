package main_builder

import (
	"testing"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	oauth2v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/oauth2/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func adsSource() *corev3.ConfigSource {
	return &corev3.ConfigSource{
		ConfigSourceSpecifier: &corev3.ConfigSource_Ads{Ads: &corev3.AggregatedConfigSource{}},
		ResourceApiVersion:    corev3.ApiVersion_V3,
	}
}

func oauth2Filter(t *testing.T, token, hmac *tlsv3.SdsSecretConfig) *hcmv3.HttpFilter {
	t.Helper()
	cfg, err := anypb.New(&oauth2v3.OAuth2{Config: &oauth2v3.OAuth2Config{
		Credentials: &oauth2v3.OAuth2Credentials{
			ClientId:    "client",
			TokenSecret: token,
			TokenFormation: &oauth2v3.OAuth2Credentials_HmacSecret{
				HmacSecret: hmac,
			},
		},
	}})
	require.NoError(t, err)
	return &hcmv3.HttpFilter{
		Name:       "envoy.filters.http.oauth2",
		ConfigType: &hcmv3.HttpFilter_TypedConfig{TypedConfig: cfg},
	}
}

func opaqueSecret(namespace, name string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Type:       corev1.SecretTypeOpaque,
		Data:       data,
	}
}

func TestBuildSDSSecrets(t *testing.T) {
	vsName := helpers.NamespacedName{Namespace: "default", Name: "vs"}

	t.Run("builds generic secrets referenced by the OAuth2 filter", func(t *testing.T) {
		s := store.New()
		s.SetSecret(opaqueSecret("ns", "oauth", map[string][]byte{
			"token":  []byte("token-value"),
			"hmac":   []byte("hmac-value"),
			"unused": []byte("unused-value"),
		}))
		b := &Builder{store: s}

		filters := []*hcmv3.HttpFilter{oauth2Filter(t,
			&tlsv3.SdsSecretConfig{Name: "ns/oauth/token", SdsConfig: adsSource()},
			&tlsv3.SdsSecretConfig{Name: "ns/oauth/hmac", SdsConfig: adsSource()},
		)}

		secrets, used := b.buildSDSSecrets(vsName, filters)

		got := map[string]string{}
		for _, secret := range secrets {
			got[secret.Name] = string(secret.GetGenericSecret().GetSecret().GetInlineBytes())
		}
		assert.Equal(t, map[string]string{
			"ns/oauth/token": "token-value",
			"ns/oauth/hmac":  "hmac-value",
		}, got)
		assert.Equal(t, []helpers.NamespacedName{{Namespace: "ns", Name: "oauth"}}, used)
	})

	t.Run("ignores secrets that Envoy loads from a file", func(t *testing.T) {
		b := &Builder{store: store.New()}
		fileSource := &corev3.ConfigSource{
			ConfigSourceSpecifier: &corev3.ConfigSource_PathConfigSource{
				PathConfigSource: &corev3.PathConfigSource{Path: "./oauth2-token.yaml"},
			},
		}

		filters := []*hcmv3.HttpFilter{oauth2Filter(t,
			&tlsv3.SdsSecretConfig{Name: "token", SdsConfig: fileSource},
			&tlsv3.SdsSecretConfig{Name: "hmac"},
		)}

		secrets, used := b.buildSDSSecrets(vsName, filters)
		assert.Empty(t, secrets)
		assert.Empty(t, used)
	})

	t.Run("skips a secret that is missing in Kubernetes", func(t *testing.T) {
		b := &Builder{store: store.New()}

		filters := []*hcmv3.HttpFilter{oauth2Filter(t,
			&tlsv3.SdsSecretConfig{Name: "ns/oauth/token", SdsConfig: adsSource()},
			&tlsv3.SdsSecretConfig{Name: "ns/oauth/hmac", SdsConfig: adsSource()},
		)}

		secrets, used := b.buildSDSSecrets(vsName, filters)
		assert.Empty(t, secrets)
		assert.Empty(t, used)
	})

	t.Run("skips a key that is missing in the Kubernetes secret", func(t *testing.T) {
		s := store.New()
		s.SetSecret(opaqueSecret("ns", "oauth", map[string][]byte{"token": []byte("token-value")}))
		b := &Builder{store: s}

		filters := []*hcmv3.HttpFilter{oauth2Filter(t,
			&tlsv3.SdsSecretConfig{Name: "ns/oauth/token", SdsConfig: adsSource()},
			&tlsv3.SdsSecretConfig{Name: "ns/oauth/hmac", SdsConfig: adsSource()},
		)}

		secrets, used := b.buildSDSSecrets(vsName, filters)
		require.Len(t, secrets, 1)
		assert.Equal(t, "ns/oauth/token", secrets[0].Name)
		assert.Equal(t, []helpers.NamespacedName{{Namespace: "ns", Name: "oauth"}}, used)
	})

	t.Run("skips a name that is not namespace/name or namespace/name/key", func(t *testing.T) {
		b := &Builder{store: store.New()}

		filters := []*hcmv3.HttpFilter{oauth2Filter(t,
			&tlsv3.SdsSecretConfig{Name: "token", SdsConfig: adsSource()},
			&tlsv3.SdsSecretConfig{Name: "a/b/c/d", SdsConfig: adsSource()},
		)}

		secrets, used := b.buildSDSSecrets(vsName, filters)
		assert.Empty(t, secrets)
		assert.Empty(t, used)
	})
}
