// /*
// Copyright 2023.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package v1alpha1

import (
	"context"
	"fmt"
	"slices"

	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/kaasops/envoy-xds-controller/pkg/config"
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/pkg/options"
	"github.com/kaasops/envoy-xds-controller/pkg/utils"
	"github.com/kaasops/envoy-xds-controller/pkg/utils/k8s"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	SecretRefType     = "secretRef"
	CertManagerType   = "certManagerType"
	AutoDiscoveryType = "autoDiscoveryType"
)

func (vs *VirtualService) Validate(
	ctx context.Context,
	config *config.Config,
	client client.Client,
	dc *discovery.DiscoveryClient,
) error {
	/**
		Validate struct
	**/

	// Validate Virtual Host spec
	if vs.Spec.VirtualHost == nil {
		return errors.New(errors.VirtualHostCantBeEmptyMessage)
	}
	vh := &routev3.VirtualHost{}
	if err := options.Unmarshaler.Unmarshal(vs.Spec.VirtualHost.Raw, vh); err != nil {
		return errors.Wrap(err, errors.UnmarshalMessage)
	}

	// Check AccessLog spec
	if vs.Spec.AccessLog != nil {
		al := &accesslogv3.AccessLog{}
		if err := options.Unmarshaler.Unmarshal(vs.Spec.AccessLog.Raw, al); err != nil {
			return errors.Wrap(err, errors.UnmarshalMessage)
		}
	}

	// Check HTTPFilters spec
	if vs.Spec.HTTPFilters != nil {
		for _, httpFilter := range vs.Spec.HTTPFilters {
			hf := &hcmv3.HttpFilter{}
			if err := options.Unmarshaler.Unmarshal(httpFilter.Raw, hf); err != nil {
				return errors.Wrap(err, errors.UnmarshalMessage)
			}
		}
	}

	// Check UpgradeConfigs spec
	if vs.Spec.UpgradeConfigs != nil {
		for _, upgradeConfig := range vs.Spec.UpgradeConfigs {
			uc := &hcmv3.HttpConnectionManager_UpgradeConfig{}
			if err := options.Unmarshaler.Unmarshal(upgradeConfig.Raw, uc); err != nil {
				return errors.Wrap(err, errors.UnmarshalMessage)
			}
			if err := uc.Validate(); err != nil {
				return errors.Wrap(err, errors.CannotValidateCacheResourceMessage)
			}
		}
	}

	// Check listener set
	if vs.Spec.Listener == nil {
		return errors.New(errors.ListenerCannotBeEmptyMessage)
	}

	/**
	Check
	**/

	// Check Listener exist in Kubernetes
	listener := &Listener{}
	err := client.Get(
		ctx,
		types.NamespacedName{
			Namespace: vs.Namespace,
			Name:      vs.Spec.Listener.Name,
		},
		listener)
	if err != nil {
		return err
	}

	// Check AccessLogConfig exist in Kubernetes
	if vs.Spec.AccessLogConfig != nil {
		accessLogConfig := &AccessLogConfig{}
		err := client.Get(
			ctx,
			types.NamespacedName{
				Namespace: vs.Namespace,
				Name:      vs.Spec.AccessLogConfig.Name,
			},
			accessLogConfig)
		if err != nil {
			return err
		}
	}

	// Check Additional HTTP Filters exists in Kubernetes
	if vs.Spec.AdditionalHttpFilters != nil {
		for _, ahf := range vs.Spec.AdditionalHttpFilters {
			httpFilter := &HttpFilter{}
			err := client.Get(
				ctx,
				types.NamespacedName{
					Namespace: vs.Namespace,
					Name:      ahf.Name,
				},
				httpFilter)
			if err != nil {
				return err
			}
		}
	}

	// Check Additional Routes exists in Kubernetes
	if vs.Spec.AdditionalRoutes != nil {
		for _, ar := range vs.Spec.AdditionalRoutes {
			route := &Route{}
			err := client.Get(
				ctx,
				types.NamespacedName{
					Namespace: vs.Namespace,
					Name:      ar.Name,
				},
				route)
			if err != nil {
				return err
			}
		}
	}

	// Check if VisturalService alredy exist for domain
	err = vs.checkIfDomainAlredyExist(ctx, client)
	if err != nil {
		return err
	}

	// Check TLSConfig
	if err := vs.Spec.TlsConfig.Validate(ctx, vs, config, client, dc); err != nil {
		return errors.Wrap(err, errors.CannotValidateCacheResourceMessage)
	}

	// TODO: Check cluster exist in Kubernetes for VS and used in routes

	return nil
}

func (vs *VirtualService) checkIfDomainAlredyExist(
	ctx context.Context,
	cl client.Client,
) error {
	// Get domain from Virtual Host
	vh := &routev3.VirtualHost{}
	if err := options.Unmarshaler.Unmarshal(vs.Spec.VirtualHost.Raw, vh); err != nil {
		return errors.Wrap(err, errors.UnmarshalMessage)
	}
	vsDomains := vh.Domains

	virtualServices := &VirtualServiceList{}
	listOpts := []client.ListOption{
		client.InNamespace(vs.Namespace),
		client.MatchingFields{options.VirtualServiceListenerNameFeild: vs.Spec.Listener.Name},
	}
	if err := cl.List(ctx, virtualServices, listOpts...); err != nil {
		return err
	}

	for _, virtualService := range virtualServices.Items {
		// skip if VirtualService is the same
		if virtualService.Name == vs.Name && virtualService.Namespace == vs.Namespace {
			continue
		}

		vh := &routev3.VirtualHost{}
		if err := options.Unmarshaler.Unmarshal(virtualService.Spec.VirtualHost.Raw, vh); err != nil {
			return errors.Wrap(err, errors.UnmarshalMessage)
		}

		for _, d := range vsDomains {
			if slices.Contains(vh.Domains, d) {
				return errors.Newf("domain %s alredy exist in VertualService %s", d, virtualService.Name)
			}
		}
	}

	return nil
}

func (tc *TlsConfig) Validate(
	ctx context.Context,
	vs *VirtualService,
	cfg *config.Config,
	client client.Client,
	dc *discovery.DiscoveryClient,
) error {
	if tc == nil {
		return nil
	}

	tlsType, err := tc.GetTLSType()
	if err != nil {
		return errors.Wrap(err, "cannot get TlsConfig Type")
	}

	// If Watch Namespaces set - try to found secret in all namespaces
	namespaces := cfg.GetWatchNamespaces()

	switch tlsType {
	case SecretRefType:
		// If .Spec.TlsConfig.SecretRef.Namespace set - find secret only in this namespace
		if vs.Spec.TlsConfig.SecretRef.Namespace != nil {
			namespaces = []string{*vs.Spec.TlsConfig.SecretRef.Namespace}
		} else {
			namespaces = []string{vs.Namespace}
		}

		return validateSecretRef(ctx, client, namespaces, tc.SecretRef)
	case AutoDiscoveryType:
		return validateAutoDiscovery(ctx, vs, namespaces, client)
	}

	return errors.New("unexpected behavior when processing a TlsConfig Type")
}

func validateSecretRef(
	ctx context.Context,
	client client.Client,
	namespaces []string,
	rr *ResourceRef,
) error {
	secrets, err := k8s.GetCertificateSecrets(ctx, client, namespaces)
	if err != nil {
		return err
	}

	for _, secret := range secrets {
		if rr.Namespace == nil {
			if secret.Name == rr.Name {
				return nil
			}
		} else {
			if secret.Namespace == *rr.Namespace {
				if secret.Name == rr.Name {
					return nil
				}
			}
		}
	}

	if rr.Namespace != nil {
		return errors.New(fmt.Sprintf("Secret %s/%s from .Spec.TlsConfig.SecretRef not found", *rr.Namespace, rr.Name))
	}
	return errors.New(fmt.Sprintf("Secret %s/%s from .Spec.TlsConfig.SecretRef not found", namespaces[0], rr.Name))
	// TODO (may be). Add check Certificate in secret have VS domain in DNS
}

func validateAutoDiscovery(
	ctx context.Context,
	vs *VirtualService,
	namespaces []string,
	client client.Client,
) error {
	// Get Virtual Host from Virtual Service
	virtualHost := &routev3.VirtualHost{}
	if err := options.Unmarshaler.Unmarshal(vs.Spec.VirtualHost.Raw, virtualHost); err != nil {
		return errors.Wrap(err, errors.UnmarshalMessage)
	}

	// Create index for fast search certificate for domain
	index, err := k8s.IndexCertificateSecrets(ctx, client, namespaces)
	if err != nil {
		return errors.Wrap(err, "cannot generate TLS certificates index from Kubernetes secrets")
	}

	for _, domain := range virtualHost.Domains {
		_, ok := index[domain]
		if !ok {
			// try to find cert for wildcard
			wildcardDomain := utils.GetWildcardDomain(domain)
			_, ok := index[wildcardDomain]
			if !ok {
				return errors.Newf("%s. Domain: %s", errors.DicoverNotFoundMessage, domain)
			}
		}
	}

	return nil
}
