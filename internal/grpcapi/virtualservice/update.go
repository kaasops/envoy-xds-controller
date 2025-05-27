package virtualservice

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/kaasops/envoy-xds-controller/internal/grpcapi"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service/v1"
)

func (s *VirtualServiceStore) UpdateVirtualService(ctx context.Context, req *connect.Request[v1.UpdateVirtualServiceRequest]) (*connect.Response[v1.UpdateVirtualServiceResponse], error) {
	if err := s.validateUpdateVirtualServiceRequest(ctx, req); err != nil {
		return nil, err
	}
	vs := s.store.GetVirtualServiceByUID(req.Msg.Uid)
	if vs == nil {
		return nil, fmt.Errorf("virtual service uid '%s' not found", req.Msg.Uid)
	}
	if !vs.IsEditable() {
		return nil, fmt.Errorf("virtual service uid '%s' is not editable", req.Msg.Uid)
	}

	authorizer := grpcapi.GetAuthorizerFromContext(ctx)

	vs.SetNodeIDs(req.Msg.NodeIds)
	vs.Namespace = s.targetNs

	accessGroup := vs.GetAccessGroup()

	if err := s.processTemplate(
		ctx,
		accessGroup,
		req.Msg.TemplateUid,
		req.Msg.TemplateOptions,
		vs,
		authorizer,
	); err != nil {
		return nil, err
	}

	if err := s.processListener(ctx, accessGroup, req.Msg.ListenerUid, vs, authorizer); err != nil {
		return nil, err
	}

	if err := s.processVirtualHost(req.Msg.VirtualHost, vs); err != nil {
		return nil, err
	}

	if err := s.processAccessLogConfig(ctx, accessGroup, req.Msg.GetAccessLogConfigUid(), vs, authorizer); err != nil {
		return nil, err
	}

	vs.SetDescription(req.Msg.Description)

	if len(req.Msg.AdditionalRouteUids) > 0 {
		vs.Spec.AdditionalRoutes = vs.Spec.AdditionalRoutes[:0]
		if err := s.processAdditionalRoutes(ctx, accessGroup, req.Msg.AdditionalRouteUids, vs, authorizer); err != nil {
			return nil, err
		}
	} else {
		vs.Spec.AdditionalRoutes = nil
	}

	if len(req.Msg.AdditionalHttpFilterUids) > 0 {
		vs.Spec.AdditionalHttpFilters = vs.Spec.AdditionalHttpFilters[:0]
		if err := s.processAdditionalHTTPFilters(ctx, accessGroup, req.Msg.AdditionalHttpFilterUids, vs, authorizer); err != nil {
			return nil, err
		}
	} else {
		vs.Spec.AdditionalHttpFilters = nil
	}

	if req.Msg.UseRemoteAddress != nil {
		vs.Spec.UseRemoteAddress = req.Msg.UseRemoteAddress
	}

	if _, err := resbuilder.BuildResources(vs, s.store); err != nil {
		return nil, err
	}

	if err := s.client.Update(ctx, vs); err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.UpdateVirtualServiceResponse{}), nil
}

func (s *VirtualServiceStore) validateUpdateVirtualServiceRequest(_ context.Context, req *connect.Request[v1.UpdateVirtualServiceRequest]) error {
	if req == nil || req.Msg == nil {
		return fmt.Errorf("request or message cannot be nil")
	}
	if req.Msg.Uid == "" {
		return fmt.Errorf("uid is required")
	}
	if req.Msg.TemplateUid == "" {
		return fmt.Errorf("template uid is required")
	}
	if req.Msg.VirtualHost != nil && len(req.Msg.VirtualHost.Domains) > 0 {
		for _, domain := range req.Msg.VirtualHost.Domains {
			if err := validateDomain(domain); err != nil {
				return fmt.Errorf("domain %s is invalid: %w", domain, err)
			}
		}
	}
	return nil
}
