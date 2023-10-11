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

	"google.golang.org/protobuf/encoding/protojson"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
)

// var (
// 	// Listener errors
// 	ErrListenerCantBeEmpty = errors.New("virtualHost could not be empty")
// )

func (c *Cluster) Validate(
	ctx context.Context,
	unmarshaler *protojson.UnmarshalOptions,
) error {
	// Validate Listener spec
	if c.Spec == nil {
		return ErrListenerCantBeEmpty
	}

	cluster := &clusterv3.Cluster{}
	if err := unmarshaler.Unmarshal(c.Spec.Raw, cluster); err != nil {
		return fmt.Errorf("%w. %w", ErrUnmarshal, err)
	}

	return nil
}
