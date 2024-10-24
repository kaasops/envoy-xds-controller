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

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/pkg/options"
)

func (r *Route) Validate(ctx context.Context) error {
	// Validate Route spec
	if r.Spec == nil {
		return errors.New(errors.RouteCannotBeEmptyMessage)
	}

	for _, route := range r.Spec {
		routev3 := &routev3.Route{}
		if err := options.Unmarshaler.Unmarshal(route.Raw, routev3); err != nil {
			return errors.Wrap(err, errors.UnmarshalMessage)
		}
	}

	return nil
}

func (r *Route) ValidateDelete(ctx context.Context, cl client.Client) error {
	virtualServices := &VirtualServiceList{}
	listOpts := []client.ListOption{client.InNamespace(r.Namespace)}
	if err := cl.List(ctx, virtualServices, listOpts...); err != nil {
		return err
	}

	if len(virtualServices.Items) > 0 {
		vsNames := []string{}
	C1:
		for _, vs := range virtualServices.Items {
			for _, route := range vs.Spec.AdditionalRoutes {
				if route.Name == r.Name {
					vsNames = append(vsNames, vs.Name)
					continue C1
				}
			}
		}
		if len(vsNames) > 0 {
			return errors.New(fmt.Sprintf("%v%+v", errors.RouteDeleteUsed, vsNames))
		}
	}

	//

	virtualServiceTemplates := &VirtualServiceTemplateList{}
	vstlistOpts := []client.ListOption{client.InNamespace(r.Namespace)}
	if err := cl.List(ctx, virtualServiceTemplates, vstlistOpts...); err != nil {
		return err
	}

	if len(virtualServiceTemplates.Items) > 0 {
		vstNames := []string{}
	C2:
		for _, vst := range virtualServiceTemplates.Items {
			for _, route := range vst.Spec.AdditionalRoutes {
				if route.Name == r.Name {
					vstNames = append(vstNames, vst.Name)
					continue C2
				}
			}
		}
		if len(vstNames) > 0 {
			return errors.New(fmt.Sprintf("%v:%+v", errors.RouteUsedInVST, vstNames))
		}
	}

	return nil
}
