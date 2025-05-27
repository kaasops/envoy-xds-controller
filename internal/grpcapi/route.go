package grpcapi

import (
	"context"
	"sort"

	"connectrpc.com/connect"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/route/v1"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/route/v1/routev1connect"
)

type RouteStore struct {
	store *store.Store
	routev1connect.RouteStoreServiceHandler
}

func NewRouteStore(s *store.Store) *RouteStore {
	return &RouteStore{
		store: s,
	}
}

func (s *RouteStore) ListRoutes(ctx context.Context, req *connect.Request[v1.ListRoutesRequest]) (*connect.Response[v1.ListRoutesResponse], error) {
	m := s.store.MapRoutes()
	list := make([]*v1.RouteListItem, 0, len(m))
	authorizer := GetAuthorizerFromContext(ctx)

	for _, v := range m {
		routeAG := v.GetAccessGroup()
		if routeAG != req.Msg.AccessGroup && routeAG != GeneralAccessGroup {
			continue
		}
		item := &v1.RouteListItem{
			Uid:         string(v.UID),
			Name:        v.Name,
			Description: v.GetDescription(),
		}
		isAllowed, err := authorizer.Authorize(routeAG, item.Name)
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
	return connect.NewResponse(&v1.ListRoutesResponse{Items: list}), nil
}
