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
		httpFilterv3 := &hcmv3.HttpFilter{}
		if err := options.Unmarshaler.Unmarshal(httpFilter.Raw, httpFilterv3); err != nil {
			return errors.Wrap(err, errors.UnmarshalMessage)
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

	return nil
}
