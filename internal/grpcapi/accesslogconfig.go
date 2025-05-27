package grpcapi

import (
	"context"
	"sort"

	"connectrpc.com/connect"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/access_log_config/v1"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/access_log_config/v1/access_log_configv1connect"
)

type AccessLogConfigStore struct {
	store *store.Store
	access_log_configv1connect.AccessLogConfigStoreServiceHandler
}

func NewAccessLogConfigStore(s *store.Store) *AccessLogConfigStore {
	return &AccessLogConfigStore{
		store: s,
	}
}

func (s *AccessLogConfigStore) ListAccessLogConfigs(ctx context.Context, req *connect.Request[v1.ListAccessLogConfigsRequest]) (*connect.Response[v1.ListAccessLogConfigsResponse], error) {
	authorizer := GetAuthorizerFromContext(ctx)

	m := s.store.MapAccessLogs()
	list := make([]*v1.AccessLogConfigListItem, 0, len(m))
	for _, v := range m {
		accessLogAG := v.GetAccessGroup()
		if accessLogAG != req.Msg.AccessGroup && accessLogAG != GeneralAccessGroup {
			continue
		}
		item := &v1.AccessLogConfigListItem{
			Uid:         string(v.UID),
			Name:        v.Name,
			Description: v.GetDescription(),
		}
		isAllowed, err := authorizer.Authorize(accessLogAG, item.Name)
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
	return connect.NewResponse(&v1.ListAccessLogConfigsResponse{Items: list}), nil
}
