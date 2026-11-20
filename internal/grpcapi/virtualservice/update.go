package virtualservice

import (
	"context"
	"fmt"
	"maps"

	"connectrpc.com/connect"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/grpcapi"
	"github.com/kaasops/envoy-xds-controller/internal/xds/resbuilder"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *VirtualServiceStore) UpdateVirtualService(ctx context.Context, req *connect.Request[v1.UpdateVirtualServiceRequest]) (*connect.Response[v1.UpdateVirtualServiceResponse], error) {
	if err := s.validateUpdateVirtualServiceRequest(ctx, req); err != nil {
		return nil, err
	}
	// Status writes advance the API resource version without updating the UID index.
	indexed := s.store.GetVirtualServiceByUID(req.Msg.Uid)
	if indexed == nil || indexed.Namespace != s.targetNs {
		return nil, fmt.Errorf("virtual service uid '%s' not found", req.Msg.Uid)
	}
	key := client.ObjectKeyFromObject(indexed)
	var original *v1alpha1.VirtualService
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var live v1alpha1.VirtualService
		if err := s.apiReader.Get(ctx, key, &live); err != nil {
			return err
		}
		if string(live.UID) != req.Msg.Uid {
			return fmt.Errorf("virtual service uid '%s' not found", req.Msg.Uid)
		}
		if !live.IsEditable() {
			return fmt.Errorf("virtual service uid '%s' is not editable", req.Msg.Uid)
		}
		// Retry status conflicts without overwriting a concurrent edit.
		if original != nil && (live.Generation != original.Generation ||
			!live.DeletionTimestamp.Equal(original.DeletionTimestamp) ||
			!maps.Equal(live.Annotations, original.Annotations) || !maps.Equal(live.Labels, original.Labels)) {
			return connect.NewError(connect.CodeAborted, fmt.Errorf("virtual service %s changed during update", key))
		}
		if original == nil {
			original = live.DeepCopy()
		}
		return s.updateVirtualService(ctx, req.Msg, &live)
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.UpdateVirtualServiceResponse{}), nil
}

func (s *VirtualServiceStore) updateVirtualService(ctx context.Context, req *v1.UpdateVirtualServiceRequest, vs *v1alpha1.VirtualService) error {
	authorizer := grpcapi.GetAuthorizerFromContext(ctx)

	vs.SetNodeIDs(req.NodeIds)
	vs.Spec.ExtraFields = req.ExtraFields

	accessGroup := vs.GetAccessGroup()

	if err := s.processTemplate(
		ctx,
		accessGroup,
		req.TemplateUid,
		req.TemplateOptions,
		vs,
		authorizer,
	); err != nil {
		return err
	}

	if err := s.processListener(ctx, accessGroup, req.ListenerUid, vs, authorizer); err != nil {
		return err
	}

	if err := s.processVirtualHost(req.VirtualHost, vs); err != nil {
		return err
	}

	if err := s.processAccessLogConfigs(ctx, accessGroup, req.GetAccessLogConfigUids(), vs, authorizer); err != nil {
		return err
	}

	vs.SetDescription(req.Description)

	if len(req.AdditionalRouteUids) > 0 {
		vs.Spec.AdditionalRoutes = vs.Spec.AdditionalRoutes[:0]
		if err := s.processAdditionalRoutes(ctx, accessGroup, req.AdditionalRouteUids, vs, authorizer); err != nil {
			return err
		}
	} else {
		vs.Spec.AdditionalRoutes = nil
	}

	if len(req.AdditionalHttpFilterUids) > 0 {
		vs.Spec.AdditionalHttpFilters = vs.Spec.AdditionalHttpFilters[:0]
		if err := s.processAdditionalHTTPFilters(ctx, accessGroup, req.AdditionalHttpFilterUids, vs, authorizer); err != nil {
			return err
		}
	} else {
		vs.Spec.AdditionalHttpFilters = nil
	}

	if req.UseRemoteAddress != nil {
		vs.Spec.UseRemoteAddress = req.UseRemoteAddress
	}

	if _, err := resbuilder.BuildResources(vs, s.store); err != nil {
		return err
	}

	return s.client.Update(ctx, vs)
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
