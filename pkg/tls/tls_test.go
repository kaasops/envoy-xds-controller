package tls

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/config"
	fakeconfig "github.com/kaasops/envoy-xds-controller/pkg/config/fake"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	restclient "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func TestProvide(t *testing.T) {
	provideCase := func(
		tlsConfig *v1alpha1.TlsConfig,
		vh *routev3.VirtualHost,
		// nodeIDs []string,
		cfg config.Config,
		namespace string,
		cl client.Client,
		dc *discovery.DiscoveryClient,
		wantCerts map[string][]string,
		wantErr error,
	) func(t *testing.T) {
		return func(t *testing.T) {
			t.Helper()
			t.Parallel()
			req := require.New(t)

			// Generage fake Controller Runtime Client
			// cl := fake.NewClientBuilder().Build()

			if namespace == "default2" {
				fmt.Println("lol")
				// issuer := &cmapi.Issuer{}
				// namespacedName := types.NamespacedName{
				// 	Name:      "test",
				// 	Namespace: "default2",
				// }
				// err := cl.Get(context.Background(), namespacedName, issuer)
				// fmt.Println(err)
			}

			ctrl := New(cl, dc, tlsConfig, vh, cfg, namespace)

			log := log.FromContext(context.TODO()).WithName("For test")
			certs, err := ctrl.Provide(context.TODO(), log, make(map[string]corev1.Secret))
			req.Equal(certs, wantCerts)

			if !errors.Is(err, wantErr) {
				req.Equal(err, wantErr)
			}

		}
	}

	type testCase struct {
		name      string
		tlsConfig *v1alpha1.TlsConfig
		vh        *routev3.VirtualHost
		// nodeIDs   []string
		cfg       config.Config
		namespace string
		client    client.Client
		dc        *discovery.DiscoveryClient
		wantCerts map[string][]string
		wantErr   error
	}

	defaultConfig := fakeconfig.New("default", "defaultIssuer", 8000)
	defaultClient := fake.NewClientBuilder().Build()

	testCases := []testCase{
		{
			name:      "Without TlsConfig case",
			tlsConfig: nil,
			vh:        getVirtualHost_Default([]string{"test.io"}),
			// nodeIDs:   []string{"test"},
			cfg:       defaultConfig,
			namespace: "default",
			client:    defaultClient,
			dc:        getDiscoveryClient_With_CertManager_CRDs(t),
			wantCerts: nil,
			wantErr:   ErrTlsConfigNotExist,
		},
		// {
		// 	name:      "Without NodeIDs case",
		// 	tlsConfig: getTLSConfig_With_SecretRef(),
		// 	vh:        getVirtualHost_Default([]string{"test.io"}),
		// 	nodeIDs:   nil,
		// 	cfg:       defaultConfig,
		// 	namespace: "default",
		// 	client:    defaultClient,
		// 	dc:        getDiscoveryClient_With_CertManager_CRDs(t),
		// 	wantCerts: nil,
		// 	wantErr:   ErrNodeIDsEmpty,
		// },
		{
			name:      "TLSConfig. SecretRef and Certmanager enabled case",
			tlsConfig: getTLSConfig_With_SecretRef_And_CertManager(),
			vh:        getVirtualHost_Default([]string{"test.io"}),
			// nodeIDs:   []string{"test"},
			cfg:       defaultConfig,
			namespace: "default",
			client:    defaultClient,
			dc:        getDiscoveryClient_With_CertManager_CRDs(t),
			wantCerts: nil,
			wantErr:   ErrManyParam,
		},
		{
			name:      "TLSConfig. SecretRef and Certmanager disabled case",
			tlsConfig: getTLSConfig_With_Zero_Param(),
			vh:        getVirtualHost_Default([]string{"test.io"}),
			// nodeIDs:   []string{"test"},
			cfg:       defaultConfig,
			namespace: "default",
			client:    defaultClient,
			dc:        getDiscoveryClient_With_CertManager_CRDs(t),
			wantCerts: nil,
			wantErr:   ErrZeroParam,
		},
		{
			name:      "SecretRef. Kubernetes secret (with TLS cert) not found case",
			tlsConfig: getTLSConfig_With_SecretRef(),
			vh:        getVirtualHost_Default([]string{"test.io"}),
			// nodeIDs:   []string{"test"},
			cfg:       defaultConfig,
			namespace: "default",
			client:    defaultClient,
			dc:        getDiscoveryClient_With_CertManager_CRDs(t),
			wantCerts: nil,
			wantErr: api_errors.NewNotFound(
				schema.GroupResource{
					Group:    "",
					Resource: "secrets",
				},
				"test",
			),
		},
		{
			name:      "SecretRef. Kubernetes secret with TLS cert doesn't have TLS type case",
			tlsConfig: getTLSConfig_With_SecretRef(),
			vh:        getVirtualHost_Default([]string{"test.io"}),
			// nodeIDs:   []string{"test"},
			cfg:       defaultConfig,
			namespace: "default",
			client:    getClient_With_Secret_Wrong_Type(),
			dc:        getDiscoveryClient_With_CertManager_CRDs(t),
			wantCerts: nil,
			wantErr:   ErrSecretNotTLSType,
		},
		{
			name:      "SecretRef. Kubernetes secret without control-label case",
			tlsConfig: getTLSConfig_With_SecretRef(),
			vh:        getVirtualHost_Default([]string{"test.io"}),
			// nodeIDs:   []string{"test"},
			cfg:       defaultConfig,
			namespace: "default",
			client:    getClient_With_Secret_Without_Label(),
			dc:        getDiscoveryClient_With_CertManager_CRDs(t),
			wantCerts: nil,
			wantErr:   ErrControlLabelNotExist,
		},
		{
			name:      "SecretRef. Kubernetes secret with disabled control-label case",
			tlsConfig: getTLSConfig_With_SecretRef(),
			vh:        getVirtualHost_Default([]string{"test.io"}),
			// nodeIDs:   []string{"test"},
			cfg:       defaultConfig,
			namespace: "default",
			client:    getClient_With_Secret_With_Disabled_Label(),
			dc:        getDiscoveryClient_With_CertManager_CRDs(t),
			wantCerts: nil,
			wantErr:   ErrControlLabelWrong,
		},
		{
			name:      "SecretRef. Normal case",
			tlsConfig: getTLSConfig_With_SecretRef(),
			vh:        getVirtualHost_Default([]string{"test.io"}),
			// nodeIDs:   []string{"test"},
			cfg:       defaultConfig,
			namespace: "default",
			client:    getClient_With_Secret(),
			dc:        getDiscoveryClient_With_CertManager_CRDs(t),
			wantCerts: map[string][]string{
				"default-test": {"test.io"},
			},
			wantErr: nil,
		},
		{
			name:      "SecretRef. Many domains case",
			tlsConfig: getTLSConfig_With_SecretRef(),
			vh:        getVirtualHost_Default([]string{"test.io", "kaasops.io", "domain.com"}),
			// nodeIDs:   []string{"test"},
			cfg:       defaultConfig,
			namespace: "default",
			client:    getClient_With_Secret(),
			dc:        getDiscoveryClient_With_CertManager_CRDs(t),
			wantCerts: map[string][]string{
				"default-test": {"test.io", "kaasops.io", "domain.com"},
			},
			wantErr: nil,
		},
		{
			name:      "CertManager. CertManager CRDs not installed case",
			tlsConfig: getTLSConfig_With_CertManager_Issuer("test"),
			vh:        getVirtualHost_Default([]string{"test.io"}),
			// nodeIDs:   []string{"test"},
			cfg:       defaultConfig,
			namespace: "default",
			client:    getClient_With_Secret(),
			dc:        getDiscoveryClient_Without_CertManager_CRDs(t),
			wantCerts: nil,
			wantErr:   ErrCertManaferCRDNotExist,
		},
		{
			name:      "CertManager. Issuer not Exist case",
			tlsConfig: getTLSConfig_With_CertManager_Issuer("test"),
			vh:        getVirtualHost_Default([]string{"test.io"}),
			// nodeIDs:   []string{"test"},
			cfg:       defaultConfig,
			namespace: "default",
			client:    getClient_With_Secret_And_CertManager_CRDs("not-equal-name", "default"),
			dc:        getDiscoveryClient_With_CertManager_CRDs(t),
			wantCerts: nil,
			wantErr: api_errors.NewNotFound(
				schema.GroupResource{
					Group:    "cert-manager.io",
					Resource: "issuers",
				},
				"test",
			),
		},
		{
			name:      "CertManager. Issuer and ClusterIssue installed case",
			tlsConfig: getTLSConfig_With_CertManager_Issuer_And_Cluster_Issuer("test"),
			vh:        getVirtualHost_Default([]string{"test.io"}),
			// nodeIDs:   []string{"test"},
			cfg:       defaultConfig,
			namespace: "default",
			client:    getClient_With_Secret_And_CertManager_CRDs("test", "default"),
			dc:        getDiscoveryClient_With_CertManager_CRDs(t),
			wantCerts: nil,
			wantErr:   ErrTlsConfigManyParam,
		},
		{
			name:      "CertManager. Cluster Issuer not Exist case",
			tlsConfig: getTLSConfig_With_CertManager_Cluster_Issuer("test"),
			vh:        getVirtualHost_Default([]string{"test.io"}),
			// nodeIDs:   []string{"test"},
			cfg:       defaultConfig,
			namespace: "default",
			client:    getClient_With_Secret_And_CertManager_CRDs("not-equal-name", "default"),
			dc:        getDiscoveryClient_With_CertManager_CRDs(t),
			wantCerts: nil,
			wantErr: api_errors.NewNotFound(
				schema.GroupResource{
					Group:    "cert-manager.io",
					Resource: "clusterissuers",
				},
				"test",
			),
		},
		{
			name:      "CertManager. Use default Issuer case",
			tlsConfig: getTLSConfig_With_CertManager_Without_Issuer(),
			vh:        getVirtualHost_Default([]string{"test.io"}),
			// nodeIDs:   []string{"test"},
			cfg:       defaultConfig,
			namespace: "default",
			client:    getClient_With_Secret_And_CertManager_CRDs("defaultIssuer", "default"),
			dc:        getDiscoveryClient_With_CertManager_CRDs(t),
			wantCerts: nil,
			wantErr: api_errors.NewNotFound(
				schema.GroupResource{
					Group:    "cert-manager.io",
					Resource: "clusterissuers",
				},
				"defaultIssuer",
			),
		},
		{
			name:      "CertManager. Normal case",
			tlsConfig: getTLSConfig_With_CertManager_Issuer("test"),
			vh:        getVirtualHost_Default([]string{"test.io"}),
			// nodeIDs:   []string{"test"},
			cfg:       defaultConfig,
			namespace: "default",
			client:    getClient_With_Secret_And_CertManager_CRDs("test", "default"),
			dc:        getDiscoveryClient_With_CertManager_CRDs(t),
			wantCerts: map[string][]string{
				"default-test-io": {"test.io"},
			},
			wantErr: nil,
		},
		{
			name:      "CertManager. Many domains case",
			tlsConfig: getTLSConfig_With_CertManager_Issuer("test"),
			vh:        getVirtualHost_Default([]string{"test.io", "kaasops.io", "domain.com"}),
			// nodeIDs:   []string{"test"},
			cfg:       defaultConfig,
			namespace: "default",
			client:    getClient_With_Secret_And_CertManager_CRDs("test", "default"),
			dc:        getDiscoveryClient_With_CertManager_CRDs(t),
			wantCerts: map[string][]string{
				"default-test-io":    {"test.io"},
				"default-kaasops-io": {"kaasops.io"},
				"default-domain-com": {"domain.com"},
			},
			wantErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, provideCase(
			tc.tlsConfig,
			tc.vh,
			// tc.nodeIDs,
			tc.cfg,
			tc.namespace,
			tc.client,
			tc.dc,
			tc.wantCerts,
			tc.wantErr,
		))
	}

}

// Generate client.Client for diff cases
func getDefault_Secret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
	}
}

func getClient_With_Secret_Wrong_Type() client.Client {
	secret := getDefault_Secret()
	client := fake.NewClientBuilder().WithObjects(secret).Build()

	return client
}

func getClient_With_Secret_Without_Label() client.Client {
	secret := getDefault_Secret()
	secret.Type = corev1.SecretTypeTLS
	client := fake.NewClientBuilder().WithObjects(secret).Build()

	return client
}
func getClient_With_Secret_With_Disabled_Label() client.Client {
	secret := getDefault_Secret()
	secret.Type = corev1.SecretTypeTLS
	secret.ObjectMeta.Labels = map[string]string{
		SecretLabel: "False",
	}
	client := fake.NewClientBuilder().WithObjects(secret).Build()

	return client
}

func getClient_With_Secret() client.Client {
	secret := getDefault_Secret()
	secret.Type = corev1.SecretTypeTLS
	secret.ObjectMeta.Labels = map[string]string{
		SecretLabel: "true",
	}
	client := fake.NewClientBuilder().WithObjects(secret).Build()

	return client
}

func getClient_With_Secret_And_CertManager_CRDs(name, namespace string) client.Client {
	secret := getDefault_Secret()
	secret.Type = corev1.SecretTypeTLS
	secret.ObjectMeta.Labels = map[string]string{
		SecretLabel: "true",
	}

	issuer := &cmapi.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	clusterIssuer := &cmapi.ClusterIssuer{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	appsv1.AddToScheme(scheme)
	cmapi.AddToScheme(scheme)

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(issuer).
		WithStatusSubresource(clusterIssuer).
		WithObjects(secret).
		WithObjects(issuer).
		WithObjects(clusterIssuer).
		Build()

	return client
}

// Generate v1alpha1.TlsConfig for diff cases
func getTLSConfig_With_SecretRef() *v1alpha1.TlsConfig {
	secretRef := v1alpha1.ResourceRef{
		Name: "test",
	}
	tlsConfig := v1alpha1.TlsConfig{
		SecretRef: &secretRef,
	}

	return &tlsConfig
}

func getTLSConfig_With_SecretRef_And_CertManager() *v1alpha1.TlsConfig {
	tlsConfig := getTLSConfig_With_SecretRef()

	tlsConfig.CertManager = &v1alpha1.CertManager{}

	return tlsConfig
}

func getTLSConfig_With_Zero_Param() *v1alpha1.TlsConfig {
	tlsConfig := v1alpha1.TlsConfig{}

	return &tlsConfig
}

func getTLSConfig_With_CertManager_Issuer(name string) *v1alpha1.TlsConfig {
	certManager := v1alpha1.CertManager{
		Issuer: &name,
	}
	tlsConfig := v1alpha1.TlsConfig{
		CertManager: &certManager,
	}

	return &tlsConfig
}

func getTLSConfig_With_CertManager_Cluster_Issuer(name string) *v1alpha1.TlsConfig {
	certManager := v1alpha1.CertManager{
		ClusterIssuer: &name,
	}
	tlsConfig := v1alpha1.TlsConfig{
		CertManager: &certManager,
	}

	return &tlsConfig
}

func getTLSConfig_With_CertManager_Issuer_And_Cluster_Issuer(name string) *v1alpha1.TlsConfig {
	tlsConfig := getTLSConfig_With_CertManager_Issuer(name)

	tlsConfig.CertManager.ClusterIssuer = &name

	return tlsConfig
}
func getTLSConfig_With_CertManager_Without_Issuer() *v1alpha1.TlsConfig {
	enable := true
	certManager := v1alpha1.CertManager{
		Enabled: &enable,
	}
	tlsConfig := v1alpha1.TlsConfig{
		CertManager: &certManager,
	}

	return &tlsConfig
}

// Get Virtual Host for diff cases
func getVirtualHost_Default(domains []string) *routev3.VirtualHost {
	vhRouteMatchPrefix := routev3.RouteMatch_Prefix{
		Prefix: "/",
	}
	vhMatch := routev3.RouteMatch{
		PathSpecifier: &vhRouteMatchPrefix,
	}
	vhRouteCluster := routev3.RouteAction_Cluster{
		Cluster: "test",
	}
	vhROuteAction := routev3.RouteAction{
		ClusterSpecifier: &vhRouteCluster,
	}
	vhRouteRoute := routev3.Route_Route{
		Route: &vhROuteAction,
	}
	vhRoute := routev3.Route{
		Name:   "test",
		Match:  &vhMatch,
		Action: &vhRouteRoute,
	}

	vh := routev3.VirtualHost{
		Name:    "test",
		Domains: domains,
		Routes:  []*routev3.Route{&vhRoute},
	}

	return &vh
}

// Generate Discovery Client for diff cases
func getDiscoveryClient_Without_CertManager_CRDs(t *testing.T) *discovery.DiscoveryClient {
	stable := metav1.APIResourceList{
		GroupVersion: "v1",
		APIResources: []metav1.APIResource{
			{Name: "pods", Namespaced: true, Kind: "Pod"},
			{Name: "services", Namespaced: true, Kind: "Service"},
			{Name: "namespaces", Namespaced: false, Kind: "Namespace"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var list interface{}
		switch req.URL.Path {
		case "/api/v1":
			list = &stable
		case "/api":
			list = &metav1.APIVersions{
				Versions: []string{
					"v1",
				},
			}
		default:
			t.Logf("unexpected request: %s", req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		output, err := json.Marshal(list)
		if err != nil {
			t.Errorf("unexpected encoding error: %v", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	}))

	dc := discovery.NewDiscoveryClientForConfigOrDie(&restclient.Config{Host: server.URL})

	return dc
}

func getDiscoveryClient_With_CertManager_CRDs(t *testing.T) *discovery.DiscoveryClient {
	stable := metav1.APIResourceList{
		GroupVersion: "v1",
		APIResources: []metav1.APIResource{
			{Name: "pods", Namespaced: true, Kind: "Pod"},
			{Name: "services", Namespaced: true, Kind: "Service"},
			{Name: "namespaces", Namespaced: false, Kind: "Namespace"},
		},
	}

	certManagetApiResourceList := metav1.APIResourceList{
		GroupVersion: "cert-manager.io/v1",
		APIResources: []metav1.APIResource{
			{
				Name:       strings.ToLower(cmapi.ClusterIssuerKind),
				Namespaced: true,
				Kind:       cmapi.ClusterIssuerKind,
			},
			{
				Name:       strings.ToLower(cmapi.IssuerKind),
				Namespaced: true,
				Kind:       cmapi.IssuerKind,
			},
			{
				Name:       strings.ToLower(cmapi.CertificateKind),
				Namespaced: true,
				Kind:       cmapi.CertificateKind,
			},
			{
				Name:       strings.ToLower(cmapi.CertificateRequestKind),
				Namespaced: true,
				Kind:       cmapi.CertificateRequestKind,
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var list interface{}
		switch req.URL.Path {
		case "/api/v1":
			list = &stable
		case "/apis/cert-manager.io/v1":
			list = &certManagetApiResourceList
		case "/api":
			list = &metav1.APIVersions{
				Versions: []string{
					"v1",
				},
			}
		case "/apis":
			list = &metav1.APIGroupList{
				Groups: []metav1.APIGroup{
					{
						Name: "apps",
						Versions: []metav1.GroupVersionForDiscovery{
							{GroupVersion: "apis/cert-manager.io", Version: "v1"},
						},
					},
				},
			}
		default:
			t.Logf("unexpected request: %s", req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		output, err := json.Marshal(list)
		if err != nil {
			t.Errorf("unexpected encoding error: %v", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(output)
	}))

	dc := discovery.NewDiscoveryClientForConfigOrDie(&restclient.Config{Host: server.URL})

	return dc
}
