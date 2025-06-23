package virtualservice

import (
	"github.com/kaasops/envoy-xds-controller/internal/store"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service/v1/virtual_servicev1connect"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VirtualServiceStore struct {
	store    *store.Store
	client   client.Client
	targetNs string
	virtual_servicev1connect.UnimplementedVirtualServiceStoreServiceHandler
}

func NewVirtualServiceStore(s *store.Store, c client.Client, targetNs string) *VirtualServiceStore {
	return &VirtualServiceStore{
		store:    s,
		client:   c,
		targetNs: targetNs,
	}
}
