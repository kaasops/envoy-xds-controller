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

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/pkg/options"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (l *Listener) Validate(ctx context.Context) error {
	// Validate Listener spec
	if l.Spec == nil {
		return errors.New(errors.ListenerCannotBeEmptyMessage)
	}

	listenerv3 := &listenerv3.Listener{}
	if err := options.Unmarshaler.Unmarshal(l.Spec.Raw, listenerv3); err != nil {
		return errors.Wrap(err, errors.UnmarshalMessage)
	}

	return nil
}

// ValidateDelete - check if the listener used by any virtual service
func (l *Listener) ValidateDelete(ctx context.Context, cl client.Client) error {
	virtualServices := &VirtualServiceList{}
	listOpts := []client.ListOption{
		client.InNamespace(l.Namespace),
		client.MatchingFields{options.VirtualServiceListenerNameField: l.Name},
	}
	if err := cl.List(ctx, virtualServices, listOpts...); err != nil {
		return err
	}

	if len(virtualServices.Items) > 0 {
		vsNames := make([]string, 0, len(virtualServices.Items))
		for _, vs := range virtualServices.Items {
			vsNames = append(vsNames, vs.Name)
		}
		return errors.New(fmt.Sprintf("listener is used in Virtual Services: %+v", vsNames))
	}

	return nil
}
