/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/kaasops/envoy-xds-controller/pkg/utils/hash"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (vs *VirtualService) SetError(ctx context.Context, cl client.Client, msg string) error {
	if vs.Status.Error != nil && *vs.Status.Error == msg {
		return nil
	}
	vs.Status.Error = &msg

	valid := false
	vs.Status.Valid = &valid

	return cl.Status().Update(ctx, vs.DeepCopy())
}

func (vs *VirtualService) SetDomainsStatus(ctx context.Context, cl client.Client, domainsWithErrors map[string]string) error {
	if vs.Status.Domains != nil {
		if reflect.DeepEqual(*vs.Status.Domains, domainsWithErrors) {
			return nil
		}
	}
	vs.Status.Domains = &domainsWithErrors

	return cl.Status().Update(ctx, vs.DeepCopy())
}

func (vs *VirtualService) SetValid(ctx context.Context, cl client.Client) error {
	if vs.Status.Valid != nil && *vs.Status.Valid {
		return nil
	}
	valid := true
	vs.Status.Valid = &valid
	vs.Status.Error = nil

	return cl.Status().Update(ctx, vs.DeepCopy())
}

func (vs *VirtualService) SetInvalid(ctx context.Context, cl client.Client) error {
	if vs.Status.Valid != nil && !*vs.Status.Valid {
		return nil
	}
	valid := false
	vs.Status.Valid = &valid

	return cl.Status().Update(ctx, vs.DeepCopy())
}

func (vs *VirtualService) SetLastAppliedHash(ctx context.Context, cl client.Client) error {
	hash, err := vs.getHash()
	if err != nil {
		return err
	}
	if vs.Status.LastAppliedHash != nil && *hash == *vs.Status.LastAppliedHash {
		return nil
	}
	vs.Status.LastAppliedHash = hash

	return cl.Status().Update(ctx, vs.DeepCopy())
}

func (vs *VirtualService) CheckHash() (bool, error) {
	hash, err := vs.getHash()
	if err != nil {
		return false, err
	}

	if vs.Status.LastAppliedHash != nil && *hash == *vs.Status.LastAppliedHash {
		return true, nil
	}

	return false, nil
}

func (vs *VirtualService) getHash() (*uint32, error) {
	specHash, err := json.Marshal(vs.Spec)
	if err != nil {
		return nil, err
	}
	annotationHash, err := json.Marshal(vs.Annotations)
	if err != nil {
		return nil, err
	}
	hash := hash.Get(specHash) + hash.Get(annotationHash)
	return &hash, nil
}
