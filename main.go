/*
Copyright 2023.

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
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	"k8s.io/client-go/discovery"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	v1alpha1 "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/config"
	"github.com/kaasops/envoy-xds-controller/pkg/options"
	"github.com/kaasops/envoy-xds-controller/pkg/webhook/handler"
	xdsclient "github.com/kaasops/envoy-xds-controller/pkg/xds/api"
	xdscache "github.com/kaasops/envoy-xds-controller/pkg/xds/cache"
	"github.com/kaasops/envoy-xds-controller/pkg/xds/server"

	testv3 "github.com/envoyproxy/go-control-plane/pkg/test/v3"

	"github.com/kaasops/envoy-xds-controller/controllers"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	utilruntime.Must(cmapi.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var enableCacheAPI bool
	var cacheAPIPort int
	var cacheAPIScheme string
	var cacheAPIAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&enableCacheAPI, "enable-cache-api", false, "Enable Cache API, for debug")
	flag.IntVar(&cacheAPIPort, "cache-api-port", 9999, "Cache API port")
	flag.StringVar(&cacheAPIScheme, "cache-api-scheme", "http", "Cache API scheme")
	flag.StringVar(&cacheAPIAddr, "cache-api-addr", "localhost:9999", "Cache API address")

	cfg, err := config.New()
	if err != nil {
		setupLog.Error(err, "Can't get params from env")
	}

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	secretReq, err := labels.NewRequirement(options.SecretLabelKey, selection.In, []string{options.SdsSecretLabelValue, options.WebhookSecretLabelValue})
	if err != nil {
		setupLog.Error(err, "Failed to build label requirement for secrets")
	}

	mgrOpts := ctrl.Options{
		Scheme:                        scheme,
		MetricsBindAddress:            metricsAddr,
		HealthProbeBindAddress:        probeAddr,
		LeaderElection:                enableLeaderElection,
		LeaderElectionID:              "80f8c36d.kaasops.io",
		Namespace:                     cfg.GetWatchNamespace(),
		LeaderElectionReleaseOnCancel: true,
		Cache: cache.Options{ByObject: map[client.Object]cache.ByObject{
			&corev1.Secret{}: {Label: labels.NewSelector().Add(*secretReq)},
		}},
	}

	if !cfg.Webhook.Disable {
		mgrOpts.WebhookServer = webhook.NewServer(webhook.Options{
			Port:    cfg.GetWebhookPort(),
			CertDir: "/tmp/k8s-webhook-server/serving-certs",
		})
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOpts)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	config := ctrl.GetConfigOrDie()
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		setupLog.Error(err, "unable to create discovery client")
		os.Exit(1)
	}

	// Register Webhook
	if !cfg.Webhook.Disable {
		webhookClient, err := client.New(ctrl.GetConfigOrDie(), client.Options{
			Scheme: mgr.GetScheme(),
			Mapper: mgr.GetRESTMapper(),
		})
		if err != nil {
			setupLog.Error(err, "unable to create webhook client")
			os.Exit(1)
		}

		// Enable Webhook Reconcile for create Certificates
		webhookReconciler := &controllers.WebhookReconciler{
			Client:    webhookClient,
			Scheme:    mgr.GetScheme(),
			Namespace: cfg.GetInstalationNamespace(),
			Config:    cfg,
		}
		if err = webhookReconciler.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Webhook")
			os.Exit(1)
		}

		// Check secret with TLS for webhook
		certSecret := &corev1.Secret{
			ObjectMeta: ctrl.ObjectMeta{
				Name:      cfg.GetTLSSecretName(),
				Namespace: cfg.GetInstalationNamespace(),
			},
		}

		if err := webhookClient.Get(context.Background(), types.NamespacedName{Namespace: certSecret.Namespace, Name: certSecret.Name}, certSecret); err != nil {
			setupLog.Error(err, "unable to get webhook secret")
			os.Exit(1)
		}

		// Reconcile secret with TLS for webhook
		if err := webhookReconciler.ReconcileCertificates(context.Background(), certSecret); err != nil {
			setupLog.Error(err, "unable to reconcile webhook secret")
			os.Exit(1)
		}

		// Register Webhook Server
		mgr.GetWebhookServer().Register(
			cfg.GetWebhookPath(),
			&webhook.Admission{
				Handler: &handler.Handler{
					Config:          cfg,
					Client:          mgr.GetClient(),
					DiscoveryClient: discoveryClient,
				},
			},
		)
	}

	xDSCache := xdscache.New()
	xDSServer := server.New(xDSCache, &testv3.Callbacks{Debug: true})
	go xDSServer.Run(cfg.GetXDSPort())

	if enableCacheAPI {
		go func() {
			if err := xdsclient.New(xDSCache).Run(cacheAPIPort, cacheAPIScheme, cacheAPIAddr); err != nil {
				setupLog.Error(err, "cannot run http xDS server")
				os.Exit(1)
			}
		}()
	}

	if err = (&controllers.ClusterReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Cache:  xDSCache,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Cluster")
		os.Exit(1)
	}
	if err = (&controllers.ListenerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Cache:  xDSCache,
		// DiscoveryClient: discoveryClient,
		Config: cfg,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Listener")
		os.Exit(1)
	}
	if err = (&controllers.EndpointReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Cache:  xDSCache,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Endpoint")
		os.Exit(1)
	}
	if err = (&controllers.VirtualHostReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Cache:  xDSCache,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VirtualHost")
		os.Exit(1)
	}
	if err = (&controllers.SecretReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Cache:  xDSCache,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Secret")
		os.Exit(1)
	}
	if err = (&controllers.KubeSecretReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Cache:  xDSCache,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Secret Certificare")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
