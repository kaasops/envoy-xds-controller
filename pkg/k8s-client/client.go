package client

import (
	"context"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VirtualServiceClient struct {
	K8sClient client.Client
}

func NewVirtualServiceClient(k8sClient client.Client) *VirtualServiceClient {
	return &VirtualServiceClient{
		K8sClient: k8sClient,
	}
}

func (c *VirtualServiceClient) GetAllVirtualServices(ctx context.Context) ([]string, error) {
	var virtualServices v1alpha1.VirtualServiceList
	if err := c.K8sClient.List(ctx, &virtualServices); err != nil {
		return nil, errors.Wrap(err, "failed to list VirtualServices")
	}
	names := make([]string, 0, len(virtualServices.Items))
	for _, vs := range virtualServices.Items {
		names = append(names, vs.Name)
	}
	return names, nil
}

func (c *VirtualServiceClient) GetAllVirtualServicesWithWrongState(ctx context.Context) ([]string, error) {
	var virtualServices v1alpha1.VirtualServiceList
	if err := c.K8sClient.List(ctx, &virtualServices); err != nil {
		return nil, errors.Wrap(err, "failed to list VirtualServices")
	}
	var wrongVsNames []string
	for _, vs := range virtualServices.Items {
		if vs.Status.Valid != nil && !*vs.Status.Valid || vs.Status.Error != nil {
			wrongVsNames = append(wrongVsNames, vs.Name)
		}
	}
	return wrongVsNames, nil
}

func (c *VirtualServiceClient) GetVirtualService(ctx context.Context, name, namespace string) (*v1alpha1.VirtualService, error) {
	vs := &v1alpha1.VirtualService{}
	if err := c.K8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, vs); err != nil {
		return nil, errors.Wrap(err, "failed to get VirtualService")
	}
	return vs, nil
}

func (c *VirtualServiceClient) GetVirtualServiceByNameAndNodeId(ctx context.Context, name, namespace string) (*v1alpha1.VirtualService, error) {
	vs := &v1alpha1.VirtualService{}
	return vs, nil
}

func (c *VirtualServiceClient) CreateVirtualService(ctx context.Context, vs *v1alpha1.VirtualService) error {
	if err := c.K8sClient.Create(ctx, vs); err != nil {
		return errors.Wrap(err, "failed to create VirtualService")
	}
	return nil
}

func (c *VirtualServiceClient) UpdateVirtualService(ctx context.Context, vs *v1alpha1.VirtualService) error {
	if err := c.K8sClient.Update(ctx, vs); err != nil {
		return errors.Wrap(err, "failed to update VirtualService")
	}
	return nil
}

func (c *VirtualServiceClient) DeleteVirtualService(ctx context.Context, vs *v1alpha1.VirtualService) error {
	if err := c.K8sClient.Delete(ctx, vs); err != nil {
		return errors.Wrap(err, "failed to delete VirtualService")
	}
	return nil
}
