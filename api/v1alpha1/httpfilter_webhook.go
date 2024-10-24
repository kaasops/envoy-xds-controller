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
	oauth2v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/oauth2/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/pkg/options"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *HttpFilter) Validate(ctx context.Context) error {
	// Validate HttpFilter spec
	if h.Spec == nil {
		return errors.New(errors.HTTPFilterCannotBeEmptyMessage)
	}

	for _, httpFilter := range h.Spec {
		hf := &hcmv3.HttpFilter{}
		if err := UnmarshalAndValidateHTTPFilter(httpFilter.Raw, hf); err != nil {
			return err
		}
	}

	return nil
}

// ValidateDelete - check if the HTTPFilter used by any virtual service
func (h *HttpFilter) ValidateDelete(ctx context.Context, cl client.Client) error {
	virtualServices := &VirtualServiceList{}
	listOpts := []client.ListOption{client.InNamespace(h.Namespace)}
	if err := cl.List(ctx, virtualServices, listOpts...); err != nil {
		return err
	}

	if len(virtualServices.Items) > 0 {
		vsNames := []string{}
	C1:
		for _, vs := range virtualServices.Items {
			for _, httpFilter := range vs.Spec.AdditionalHttpFilters {
				if httpFilter.Name == h.Name {
					vsNames = append(vsNames, vs.Name)
					continue C1
				}
			}
		}
		if len(vsNames) > 0 {
			return errors.New(fmt.Sprintf("%v%+v", errors.HTTPFilterDeleteUsed, vsNames))
		}
	}

	virtualServiceTemplates := &VirtualServiceTemplateList{}
	vstListOpts := []client.ListOption{client.InNamespace(h.Namespace)}
	if err := cl.List(ctx, virtualServiceTemplates, vstListOpts...); err != nil {
		return err
	}

	if len(virtualServiceTemplates.Items) > 0 {
		vstNames := []string{}
	C2:
		for _, vst := range virtualServiceTemplates.Items {
			for _, httpFilter := range vst.Spec.AdditionalHttpFilters {
				if httpFilter.Name == h.Name {
					vstNames = append(vstNames, vst.Name)
					continue C2
				}
			}
		}
		if len(vstNames) > 0 {
			return errors.New(fmt.Sprintf("%v:%+v", errors.HTTPFilterUsedInVST, vstNames))
		}
	}

	return nil
}

func UnmarshalAndValidateHTTPFilter(raw []byte, httpFilter *hcmv3.HttpFilter) error {
	if err := options.Unmarshaler.Unmarshal(raw, httpFilter); err != nil {
		return errors.Wrap(err, errors.UnmarshalMessage)
	}
	if err := httpFilter.ValidateAll(); err != nil {
		return errors.WrapUKS(err, errors.InvalidHTTPFilter)
	}
	switch v := httpFilter.ConfigType.(type) {
	case *hcmv3.HttpFilter_TypedConfig:
		switch v.TypedConfig.TypeUrl {
		case "type.googleapis.com/envoy.extensions.filters.http.oauth2.v3.OAuth2":
			if err := validateOAuth2Filter(v); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateOAuth2Filter(v *hcmv3.HttpFilter_TypedConfig) error {
	var oauthCfg oauth2v3.OAuth2
	if err := v.TypedConfig.UnmarshalTo(&oauthCfg); err != nil {
		return errors.Wrap(err, errors.UnmarshalMessage)
	}
	if err := oauthCfg.ValidateAll(); err != nil {
		return errors.WrapUKS(err, errors.InvalidHTTPFilter)
	}
	if oauthCfg.Config.PreserveAuthorizationHeader && oauthCfg.Config.ForwardBearerToken {
		return errors.Newf("%s: preserve_authorization_header=true and forward_bearer_token=true", errors.InvalidParamsCombination)
	}
	return nil
}
