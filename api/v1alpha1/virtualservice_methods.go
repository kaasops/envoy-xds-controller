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
	"strings"

	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"github.com/kaasops/envoy-xds-controller/pkg/utils/hash"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (vs *VirtualService) SetError(ctx context.Context, cl client.Client, msg Message) error {
	if !vs.validAlredySet() && vs.messageAlredySet(msg) {
		return nil
	}

	vs.Status.Message = msg
	vs.Status.Valid = false

	return cl.Status().Update(ctx, vs.DeepCopy())
}

func (vs *VirtualService) SetValid(ctx context.Context, cl client.Client, msg Message) error {
	if vs.validAlredySet() && vs.messageAlredySet(msg) {
		return nil
	}

	vs.Status.Message = msg
	vs.Status.Valid = true

	return cl.Status().Update(ctx, vs.DeepCopy())
}

func (vs *VirtualService) SetValidWithUsedSecrets(ctx context.Context, cl client.Client, secrets []string, msg Message) error {
	err := vs.setUsedSecrets(secrets)
	if err != nil {
		return err
	}

	vs.Status.Message = msg
	vs.Status.Valid = true

	return cl.Status().Update(ctx, vs.DeepCopy())
}

func (vs *VirtualService) setUsedSecrets(secrets []string) error {
	usedSecrets := []ResourceRef{}

	for _, s := range secrets {
		splitS := strings.Split(s, "/")

		if len(splitS) != 2 {
			return errors.New("something go wrong, when trying to get secret namespace and name")
		}

		usedSecret := ResourceRef{
			Name:      splitS[1],
			Namespace: &splitS[0],
		}

		usedSecrets = append(usedSecrets, usedSecret)
	}

	vs.Status.UsedSecrets = usedSecrets

	return nil
}

// func (vs *VirtualService) SetInvalid(ctx context.Context, cl client.Client) error {
// 	if vs.Status.Valid != nil && !*vs.Status.Valid {
// 		return nil
// 	}
// 	valid := false
// 	vs.Status.Valid = &valid

// 	vs.SetLastAppliedHash(ctx, cl)

// 	return cl.Status().Update(ctx, vs.DeepCopy())
// }

func (vs *VirtualService) SetLastAppliedHash(ctx context.Context, cl client.Client) error {
	hash, err := vs.getHash()
	if err != nil {
		return err
	}
	if vs.Status.LastAppliedHash != nil && *hash == *vs.Status.LastAppliedHash {
		return nil
	}
	vs.Status.LastAppliedHash = hash

	return nil
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

func (vs *VirtualService) validAlredySet() bool {
	return vs.Status.Valid
}

func (vs *VirtualService) messageAlredySet(msg Message) bool {
	if vs.Status.Message == msg {
		return true
	}

	return false
}

/**
	TlsConfig Methods
**/

func (tc *TlsConfig) GetTLSType() (string, error) {
	if tc.SecretRef != nil {
		if tc.AutoDiscovery != nil {
			return "", errors.New(errors.ManyParamMessage)
		}
		return SecretRefType, nil
	}

	if tc.AutoDiscovery != nil {
		return AutoDiscoveryType, nil
	}

	return "", errors.New(errors.ZeroParamMessage)
}
