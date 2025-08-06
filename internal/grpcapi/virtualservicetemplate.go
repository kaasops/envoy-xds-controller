package grpcapi

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/kaasops/envoy-xds-controller/internal/helpers"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"

	"connectrpc.com/connect"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/store"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service_template/v1"
	"github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service_template/v1/virtual_service_templatev1connect"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
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
			Raw:         string(v.Raw()),
		}

		if len(v.Spec.ExtraFields) > 0 {
			item.ExtraFields = make([]*v1.ExtraField, 0, len(v.Spec.ExtraFields))
			for _, field := range v.Spec.ExtraFields {
				item.ExtraFields = append(item.ExtraFields, &v1.ExtraField{
					Name:        field.Name,
					Type:        field.Type,
					Description: field.Description,
					Default:     field.Default,
					Enum:        field.Enum,
					Required:    field.Required,
				})
			}
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

	accessLogConfigUIDs := req.Msg.GetAccessLogConfigUids()
	if accessLogConfigUIDs != nil && len(accessLogConfigUIDs.GetUids()) > 0 {
		vs.Spec.AccessLogConfigs = make([]*v1alpha1.ResourceRef, 0, len(accessLogConfigUIDs.GetUids()))
		for _, alcUID := range accessLogConfigUIDs.GetUids() {
			if alcUID != "" {
				alc := s.store.GetAccessLogByUID(alcUID)
				if alc == nil {
					return nil, fmt.Errorf("access log config uid '%s' not found", alcUID)
				}
				vs.Spec.AccessLogConfigs = append(vs.Spec.AccessLogConfigs, &v1alpha1.ResourceRef{
					Name:      alc.Name,
					Namespace: &alc.Namespace,
				})
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

	// Handle extra fields from the request
	if len(req.Msg.ExtraFields) > 0 {
		vs.Spec.ExtraFields = req.Msg.ExtraFields
	}

	if err := vs.FillFromTemplate(template, vs.Spec.TemplateOptions...); err != nil {
		return nil, err
	}

	res := &v1.FillTemplateResponse{}

	if req.Msg.ExpandReferences {
		data, err := s.expandReferences(vs)
		if err != nil {
			return nil, err
		}
		res.Raw = string(data)
	} else {
		data, err := json.Marshal(vs.Spec)
		if err != nil {
			return nil, err
		}
		res.Raw = string(data)
	}

	return connect.NewResponse(res), nil
}

func (s *VirtualServiceTemplateStore) expandReferences(vs *v1alpha1.VirtualService) ([]byte, error) {
	data, err := json.Marshal(vs.Spec.VirtualServiceCommonSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal to map: %w", err)
	}

	if vs.Spec.VirtualServiceCommonSpec.Listener != nil {
		if vs.Spec.VirtualServiceCommonSpec.Listener.Namespace == nil {
			return nil, fmt.Errorf("listener namespace is required")
		}
		listener := s.store.GetListener(helpers.NamespacedName{
			Namespace: *vs.Spec.VirtualServiceCommonSpec.Listener.Namespace,
			Name:      vs.Spec.VirtualServiceCommonSpec.Listener.Name,
		})
		if listener == nil {
			return nil, fmt.Errorf("listener '%s' not found", vs.Spec.VirtualServiceCommonSpec.Listener.Name)
		}
		var listenerMap map[string]interface{}
		if err := yaml.Unmarshal(listener.Spec.Raw, &listenerMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal listener spec: %w", err)
		}
		result["listener"] = listenerMap
	}

	if vs.Spec.AccessLogConfig != nil {
		if vs.Spec.VirtualServiceCommonSpec.AccessLogConfig.Namespace == nil {
			return nil, fmt.Errorf("access log config namespace is required")
		}
		accessLogConfig := s.store.GetAccessLog(helpers.NamespacedName{
			Namespace: *vs.Spec.VirtualServiceCommonSpec.AccessLogConfig.Namespace,
			Name:      vs.Spec.VirtualServiceCommonSpec.AccessLogConfig.Name,
		})
		if accessLogConfig == nil {
			return nil, fmt.Errorf("accessLogConfig '%s' not found", vs.Spec.VirtualServiceCommonSpec.AccessLogConfig.Name)
		}
		var accessLogConfigMap map[string]interface{}
		if err := yaml.Unmarshal(accessLogConfig.Spec.Raw, &accessLogConfigMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal accessLogConfig spec: %w", err)
		}
		result["accessLogConfig"] = accessLogConfigMap
	}

	if len(vs.Spec.AccessLogConfigs) > 0 {
		result["accessLogConfigs"] = make([]map[string]interface{}, 0, len(vs.Spec.AccessLogConfigs))
		for _, alc := range vs.Spec.AccessLogConfigs {
			if alc.Namespace == nil {
				return nil, fmt.Errorf("access log config namespace is required")
			}
			accessLogConfig := s.store.GetAccessLog(helpers.NamespacedName{
				Namespace: *alc.Namespace,
				Name:      alc.Name,
			})
			if accessLogConfig == nil {
				return nil, fmt.Errorf("accessLogConfig '%s' not found", alc.Name)
			}
			var accessLogConfigMap map[string]interface{}
			if err := yaml.Unmarshal(accessLogConfig.Spec.Raw, &accessLogConfigMap); err != nil {
				return nil, fmt.Errorf("failed to unmarshal accessLogConfig spec: %w", err)
			}
			result["accessLogConfigs"] = append(result["accessLogConfigs"].([]map[string]interface{}), accessLogConfigMap)
		}
	}

	if len(vs.Spec.AdditionalRoutes) > 0 {
		routes := make([]map[string]interface{}, 0)
		for _, route := range vs.Spec.AdditionalRoutes {
			if route.Namespace == nil {
				return nil, fmt.Errorf("route namespace is required")
			}
			r := s.store.GetRoute(helpers.NamespacedName{
				Namespace: *route.Namespace,
				Name:      route.Name,
			})
			if r == nil {
				return nil, fmt.Errorf("route '%s' not found", route.Name)
			}
			for _, rr := range r.Spec {
				var tmp map[string]interface{}
				if err := yaml.Unmarshal(rr.Raw, &tmp); err != nil {
					return nil, fmt.Errorf("failed to unmarshal route spec: %w", err)
				}
				routes = append(routes, tmp)
			}
		}
		result["additionalRoutes"] = routes
	}

	if len(vs.Spec.AdditionalHttpFilters) > 0 {
		httpFilters := make([]map[string]interface{}, 0)
		for _, httpFilter := range vs.Spec.AdditionalHttpFilters {
			if httpFilter.Namespace == nil {
				return nil, fmt.Errorf("httpFilter namespace is required")
			}
			hf := s.store.GetHTTPFilter(helpers.NamespacedName{
				Namespace: *httpFilter.Namespace,
				Name:      httpFilter.Name,
			})
			if hf == nil {
				return nil, fmt.Errorf("httpFilter '%s' not found", httpFilter.Name)
			}
			for _, h := range hf.Spec {
				var tmp map[string]interface{}
				if err := yaml.Unmarshal(h.Raw, &tmp); err != nil {
					return nil, fmt.Errorf("failed to unmarshal httpFilter spec: %w", err)
				}
				httpFilters = append(httpFilters, tmp)
			}
		}
		result["additionalHttpFilters"] = httpFilters
	}

	resultData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal virtual service map: %v", err)
	}

	return resultData, nil
}
