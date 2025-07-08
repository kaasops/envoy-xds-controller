package grpcapi

import (
	"context"
	"sort"

	"connectrpc.com/connect"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/policy/v1"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/policy/v1/policyv1connect"
)

type PolicyStore struct {
	store *store.Store
	policyv1connect.PolicyStoreServiceHandler
}

func NewPolicyStore(s *store.Store) *PolicyStore {
	return &PolicyStore{
		store: s,
	}
}

func (s *PolicyStore) ListPolicies(ctx context.Context, _ *connect.Request[v1.ListPoliciesRequest]) (*connect.Response[v1.ListPoliciesResponse], error) {
	m := s.store.MapPolicies()
	list := make([]*v1.PolicyListItem, 0, len(m))
	authorizer := GetAuthorizerFromContext(ctx)
	for _, v := range m {
		item := &v1.PolicyListItem{
			Uid:         string(v.UID),
			Name:        v.Name,
			Description: v.GetDescription(),
		}
		isAllowed, err := authorizer.Authorize(GeneralAccessGroup, item.Name)
		if err != nil {
			return nil, err
		}
		if !isAllowed {
			continue
		}
		list = append(list, item)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return connect.NewResponse(&v1.ListPoliciesResponse{Items: list}), nil
}
