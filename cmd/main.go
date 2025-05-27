/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"

	xdsClients "github.com/kaasops/envoy-xds-controller/internal/xds/clients"

	"github.com/kaasops/envoy-xds-controller/internal/filewatcher"

	mgrCache "sigs.k8s.io/controller-runtime/pkg/cache"

	"github.com/go-logr/zapr"
	"go.uber.org/zap/zapcore"

	"github.com/kaasops/envoy-xds-controller/internal/store"

	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/kelseyhightower/envconfig"

	"github.com/kaasops/envoy-xds-controller/internal/xds"
	"github.com/kaasops/envoy-xds-controller/internal/xds/api"
	"github.com/kaasops/envoy-xds-controller/internal/xds/cache"
	"github.com/kaasops/envoy-xds-controller/internal/xds/updater"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	envoyv1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/controller"
	webhookenvoyv1alpha1 "github.com/kaasops/envoy-xds-controller/internal/webhook/v1alpha1"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(envoyv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

type Config struct {
	WatchNamespaces       []string `default:""                     envconfig:"WATCH_NAMESPACES"`
	InstallationNamespace string   `default:"envoy-xds-controller" envconfig:"INSTALLATION_NAMESPACE"`
	TargetNamespace       string   `default:"envoy-xds-controller" envconfig:"TARGET_NAMESPACE"` // ns for creating cr
	XDS                   struct {
		Port int `default:"9000" envconfig:"XDS_PORT"`
	}
	Webhook struct {
		TLSSecretName  string `default:"envoy-xds-controller-webhook-cert"           envconfig:"WEBHOOK_TLS_SECRET_NAME"`
		WebhookCfgName string `default:"envoy-xds-controller-validating-webhook-configuration" envconfig:"WEBHOOK_CFG_NAME"`
		ServiceName    string `default:"envoy-xds-controller-webhook-service"        envconfig:"SERVICE_NAME"`
		Path           string `default:"/validate"                                   envconfig:"WEBHOOK_PATH"`
		Port           int    `default:"9443"                                        envconfig:"WEBHOOK_PORT"`
	}
}

func (c *Config) GetNamespaceForResourceCreation() string {
	if c.TargetNamespace != "" {
		return c.TargetNamespace
	}
	if c.InstallationNamespace != "" {
		return c.InstallationNamespace
	}
	return "default"
}

// nolint:gocyclo
func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	var tlsOpts []func(*tls.Config)
	var enableAPI bool
	var cacheAPIPort int
	var cacheAPIScheme string
	var cacheAPIAddr string
	var grpcAPIPort int
	var devMode bool
	var accessControlModelPath string
	var accessControlPolicyPath string
	var configPath string
	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	flag.BoolVar(&enableAPI, "enable-cache-api", false, "Enable Cache API, for debug") // TODO: rename enable-api
	flag.IntVar(&cacheAPIPort, "cache-api-port", 9999, "Cache API port")
	flag.StringVar(&cacheAPIScheme, "cache-api-scheme", "http", "Cache API scheme")
	flag.StringVar(&cacheAPIAddr, "cache-api-addr", "localhost:9999", "Cache API address")
	flag.IntVar(&grpcAPIPort, "grpc-api-port", 10000, "GRPC API port")
	flag.BoolVar(&devMode, "development", false, "Enable dev mode")
	flag.StringVar(&accessControlModelPath,
		"access-control-model-path",
		"/etc/exc/access-control/model.conf",
		"Access Control Model Path",
	)
	flag.StringVar(
		&accessControlPolicyPath,
		"access-control-policy-path",
		"/etc/exc/access-control/policy.csv",
		"Access Control Policy Path",
	)
	flag.StringVar(
		&configPath,
		"config",
		"/etc/exc/config.json",
		"Config Path",
	)
	opts := zap.Options{
		Development: devMode,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	zapLevel := zap.Level(zapcore.InfoLevel)
	if devMode {
		zapLevel = zap.Level(zapcore.DebugLevel)
	}

	zapLogger := zap.NewRaw(zap.UseFlagOptions(&opts), zapLevel)
	ctrl.SetLogger(zapr.NewLogger(zapLogger))

	var cfg Config
	err := envconfig.Process("APP", &cfg)
	if err != nil {
		setupLog.Error(err, "unable to process env var")
		os.Exit(1)
	}

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
	})

	// Metrics endpoint is enabled in 'config/default/kustomization.yaml'. The Metrics options configure the server.
	// More info:
	// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/metrics/server
	// - https://book.kubebuilder.io/reference/metrics.html
	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		TLSOpts:       tlsOpts,
	}

	if secureMetrics {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/metrics/filters#WithAuthenticationAndAuthorization
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization

		// TODO(user): If CertDir, CertName, and KeyName are not specified, controller-runtime will automatically
		// generate self-signed certificates for the metrics server. While convenient for development and testing,
		// this setup is not recommended for production.
	}

	mgrOpts := ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "80f8c36d.kaasops.io",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	}
	if mgrCacheOpts := managerCacheOptions(&cfg); mgrCacheOpts != nil {
		setupLog.Info("watching namespaces", "namespaces", cfg.WatchNamespaces)
		mgrOpts.Cache = *mgrCacheOpts
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOpts)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	resStore := store.New()
	snapshotCache := cache.NewSnapshotCache()
	cacheUpdater := updater.NewCacheUpdater(snapshotCache, resStore)
	fWatcher, err := filewatcher.NewFileWatcher()
	if err != nil {
		setupLog.Error(err, "unable to create file watcher")
		os.Exit(1)
	}
	defer fWatcher.Cancel()

	if err = (&controller.ClusterReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Updater: cacheUpdater,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Cluster")
		os.Exit(1)
	}
	if err = (&controller.ListenerReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Updater: cacheUpdater,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Listener")
		os.Exit(1)
	}
	if err = (&controller.RouteReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Updater: cacheUpdater,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Route")
		os.Exit(1)
	}
	if err = (&controller.VirtualServiceReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Updater: cacheUpdater,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VirtualService")
		os.Exit(1)
	}
	if err = (&controller.AccessLogConfigReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Updater: cacheUpdater,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AccessLogConfig")
		os.Exit(1)
	}
	if err = (&controller.HttpFilterReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Updater: cacheUpdater,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "HttpFilter")
		os.Exit(1)
	}
	if err = (&controller.PolicyReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Updater: cacheUpdater,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Policy")
		os.Exit(1)
	}
	if err = (&controller.VirtualServiceTemplateReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Updater: cacheUpdater,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VirtualServiceTemplate")
		os.Exit(1)
	}
	if err = (&controller.SecretReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Updater: cacheUpdater,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Secret")
		os.Exit(1)
	}

	// nolint:goconst
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {

		// InitFromKubernetes Kubernetes Client for Webhook
		webhookClient, err := client.New(ctrl.GetConfigOrDie(), client.Options{
			Scheme: mgr.GetScheme(),
			Mapper: mgr.GetRESTMapper(),
		})
		if err != nil {
			setupLog.Error(err, "unable to create webhook client")
			os.Exit(1)
		}

		// Enable Webhook Reconcile for create Certificate
		webhookReconciler := &controller.WebhookReconciler{
			Client:                webhookClient,
			Scheme:                mgr.GetScheme(),
			Namespace:             cfg.InstallationNamespace,
			TLSSecretName:         cfg.Webhook.TLSSecretName,
			ValidationWebhookName: cfg.Webhook.WebhookCfgName,
		}
		if err = webhookReconciler.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Webhook")
			os.Exit(1)
		}

		// Check secret with TLS for webhook
		certSecret := &corev1.Secret{
			ObjectMeta: ctrl.ObjectMeta{
				Name:      cfg.Webhook.TLSSecretName,
				Namespace: cfg.InstallationNamespace,
			},
		}
		// Reconcile secret with TLS for webhook
		if err := webhookReconciler.ReconcileCertificates(context.Background(), certSecret); err != nil {
			setupLog.Error(err, "unable to reconcile webhook secret")
			os.Exit(1)
		}

		if err = webhookenvoyv1alpha1.SetupAccessLogConfigWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "AccessLogConfig")
			os.Exit(1)
		}
		if err = webhookenvoyv1alpha1.SetupListenerWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Listener")
			os.Exit(1)
		}
		if err = webhookenvoyv1alpha1.SetupClusterWebhookWithManager(mgr, cacheUpdater); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Cluster")
			os.Exit(1)
		}
		if err = webhookenvoyv1alpha1.SetupRouteWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Route")
			os.Exit(1)
		}
		if err = webhookenvoyv1alpha1.SetupPolicyWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Policy")
			os.Exit(1)
		}
		if err = webhookenvoyv1alpha1.SetupHttpFilterWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "HttpFilter")
			os.Exit(1)
		}
		if err = webhookenvoyv1alpha1.SetupVirtualServiceWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "VirtualService")
			os.Exit(1)
		}
		if err = webhookenvoyv1alpha1.SetupSecretWebhookWithManager(mgr, cacheUpdater); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Secret")
			os.Exit(1)
		}
		if err = webhookenvoyv1alpha1.SetupVirtualServiceTemplateWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "VirtualServiceTemplate")
			os.Exit(1)
		}
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	var startServers manager.RunnableFunc = func(ctx context.Context) error {
		setupServers := log.FromContext(ctx)
		setupServers.Info("Starting servers")

		if err := cacheUpdater.InitFromKubernetes(ctx, mgr.GetClient()); err != nil {
			return fmt.Errorf("unable to init cache updater: %w", err)
		}

		connectedClients := xdsClients.NewRegistry()

		go func() {
			srv := server.NewServer(ctx, snapshotCache, xds.NewCallbacks(
				ctrl.Log.WithName("xds.server.callbacks"),
				connectedClients),
			)
			if err = xds.RunServer(srv, cfg.XDS.Port); err != nil {
				setupServers.Error(err, "cannot run xDS server")
				os.Exit(1)
			}
		}()

		if enableAPI {
			go func() {
				apiServerCfg := &api.Config{}
				apiServerCfg.EnableDevMode = devMode
				apiServerCfg.Auth.Enabled, _ = strconv.ParseBool(os.Getenv("OIDC_ENABLED"))
				apiServerCfg.Auth.IssuerURL = os.Getenv("OIDC_ISSUER_URL")
				apiServerCfg.Auth.ClientID = os.Getenv("OIDC_CLIENT_ID")
				apiServerCfg.Auth.AccessControlModel = accessControlModelPath
				apiServerCfg.Auth.AccessControlPolicy = accessControlPolicyPath
				if acl := os.Getenv("ACL_CONFIG"); acl != "" {
					err = json.Unmarshal([]byte(acl), &apiServerCfg.Auth.ACL)
					if err != nil {
						setupServers.Error(err, "failed to parse ACL config")
						os.Exit(1)
					}
				}
				data, err := os.ReadFile(configPath)
				if err != nil {
					setupServers.Error(err, "failed to read config")
					os.Exit(1)
				}
				if err = json.Unmarshal(data, &apiServerCfg.StaticResources); err != nil {
					setupServers.Error(err, "failed to parse static resources config")
					os.Exit(1)
				}
				apiServer, err := api.New(snapshotCache, apiServerCfg, zapLogger, devMode, fWatcher, configPath)
				if err != nil {
					setupServers.Error(err, "failed to create api server")
					os.Exit(1)
				}
				if err := apiServer.RunGRPC(
					grpcAPIPort,
					resStore,
					mgr.GetClient(),
					cfg.GetNamespaceForResourceCreation(),
				); err != nil {
					setupServers.Error(err, "cannot run grpc xDS server")
					os.Exit(1)
				}
				if err := apiServer.
					Run(cacheAPIPort, cacheAPIScheme, cacheAPIAddr); err != nil {
					setupServers.Error(err, "cannot run http xDS server")
					os.Exit(1)
				}
			}()
		}

		if devMode {
			go func() {
				dLog := log.FromContext(ctx).WithName("debug-server")

				http.HandleFunc("/debug/store", func(w http.ResponseWriter, r *http.Request) {
					data, err := cacheUpdater.GetMarshaledStore()
					if err != nil {
						dLog.Error(err, "failed to marshal store")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					_, _ = w.Write(data)
				})
				http.HandleFunc("/debug/xds", func(w http.ResponseWriter, r *http.Request) {
					keys := snapshotCache.GetNodeIDsAsMap()
					dump := make(map[string]any)
					for key := range keys {
						snapshot, _ := snapshotCache.GetSnapshot(key)
						dump[key] = snapshot
					}
					data, err := json.MarshalIndent(dump, "", "\t")
					if err != nil {
						dLog.Error(err, "failed to marshal xds")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					_, _ = w.Write(data)
				})
				http.HandleFunc("/debug/used-secrets", func(w http.ResponseWriter, r *http.Request) {
					secrets := cacheUpdater.GetUsedSecrets()
					m := make(map[string]string, len(secrets))
					for k, v := range secrets {
						m[k.String()] = v.String()
					}
					data, err := json.MarshalIndent(m, "", "\t")
					if err != nil {
						dLog.Error(err, "failed to marshal used secrets")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					_, _ = w.Write(data)
				})
				http.HandleFunc("/debug/connected-clients", func(w http.ResponseWriter, r *http.Request) {
					data, err := json.MarshalIndent(connectedClients.List(), "", "\t")
					if err != nil {
						dLog.Error(err, "failed to marshal connected clients")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					_, _ = w.Write(data)
				})
				_ = http.ListenAndServe(":4444", nil)
			}()
		}
		return nil
	}

	if err = mgr.Add(startServers); err != nil {
		setupLog.Error(err, "unable to add startServers to manager")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func managerCacheOptions(cfg *Config) *mgrCache.Options {
	if len(cfg.WatchNamespaces) == 0 {
		return nil
	}
	mgrCacheOpts := &mgrCache.Options{
		DefaultNamespaces: make(map[string]mgrCache.Config),
	}
	for _, namespace := range cfg.WatchNamespaces {
		mgrCacheOpts.DefaultNamespaces[namespace] = mgrCache.Config{}
	}
	mgrCacheOpts.DefaultNamespaces[cfg.InstallationNamespace] = mgrCache.Config{}
	return mgrCacheOpts
}
