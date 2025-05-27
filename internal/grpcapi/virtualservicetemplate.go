package grpcapi

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"

	"connectrpc.com/connect"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service_template/v1"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service_template/v1/virtual_service_templatev1connect"
	"k8s.io/apimachinery/pkg/runtime"
)

type VirtualServiceTemplateStore struct {
	store *store.Store
	virtual_service_templatev1connect.UnimplementedVirtualServiceTemplateStoreServiceHandler
}

func NewVirtualServiceTemplateStore(s *store.Store) *VirtualServiceTemplateStore {
	return &VirtualServiceTemplateStore{
		store: s,
	}
}

func (s *VirtualServiceTemplateStore) ListVirtualServiceTemplates(ctx context.Context, req *connect.Request[v1.ListVirtualServiceTemplatesRequest]) (*connect.Response[v1.ListVirtualServiceTemplatesResponse], error) {
	m := s.store.MapVirtualServiceTemplates()
	list := make([]*v1.VirtualServiceTemplateListItem, 0, len(m))
	authorizer := GetAuthorizerFromContext(ctx)

	accessGroup := req.Msg.AccessGroup
	if accessGroup == "" {
		accessGroup = GeneralAccessGroup
	}

	for _, v := range m {
		item := &v1.VirtualServiceTemplateListItem{
			Uid:         string(v.UID),
			Name:        v.Name,
			Description: v.GetDescription(),
		}
		isAllowed, err := authorizer.Authorize(accessGroup, item.Name)
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
	return connect.NewResponse(&v1.ListVirtualServiceTemplatesResponse{Items: list}), nil
}

func (s *VirtualServiceTemplateStore) FillTemplate(ctx context.Context, req *connect.Request[v1.FillTemplateRequest]) (*connect.Response[v1.FillTemplateResponse], error) {
	authorizer := GetAuthorizerFromContext(ctx)
	if req.Msg.TemplateUid == "" {
		return nil, fmt.Errorf("template uid is required")
	}
	template := s.store.GetVirtualServiceTemplateByUID(req.Msg.TemplateUid)
	if template == nil {
		return nil, fmt.Errorf("template not found")
	}
	ok, err := authorizer.Authorize(template.GetAccessGroup(), template.Name)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("access group '%s' is not allowed to fill template '%s'", template.GetAccessGroup(), template.Name)
	}

	vs := &v1alpha1.VirtualService{}
	vs.Spec.Template = &v1alpha1.ResourceRef{
		Name:      template.Name,
		Namespace: &template.Namespace,
	}
	if req.Msg.ListenerUid != "" {
		listener := s.store.GetListenerByUID(req.Msg.ListenerUid)
		if listener == nil {
			return nil, fmt.Errorf("listener uid '%s' not found", req.Msg.ListenerUid)
		}
		vs.Spec.Listener = &v1alpha1.ResourceRef{
			Name:      listener.Name,
			Namespace: &listener.Namespace,
		}
	}

	if req.Msg.Name != "" {
		vs.Name = req.Msg.Name
	} else {
		vs.Name = template.Name + "-vs"
	}

	if req.Msg.VirtualHost != nil {
		virtualHost := &routev3.VirtualHost{
			Name:    vs.Name + "-virtual-host",
			Domains: req.Msg.VirtualHost.Domains,
		}
		vhData, err := protoutil.Marshaler.Marshal(virtualHost)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal virtual host: %w", err)
		}
		vs.Spec.VirtualHost = &runtime.RawExtension{Raw: vhData}
	}

	if req.Msg.AccessLogConfig != nil {
		if alcUID := req.Msg.GetAccessLogConfigUid(); alcUID != "" {
			alc := s.store.GetAccessLogByUID(alcUID)
			if alc == nil {
				return nil, fmt.Errorf("access log config uid '%s' not found", alcUID)
			}
			vs.Spec.AccessLogConfig = &v1alpha1.ResourceRef{
				Name:      alc.Name,
				Namespace: &alc.Namespace,
			}
		}
	}

	if len(req.Msg.AdditionalRouteUids) > 0 {
		for _, uid := range req.Msg.AdditionalRouteUids {
			route := s.store.GetRouteByUID(uid)
			if route == nil {
				return nil, fmt.Errorf("route uid '%s' not found", uid)
			}
			vs.Spec.AdditionalRoutes = append(vs.Spec.AdditionalRoutes, &v1alpha1.ResourceRef{
				Name:      route.Name,
				Namespace: &route.Namespace,
			})
		}
	}

	if len(req.Msg.AdditionalHttpFilterUids) > 0 {
		for _, uid := range req.Msg.AdditionalHttpFilterUids {
			filter := s.store.GetHTTPFilterByUID(uid)
			if filter == nil {
				return nil, fmt.Errorf("http filter uid '%s' not found", uid)
			}
			vs.Spec.AdditionalHttpFilters = append(vs.Spec.AdditionalHttpFilters, &v1alpha1.ResourceRef{
				Name:      filter.Name,
				Namespace: &filter.Namespace,
			})
		}
	}

	if req.Msg.UseRemoteAddress != nil {
		vs.Spec.UseRemoteAddress = req.Msg.UseRemoteAddress
	}

	if len(req.Msg.TemplateOptions) > 0 {
		tOpts := make([]v1alpha1.TemplateOpts, 0, len(req.Msg.TemplateOptions))
		for _, opt := range req.Msg.TemplateOptions {
			tOpts = append(tOpts, v1alpha1.TemplateOpts{
				Field:    opt.Field,
				Modifier: ParseTemplateOptionModifier(opt.Modifier),
			})
		}
		vs.Spec.TemplateOptions = tOpts
	}

	if err := vs.FillFromTemplate(template, vs.Spec.TemplateOptions...); err != nil {
		return nil, err
	}

	data, err := json.Marshal(vs.Spec)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&v1.FillTemplateResponse{Raw: string(data)}), nil
}
