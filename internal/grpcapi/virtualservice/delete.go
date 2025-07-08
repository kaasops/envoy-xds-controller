package virtualservice

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/kaasops/envoy-xds-controller/internal/grpcapi"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service/v1"
)

func (s *VirtualServiceStore) DeleteVirtualService(ctx context.Context, req *connect.Request[v1.DeleteVirtualServiceRequest]) (*connect.Response[v1.DeleteVirtualServiceResponse], error) {
	if req.Msg.Uid == "" {
		return nil, fmt.Errorf("uid is required")
	}
	vs := s.store.GetVirtualServiceByUID(req.Msg.Uid)
	if vs == nil {
		return nil, fmt.Errorf("virtual service uid '%s' not found", req.Msg.Uid)
	}
	if !vs.IsEditable() {
		return nil, fmt.Errorf("virtual service uid '%s' is not editable", req.Msg.Uid)
	}
	authorizer := grpcapi.GetAuthorizerFromContext(ctx)
	ok, err := authorizer.Authorize(vs.GetAccessGroup(), vs.Name)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("user is not authorized to delete virtual service")
	}
	if err := s.client.Delete(ctx, vs); err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.DeleteVirtualServiceResponse{}), nil
}
