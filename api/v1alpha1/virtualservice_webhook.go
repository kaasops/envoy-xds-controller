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

	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/kaasops/envoy-xds-controller/pkg/errors"

	"google.golang.org/protobuf/encoding/protojson"
)

var (
	SecretRefType     = "secretRef"
	CertManagerType   = "certManagerType"
	AutoDiscoveryType = "autoDiscoveryType"
)

func (vs *VirtualService) Validate(
	ctx context.Context,
	unmarshaler protojson.UnmarshalOptions,
) error {
	// Validate Virtual Host spec
	if vs.Spec.VirtualHost == nil {
		return errors.New(errors.VirtualHostCantBeEmptyMessage)
	}
	vh := &routev3.VirtualHost{}
	if err := unmarshaler.Unmarshal(vs.Spec.VirtualHost.Raw, vh); err != nil {
		return errors.Wrap(err, errors.UnmarshalMessage)
	}

	// Check AccessLog spec
	if vs.Spec.AccessLog != nil {
		al := &accesslogv3.AccessLog{}
		if err := unmarshaler.Unmarshal(vs.Spec.AccessLog.Raw, al); err != nil {
			return errors.Wrap(err, errors.UnmarshalMessage)
		}
	}

	// Check HTTPFilters spec
	if vs.Spec.HTTPFilters != nil {
		for _, httpFilter := range vs.Spec.HTTPFilters {
			hf := &hcmv3.HttpFilter{}
			if err := unmarshaler.Unmarshal(httpFilter.Raw, hf); err != nil {
				return errors.Wrap(err, errors.UnmarshalMessage)
			}
		}
	}

	// Check listener exist
	if vs.Spec.Listener == nil {
		return errors.New(errors.ListenerCantBeEmptyMessage)
	}

	// Check TLSConfig
	if err := vs.Spec.TlsConfig.Validate(); err != nil {
		return errors.Wrap(err, errors.CannotValidateCacheResourceMessage)
	}

	return nil
}

func (tc *TlsConfig) Validate() error {
	if tc == nil {
		return nil
	}

	tlsType, err := tc.GetTLSType()
	if err != nil {
		return errors.Wrap(err, "cannot get TlsConfog Type")
	}

	switch tlsType {
	case SecretRefType:
		return nil
	case CertManagerType:
		cm := tc.CertManager
		return cm.validate()
	case AutoDiscoveryType:
		return nil
	}

	return errors.New("unexpected behavior when processing a TlsConfig Type")
}

func (tc *TlsConfig) GetTLSType() (string, error) {
	if tc.SecretRef != nil {
		if tc.CertManager != nil || tc.AutoDiscovery != nil {
			return "", errors.New(errors.ManyParamMessage)
		}
		return SecretRefType, nil
	}

	if tc.CertManager != nil {
		if tc.AutoDiscovery != nil {
			return "", errors.New(errors.ManyParamMessage)
		}
		return CertManagerType, nil
	}

	if tc.AutoDiscovery != nil {
		return AutoDiscoveryType, nil
	}

	return "", errors.New(errors.ZeroParamMessage)
}

func (cm *CertManager) validate() error {
	if cm.Issuer != nil {
		if cm.ClusterIssuer != nil {
			return errors.New(errors.TlsConfigManyParamMessage)
		}
		return nil
	}

	if cm.ClusterIssuer != nil {
		return nil
	}

	// TODO: Hoe check default in config????
	// if *cm.Enabled {
	// 	if options.DefaultClusterIssuer != "" {
	// 		return nil
	// 	}
	// }

	return errors.New("issuer for Certificate not set")
}
