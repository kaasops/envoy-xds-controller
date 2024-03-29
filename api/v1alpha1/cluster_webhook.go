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

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/pkg/options"
)

func (c *Cluster) Validate(ctx context.Context) error {
	// Validate Listener spec
	if c.Spec == nil {
		return errors.New(errors.ClusterCannotBeEmptyMessage)
	}

	clusterv3 := &clusterv3.Cluster{}
	if err := options.Unmarshaler.Unmarshal(c.Spec.Raw, clusterv3); err != nil {
		return errors.Wrap(err, errors.UnmarshalMessage)
	}

	return nil
}

// TODO: Add check cluster not used, when deleted
