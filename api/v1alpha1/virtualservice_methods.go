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
	"fmt"
	"github.com/kaasops/envoy-xds-controller/pkg/utils/k8s"
	"slices"
	"strings"

	"github.com/kaasops/envoy-xds-controller/pkg/errors"
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

func (vs *VirtualService) SetValid(ctx context.Context, cl client.Client) error {
	if vs.Status.Valid != nil && *vs.Status.Valid {
		return nil
	}
	valid := true
	vs.Status.Valid = &valid
	vs.Status.Error = nil

	return cl.Status().Update(ctx, vs.DeepCopy())
}

func (vs *VirtualService) SetValidWithUsedSecrets(ctx context.Context, cl client.Client, secrets []string) error {
	err := vs.setUsedSecrets(secrets)
	if err != nil {
		return err
	}

	if vs.Status.Valid != nil && *vs.Status.Valid {
		return cl.Status().Update(ctx, vs.DeepCopy())
	}

	valid := true
	vs.Status.Valid = &valid
	vs.Status.Error = nil

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

func (vs *VirtualService) SetInvalid(ctx context.Context, cl client.Client) error {
	if vs.Status.Valid != nil && !*vs.Status.Valid {
		return nil
	}
	valid := false
	vs.Status.Valid = &valid

	vs.SetLastAppliedHash(ctx, cl)

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

func (vs *VirtualService) GetAll(ctx context.Context, namespace string, cl client.Client) ([]string, error) {
	vsList := &VirtualServiceList{}
	options := []client.ListOption{}

	if namespace != "" {
		options = append(options, client.InNamespace(namespace))
	}

	if err := cl.List(ctx, vsList, options...); err != nil {
		return nil, fmt.Errorf("failed to list VirtualServices: %w", err)
	}

	names := []string{}
	for _, item := range vsList.Items {
		names = append(names, item.Name)
	}

	return names, nil
}

func (vs *VirtualService) GetAllWithWrongState(ctx context.Context, namespace string, cl client.Client) ([]string, error) {
	vsList := &VirtualServiceList{}
	options := []client.ListOption{}

	if namespace != "" {
		options = append(options, client.InNamespace(namespace))
	}

	if err := cl.List(ctx, vsList, options...); err != nil {
		return nil, fmt.Errorf("failed to list VirtualServices: %w", err)
	}

	wrongVsNames := []string{}

	for _, item := range vsList.Items {
		if (item.Status.Valid != nil && !*item.Status.Valid) || item.Status.Error != nil {
			wrongVsNames = append(wrongVsNames, item.Name)
		}
	}
	return wrongVsNames, nil
}

func (vs *VirtualService) Get(ctx context.Context, name, namespace string, cl client.Client) (*VirtualService, error) {
	item := &VirtualService{}
	if err := cl.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, item); err != nil {
		return nil, fmt.Errorf("failed to get VirtualService: %w", err)
	}
	return item, nil
}

func (vs *VirtualService) GetByNameAndNodeId(ctx context.Context, name, nodeId, namespace string, cl client.Client) (*VirtualService, error) {
	vsList := &VirtualServiceList{}

	if err := cl.List(ctx, vsList, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("failed to list VirtualServices: %w", err)
	}

	for _, item := range vsList.Items {
		if item.Name == name {
			nodeIDs := k8s.NodeIDs(&item)
			if slices.Contains(nodeIDs, nodeId) {
				return &item, nil
			}
		}
	}

	return nil, fmt.Errorf("VirtualService not found")
}

func (vs *VirtualService) CreateVirtualService(ctx context.Context, item *VirtualService, cl client.Client) error {
	if err := cl.Create(ctx, item); err != nil {
		return fmt.Errorf("failed to create VirtualService: %w", err)
	}
	return nil
}

func (vs *VirtualService) UpdateVirtualService(ctx context.Context, item *VirtualService, cl client.Client) error {
	if err := cl.Update(ctx, item); err != nil {
		return fmt.Errorf("failed to update VirtualService: %w", err)
	}
	return nil
}

func (vs *VirtualService) DeleteVirtualService(ctx context.Context, name, namespace string, cl client.Client) error {
	item := &VirtualService{}
	if err := cl.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, item); err != nil {
		return fmt.Errorf("failed to get VirtualService: %w", err)
	}

	if err := cl.Delete(ctx, item); err != nil {
		return fmt.Errorf("failed to delete VirtualService: %w", err)
	}
	return nil
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
