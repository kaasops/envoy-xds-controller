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

	"google.golang.org/protobuf/encoding/protojson"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"

	"github.com/kaasops/envoy-xds-controller/pkg/errors"
)

func (r *Route) Validate(
	ctx context.Context,
	unmarshaler *protojson.UnmarshalOptions,
) error {
	// Validate Route spec
	if r.Spec == nil {
		return errors.New(errors.HTTPFilterCannotBeEmptyMessage)
	}

	for _, route := range r.Spec {
		routev3 := &routev3.Route{}
		if err := unmarshaler.Unmarshal(route.Raw, routev3); err != nil {
			return errors.Wrap(err, errors.UnmarshalMessage)
		}
	}

	return nil
}
