package client

import (
	"context"
	"fmt"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/utils/k8s"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"slices"
)

type VirtualServiceClient struct {
	K8sClient client.Client
}

func NewVirtualServiceClient(k8sClient client.Client) *VirtualServiceClient {
	return &VirtualServiceClient{
		K8sClient: k8sClient,
	}
}

func (c *VirtualServiceClient) GetAllVirtualServices(ctx context.Context, namespace string) ([]string, error) {
	vsList := &v1alpha1.VirtualServiceList{}
	options := []client.ListOption{}

	if namespace != "" {
		options = append(options, client.InNamespace(namespace))
	}

	if err := c.K8sClient.List(ctx, vsList, options...); err != nil {
		return nil, fmt.Errorf("failed to list VirtualServices: %w", err)
	}

	names := []string{}
	for _, vs := range vsList.Items {
		names = append(names, vs.Name)
	}

	return names, nil
}

func (c *VirtualServiceClient) GetAllVirtualServicesWithWrongState(ctx context.Context, namespace string) ([]string, error) {
	vsList := &v1alpha1.VirtualServiceList{}
	options := []client.ListOption{}

	if namespace != "" {
		options = append(options, client.InNamespace(namespace))
	}

	if err := c.K8sClient.List(ctx, vsList, options...); err != nil {
		return nil, fmt.Errorf("failed to list VirtualServices: %w", err)
	}

	wrongVsNames := []string{}

	for _, vs := range vsList.Items {
		if (vs.Status.Valid != nil && !*vs.Status.Valid) || vs.Status.Error != nil {
			wrongVsNames = append(wrongVsNames, vs.Name)
		}
	}
	return wrongVsNames, nil
}

func (c *VirtualServiceClient) GetVirtualService(ctx context.Context, name, namespace string) (*v1alpha1.VirtualService, error) {
	vs := &v1alpha1.VirtualService{}
	if err := c.K8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, vs); err != nil {
		return nil, fmt.Errorf("failed to get VirtualService: %w", err)
	}
	return vs, nil
}

func (c *VirtualServiceClient) GetVirtualServiceByNameAndNodeId(ctx context.Context, name, nodeId, namespace string) (*v1alpha1.VirtualService, error) {
	vsList := &v1alpha1.VirtualServiceList{}

	if err := c.K8sClient.List(ctx, vsList, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("failed to list VirtualServices: %w", err)
	}

	for _, vs := range vsList.Items {
		if vs.Name == name {
			nodeIDs := k8s.NodeIDs(&vs)
			if slices.Contains(nodeIDs, nodeId) {
				return &vs, nil
			}
		}
	}

	return nil, fmt.Errorf("VirtualService not found")
}

func (c *VirtualServiceClient) CreateVirtualService(ctx context.Context, vs *v1alpha1.VirtualService) error {
	if err := c.K8sClient.Create(ctx, vs); err != nil {
		return fmt.Errorf("failed to create VirtualService: %w", err)
	}
	return nil
}

func (c *VirtualServiceClient) UpdateVirtualService(ctx context.Context, vs *v1alpha1.VirtualService) error {
	if err := c.K8sClient.Update(ctx, vs); err != nil {
		return fmt.Errorf("failed to update VirtualService: %w", err)
	}
	return nil
}

func (c *VirtualServiceClient) DeleteVirtualService(ctx context.Context, name, namespace string) error {
	vs := &v1alpha1.VirtualService{}
	if err := c.K8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, vs); err != nil {
		return fmt.Errorf("failed to get VirtualService: %w", err)
	}

	if err := c.K8sClient.Delete(ctx, vs); err != nil {
		return fmt.Errorf("failed to delete VirtualService: %w", err)
	}
	return nil
}
