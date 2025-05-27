package grpcapi

import (
	"context"
	"sort"

	"connectrpc.com/connect"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/http_filter/v1"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/http_filter/v1/http_filterv1connect"
)

type HTTPFilterStore struct {
	store *store.Store
	http_filterv1connect.UnimplementedHTTPFilterStoreServiceHandler
}

func NewHTTPFilterStore(s *store.Store) *HTTPFilterStore {
	return &HTTPFilterStore{
		store: s,
	}
}

func (s *HTTPFilterStore) ListHTTPFilters(ctx context.Context, req *connect.Request[v1.ListHTTPFiltersRequest]) (*connect.Response[v1.ListHTTPFiltersResponse], error) {
	m := s.store.MapHTTPFilters()
	list := make([]*v1.HTTPFilterListItem, 0, len(m))

	authorizer := GetAuthorizerFromContext(ctx)

	for _, v := range m {
		httpFilterAG := v.GetAccessGroup()
		if httpFilterAG != req.Msg.AccessGroup && httpFilterAG != GeneralAccessGroup {
			continue
		}
		item := &v1.HTTPFilterListItem{
			Uid:         string(v.UID),
			Name:        v.Name,
			Description: v.GetDescription(),
		}
		isAllowed, err := authorizer.Authorize(httpFilterAG, item.Name)
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
	return connect.NewResponse(&v1.ListHTTPFiltersResponse{Items: list}), nil
}
