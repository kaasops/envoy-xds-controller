package virtualservice

import (
	"context"
	"fmt"

	"github.com/kaasops/envoy-xds-controller/internal/grpcapi"

	"connectrpc.com/connect"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	commonv1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/common/v1"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service/v1"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service/v1/virtual_servicev1connect"
	virtual_service_templatev1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service_template/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VirtualServiceStore struct {
	store    *store.Store
	client   client.Client
	targetNs string
	virtual_servicev1connect.UnimplementedVirtualServiceStoreServiceHandler
}

func NewVirtualServiceStore(s *store.Store, c client.Client, targetNs string) *VirtualServiceStore {
	return &VirtualServiceStore{
		store:    s,
		client:   c,
		targetNs: targetNs,
	}
}

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

func (s *VirtualServiceStore) GetVirtualService(_ context.Context, req *connect.Request[v1.GetVirtualServiceRequest]) (*connect.Response[v1.GetVirtualServiceResponse], error) {
	if req.Msg.Uid == "" {
		return nil, fmt.Errorf("uid is required")
	}
	vs := s.store.GetVirtualServiceByUID(req.Msg.Uid)
	if vs == nil {
		return nil, fmt.Errorf("virtual service uid '%s' not found", req.Msg.Uid)
	}
	resp := &v1.GetVirtualServiceResponse{
		Uid:         string(vs.UID),
		Name:        vs.GetLabelName(),
		NodeIds:     vs.GetNodeIDs(),
		AccessGroup: vs.GetAccessGroup(),
		IsEditable:  vs.IsEditable(),
		Description: vs.GetDescription(),
	}
	if vs.Spec.Template != nil {
		template := s.store.GetVirtualServiceTemplate(helpers.NamespacedName{
			Namespace: helpers.GetNamespace(vs.Spec.Template.Namespace, vs.Namespace),
			Name:      vs.Spec.Template.Name,
		})
		resp.Template = &commonv1.ResourceRef{
			Uid:  string(template.UID),
			Name: template.Name,
		}
		if len(vs.Spec.TemplateOptions) > 0 {
			resp.TemplateOptions = make([]*virtual_service_templatev1.TemplateOption, 0, len(vs.Spec.TemplateOptions))
			for _, opt := range vs.Spec.TemplateOptions {
				resp.TemplateOptions = append(resp.TemplateOptions, &virtual_service_templatev1.TemplateOption{
					Field:    opt.Field,
					Modifier: grpcapi.ParseModifierToTemplateOption(opt.Modifier),
				})
			}
		}
	}
	if vs.Spec.Listener != nil {
		listener := s.store.GetListener(helpers.NamespacedName{
			Namespace: helpers.GetNamespace(vs.Spec.Listener.Namespace, vs.Namespace),
			Name:      vs.Spec.Listener.Name,
		})
		resp.Listener = &commonv1.ResourceRef{
			Uid:  string(listener.UID),
			Name: listener.Name,
		}
	}
	if vs.Spec.VirtualHost != nil {
		virtualHost := &routev3.VirtualHost{}
		err := protoutil.Unmarshaler.Unmarshal(vs.Spec.VirtualHost.Raw, virtualHost)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal virtual host: %w", err)
		}
		resp.VirtualHost = &commonv1.VirtualHost{Domains: virtualHost.Domains}
	}
	if vs.Spec.AccessLogConfig != nil {
		alc := s.store.GetAccessLog(helpers.NamespacedName{
			Namespace: helpers.GetNamespace(vs.Spec.AccessLogConfig.Namespace, vs.Namespace),
			Name:      vs.Spec.AccessLogConfig.Name,
		})
		resp.AccessLog = &v1.GetVirtualServiceResponse_AccessLogConfig{AccessLogConfig: &commonv1.ResourceRef{
			Uid:  string(alc.UID),
			Name: alc.Name,
		}}
	}
	if vs.Spec.AdditionalRoutes != nil {
		resp.AdditionalRoutes = make([]*commonv1.ResourceRef, 0, len(vs.Spec.AdditionalRoutes))
		for _, route := range vs.Spec.AdditionalRoutes {
			r := s.store.GetRoute(helpers.NamespacedName{
				Namespace: helpers.GetNamespace(route.Namespace, vs.Namespace),
				Name:      route.Name,
			})
			resp.AdditionalRoutes = append(resp.AdditionalRoutes, &commonv1.ResourceRef{
				Uid:  string(r.UID),
				Name: r.Name,
			})
		}
	}
	if vs.Spec.AdditionalHttpFilters != nil {
		resp.AdditionalHttpFilters = make([]*commonv1.ResourceRef, 0, len(vs.Spec.AdditionalHttpFilters))
		for _, filter := range vs.Spec.AdditionalHttpFilters {
			f := s.store.GetHTTPFilter(helpers.NamespacedName{
				Namespace: helpers.GetNamespace(filter.Namespace, vs.Namespace),
				Name:      filter.Name,
			})
			resp.AdditionalHttpFilters = append(resp.AdditionalHttpFilters, &commonv1.ResourceRef{
				Uid:  string(f.UID),
				Name: f.Name,
			})
		}
	}
	if vs.Spec.UseRemoteAddress != nil {
		resp.UseRemoteAddress = vs.Spec.UseRemoteAddress
	}
	return connect.NewResponse(resp), nil
}
