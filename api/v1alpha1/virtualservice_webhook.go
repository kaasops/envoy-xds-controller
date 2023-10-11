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
	"errors"
	"fmt"

	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	hcmv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"

	"google.golang.org/protobuf/encoding/protojson"
)

var (
	ErrUnmarshal = errors.New("can't unmarshal resource")

	// Virtual Host errors
	ErrVHCantBeEmpty = errors.New("virtualHost could not be empty")

	// Listener errors
	ErrListenerCantBeEmpty = errors.New("listener could not be empty")

	// TLSConfig errors
	ErrTlsConfigNotExist = errors.New("tls Config not set")
)

func (vs *VirtualService) Validate(
	ctx context.Context,
	unmarshaler *protojson.UnmarshalOptions,
) error {
	// Validate Virtual Host spec
	if vs.Spec.VirtualHost == nil {
		return ErrVHCantBeEmpty
	}
	vh := &routev3.VirtualHost{}
	if err := unmarshaler.Unmarshal(vs.Spec.VirtualHost.Raw, vh); err != nil {
		return fmt.Errorf("%w. %w", ErrUnmarshal, err)
	}

	// Check AccessLog spec
	if vs.Spec.AccessLog != nil {
		al := &accesslogv3.AccessLog{}
		if err := unmarshaler.Unmarshal(vs.Spec.AccessLog.Raw, al); err != nil {
			return fmt.Errorf("%w. %w", ErrUnmarshal, err)
		}
	}

	// Check HTTPFilters spec
	if vs.Spec.HTTPFilters != nil {
		for _, httpFilter := range vs.Spec.HTTPFilters {
			hf := &hcmv3.HttpFilter{}
			if err := unmarshaler.Unmarshal(httpFilter.Raw, hf); err != nil {
				return fmt.Errorf("%w. %w", ErrUnmarshal, err)
			}
		}
	}

	// Check listener exist
	if vs.Spec.Listener == nil {
		return ErrListenerCantBeEmpty
	}

	// Check TLSConfig
	if err := vs.Spec.TlsConfig.Validate(); err != nil {
		return err
	}

	return nil
}

func (tc *TlsConfig) Validate() error {
	if tc == nil {
		return ErrTlsConfigNotExist
	}

	return nil
}
