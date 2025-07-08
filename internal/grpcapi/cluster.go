package grpcapi

import (
	"context"
	"sort"

	"connectrpc.com/connect"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/cluster/v1"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/cluster/v1/clusterv1connect"
)

type ClusterStore struct {
	store *store.Store
	clusterv1connect.ClusterStoreServiceHandler
}

func NewClusterStore(s *store.Store) *ClusterStore {
	return &ClusterStore{
		store: s,
	}
}

func (s *ClusterStore) ListClusters(context.Context, *connect.Request[v1.ListClustersRequest]) (*connect.Response[v1.ListClustersResponse], error) {
	m := s.store.MapClusters()
	list := make([]*v1.ClusterListItem, 0, len(m))
	for _, v := range m {
		item := &v1.ClusterListItem{
			Uid:  string(v.UID),
			Name: v.Name,
		}
		list = append(list, item)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return connect.NewResponse(&v1.ListClustersResponse{Items: list}), nil
}
