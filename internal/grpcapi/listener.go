package grpcapi

import (
	"context"
	"sort"

	"connectrpc.com/connect"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/listener/v1"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/listener/v1/listenerv1connect"
)

type ListenerStore struct {
	store *store.Store
	listenerv1connect.ListenerStoreServiceHandler
}

func NewListenerStore(s *store.Store) *ListenerStore {
	return &ListenerStore{
		store: s,
	}
}

func (s *ListenerStore) ListListeners(ctx context.Context, req *connect.Request[v1.ListListenersRequest]) (*connect.Response[v1.ListListenersResponse], error) {
	m := s.store.MapListeners()
	list := make([]*v1.ListenerListItem, 0, len(m))
	authorizer := GetAuthorizerFromContext(ctx)

	for _, v := range m {
		listenerAG := v.GetAccessGroup()
		if listenerAG != req.Msg.AccessGroup && listenerAG != GeneralAccessGroup {
			continue
		}
		item := &v1.ListenerListItem{
			Uid:         string(v.UID),
			Name:        v.Name,
			Type:        listenerType(v),
			Description: v.GetDescription(),
		}
		isAllowed, err := authorizer.Authorize(listenerAG, item.Name)
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
	return connect.NewResponse(&v1.ListListenersResponse{Items: list}), nil
}

func listenerType(l *v1alpha1.Listener) v1.ListenerType {
	lv3 := listenerv3.Listener{}
	if err := protoutil.Unmarshaler.Unmarshal(l.Spec.Raw, &lv3); err != nil {
		return v1.ListenerType_LISTENER_TYPE_UNSPECIFIED
	}
	switch {
	case isTCPType(&lv3):
		return v1.ListenerType_LISTENER_TYPE_TCP
	case isHTTPSType(&lv3):
		return v1.ListenerType_LISTENER_TYPE_HTTPS
	default:
		return v1.ListenerType_LISTENER_TYPE_HTTP
	}
}

func isTCPType(l *listenerv3.Listener) bool {
	filterChains := l.GetFilterChains()
	if len(filterChains) == 0 {
		return false
	}
	for _, fc := range filterChains {
		if filters := fc.GetFilters(); len(filters) > 0 {
			for _, f := range filters {
				if tc := f.GetTypedConfig(); tc != nil {
					if tc.TypeUrl == "type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy" {
						return true
					}
				}
			}
		}
	}
	return false
}

func isHTTPSType(l *listenerv3.Listener) bool {
	filters := l.GetListenerFilters()
	if len(filters) == 0 {
		return false
	}
	for _, filter := range filters {
		if tc := filter.GetTypedConfig(); tc != nil {
			if tc.TypeUrl == "type.googleapis.com/envoy.extensions.filters.listener.tls_inspector.v3.TlsInspector" {
				return true
			}
		}
	}
	return false
}
