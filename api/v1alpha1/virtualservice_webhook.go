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

	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"

	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// import (
// 	"k8s.io/apimachinery/pkg/runtime"
// )

func (r *VirtualService) Validate(cl client.Client, unmarshaler *protojson.UnmarshalOptions) error {
	// Validate Virtual Host spec
	if r.Spec.VirtualHost == nil {
		return fmt.Errorf("virtualHost could not be empty")
	}
	vh := &routev3.VirtualHost{}
	if err := unmarshaler.Unmarshal(r.Spec.VirtualHost.Raw, vh); err != nil {
		return fmt.Errorf("can't unmarshal Virtual Host. Error: %s", err.Error())
	}

	// Check AccessLog spec
	if r.Spec.AccessLog != nil {
		al := &accesslogv3.AccessLog{}
		if err := unmarshaler.Unmarshal(r.Spec.AccessLog.Raw, al); err != nil {
			return err
		}
	}

	// Check HTTPFilters spec
	if r.Spec.HTTPFilters != nil {
		for _, httpFilter := range r.Spec.HTTPFilters {
			hf := &hcmv3.HttpFilter{}
			if err := unmarshaler.Unmarshal(httpFilter.Raw, hf); err != nil {
				return err
			}
		}
	}

	// Check listener exist
	if r.Spec.Listener == nil {
		return fmt.Errorf("listener could not be empty")
	}
	listener := &Listener{}
	listenerNN := types.NamespacedName{
		Namespace: r.Namespace,
		Name:      r.Spec.Listener.Name,
	}
	if err := cl.Get(context.Background(), listenerNN, listener); err != nil {
		return err
	}

	// Check AccessLogsConfig exist
	if r.Spec.AccessLogConfig != nil {
		alc := &AccessLogConfig{}
		alcNN := types.NamespacedName{
			Namespace: r.Namespace,
			Name:      r.Spec.AccessLogConfig.Name,
		}
		if err := cl.Get(context.Background(), alcNN, alc); err != nil {
			return err
		}
	}

	// Check AdditionalRoutes exist
	if r.Spec.AdditionalRoutes != nil {
		for _, ar := range r.Spec.AdditionalRoutes {
			route := &Route{}
			routeNN := types.NamespacedName{
				Namespace: r.Namespace,
				Name:      ar.Name,
			}
			if err := cl.Get(context.Background(), routeNN, route); err != nil {
				return err
			}
		}
	}

	// Check Domains. Error if already exist
	virtualServices := &VirtualServiceList{}
	listOpts := []client.ListOption{
		client.InNamespace(r.Namespace),
		client.MatchingFields{"spec.listener.name": r.Spec.Listener.Name},
	}

	if err := cl.List(context.Background(), virtualServices, listOpts...); err != nil {
		return err
	}

	for _, vs := range virtualServices.Items {
		if vs.Name == r.Name {
			continue
		}

		vhVS := &routev3.VirtualHost{}
		unmarshaler.Unmarshal(vs.Spec.VirtualHost.Raw, vhVS)

		for _, domainVS := range vhVS.Domains {
			if slices.Contains(vh.Domains, domainVS) {
				return fmt.Errorf("domain %s alredy exist in Virtual Service %s", domainVS, vs.Name)
			}
		}
	}

	return nil
}

// func (r*)

// // ValidateCreate implements webhook.Validator so a webhook will be registered for the type
// func (r *VirtualService) ValidateCreate() error {
// 	virtualservicelog.Info("validate create", "name", r.Name)

// 	// TODO(user): fill in your validation logic upon object creation.
// 	return nil
// }

// // ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
// func (r *VirtualService) ValidateUpdate(old runtime.Object) error {
// 	virtualservicelog.Info("validate update", "name", r.Name)

// 	// TODO(user): fill in your validation logic upon object update.
// 	return nil
// }

// // ValidateDelete implements webhook.Validator so a webhook will be registered for the type
// func (r *VirtualService) ValidateDelete() error {
// 	virtualservicelog.Info("validate delete", "name", r.Name)

// 	// TODO(user): fill in your validation logic upon object deletion.
// 	return nil
// }
