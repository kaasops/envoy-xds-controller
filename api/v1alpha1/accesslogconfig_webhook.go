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
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/pkg/options"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (alc *AccessLogConfig) Validate(ctx context.Context) error {
	// Validate AccessLogConfig spec
	if alc.Spec == nil {
		return errors.New(errors.AccessLogConfigCannotBeEmptyMessage)
	}

	accessLogConfigv3 := &accesslogv3.AccessLog{}
	if err := options.Unmarshaler.Unmarshal(alc.Spec.Raw, accessLogConfigv3); err != nil {
		return errors.Wrap(err, errors.UnmarshalMessage)
	}

	return nil
}

func (alc *AccessLogConfig) ValidateDelete(ctx context.Context, cl client.Client) error {
	virtualServices := &VirtualServiceList{}
	listOpts := []client.ListOption{client.InNamespace(alc.Namespace)}
	if err := cl.List(ctx, virtualServices, listOpts...); err != nil {
		return err
	}

	if len(virtualServices.Items) > 0 {
		vsNames := []string{}
		for _, vs := range virtualServices.Items {
			if vs.Spec.AccessLogConfig != nil {
				if vs.Spec.AccessLogConfig.Name == alc.Name {
					vsNames = append(vsNames, vs.Name)
					continue
				}
			}
		}
		if len(vsNames) > 0 {
			return errors.New(fmt.Sprintf("%v%+v", errors.AccessLogConfigDeleteUsedMessage, vsNames))
		}
	}

	return nil
}
