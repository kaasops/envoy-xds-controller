package grpcapi

import (
	"context"
	"sort"

	"connectrpc.com/connect"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/cluster/v1"
)

type ClusterStore struct {
	store store.Store
}

func NewClusterStore(s store.Store) *ClusterStore {
	return &ClusterStore{
		store: s,
	}
}

func (s *ClusterStore) ListCluster(ctx context.Context, _ *connect.Request[v1.ListClustersRequest]) (*connect.Response[v1.ListClustersResponse], error) {
	m := s.store.MapClusters()
	list := make([]*v1.ClusterListItem, 0, len(m))
	authorizer := GetAuthorizerFromContext(ctx)
	for _, v := range m {
		group := v.Labels[v1alpha1.LabelAccessGroup]
		if group == "" {
			group = GeneralAccessGroup
		}
		allowed, err := authorizer.Authorize(group, v.Name)
		if err != nil {
			return nil, err
		}
		if !allowed {
			continue
		}
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
