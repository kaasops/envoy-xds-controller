package grpcapi

import (
	"context"
	"sort"

	"connectrpc.com/connect"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/node/v1"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/node/v1/nodev1connect"
)

type NodeStore struct {
	nodeSvc NodeService
	nodev1connect.NodeStoreServiceHandler
}

type NodeService interface {
	GetNodeIDs() []string
}

func NewNodeStore(svc NodeService) *NodeStore {
	return &NodeStore{nodeSvc: svc}
}

func (s *NodeStore) ListNodes(ctx context.Context, req *connect.Request[v1.ListNodesRequest]) (*connect.Response[v1.ListNodesResponse], error) {
	nodeIDs := s.nodeSvc.GetNodeIDs()

	list := make([]*v1.NodeListItem, 0, len(nodeIDs))
	authorizer := GetAuthorizerFromContext(ctx)

	accessGroup := req.Msg.AccessGroup
	if accessGroup == "" {
		accessGroup = GeneralAccessGroup
	}

	for _, v := range nodeIDs {
		item := &v1.NodeListItem{
			Id: v,
		}
		isAllowed, err := authorizer.Authorize(accessGroup, item.Id)
		if err != nil {
			return nil, err
		}
		if !isAllowed {
			continue
		}
		list = append(list, item)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Id < list[j].Id
	})
	return connect.NewResponse(&v1.ListNodesResponse{Items: list}), nil
}
