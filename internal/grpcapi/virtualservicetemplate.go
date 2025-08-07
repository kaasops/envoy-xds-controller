package grpcapi

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	commonv1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/common/v1"

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

// validateTemplateAccess checks if the template exists and if the user has access to it
func (s *VirtualServiceTemplateStore) validateTemplateAccess(ctx context.Context, templateUID string) (*v1alpha1.VirtualServiceTemplate, error) {
	if templateUID == "" {
		return nil, fmt.Errorf("template uid is required")
	}

	template := s.store.GetVirtualServiceTemplateByUID(templateUID)
	if template == nil {
		return nil, fmt.Errorf("template not found")
	}

	authorizer := GetAuthorizerFromContext(ctx)
	ok, err := authorizer.Authorize(template.GetAccessGroup(), template.Name)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, fmt.Errorf("access group '%s' is not allowed to fill template '%s'", template.GetAccessGroup(), template.Name)
	}

	return template, nil
}

// setupVirtualServiceName sets the name for the virtual service
func setupVirtualServiceName(vs *v1alpha1.VirtualService, requestName, templateName string) {
	if requestName != "" {
		vs.Name = requestName
	} else {
		vs.Name = templateName + "-vs"
	}
}

// setupListenerRef adds listener reference to the virtual service if listener UID is provided
func (s *VirtualServiceTemplateStore) setupListenerRef(vs *v1alpha1.VirtualService, listenerUID string) error {
	if listenerUID == "" {
		return nil
	}

	listener := s.store.GetListenerByUID(listenerUID)
	if listener == nil {
		return fmt.Errorf("listener uid '%s' not found", listenerUID)
	}

	vs.Spec.Listener = &v1alpha1.ResourceRef{
		Name:      listener.Name,
		Namespace: &listener.Namespace,
	}

	return nil
}

// setupVirtualHost adds virtual host configuration to the virtual service if provided
func setupVirtualHost(vs *v1alpha1.VirtualService, virtualHost *commonv1.VirtualHost) error {
	if virtualHost == nil {
		return nil
	}

	vHost := &routev3.VirtualHost{
		Name:    vs.Name + "-virtual-host",
		Domains: virtualHost.Domains,
	}

	vhData, err := protoutil.Marshaler.Marshal(vHost)
	if err != nil {
		return fmt.Errorf("failed to marshal virtual host: %w", err)
	}

	vs.Spec.VirtualHost = &runtime.RawExtension{Raw: vhData}
	return nil
}

// setupAccessLogConfigs adds access log configurations to the virtual service
func (s *VirtualServiceTemplateStore) setupAccessLogConfigs(vs *v1alpha1.VirtualService, accessLogConfigUIDs *commonv1.UIDS) error {
	if accessLogConfigUIDs == nil || len(accessLogConfigUIDs.GetUids()) == 0 {
		return nil
	}

	vs.Spec.AccessLogConfigs = make([]*v1alpha1.ResourceRef, 0, len(accessLogConfigUIDs.GetUids()))

	for _, alcUID := range accessLogConfigUIDs.GetUids() {
		if alcUID == "" {
			continue
		}

		alc := s.store.GetAccessLogByUID(alcUID)
		if alc == nil {
			return fmt.Errorf("access log config uid '%s' not found", alcUID)
		}

		vs.Spec.AccessLogConfigs = append(vs.Spec.AccessLogConfigs, &v1alpha1.ResourceRef{
			Name:      alc.Name,
			Namespace: &alc.Namespace,
		})
	}

	return nil
}

// setupAdditionalRoutes adds additional routes to the virtual service
func (s *VirtualServiceTemplateStore) setupAdditionalRoutes(vs *v1alpha1.VirtualService, routeUIDs []string) error {
	if len(routeUIDs) == 0 {
		return nil
	}

	for _, uid := range routeUIDs {
		route := s.store.GetRouteByUID(uid)
		if route == nil {
			return fmt.Errorf("route uid '%s' not found", uid)
		}

		vs.Spec.AdditionalRoutes = append(vs.Spec.AdditionalRoutes, &v1alpha1.ResourceRef{
			Name:      route.Name,
			Namespace: &route.Namespace,
		})
	}

	return nil
}

// setupAdditionalHttpFilters adds additional HTTP filters to the virtual service
func (s *VirtualServiceTemplateStore) setupAdditionalHttpFilters(vs *v1alpha1.VirtualService, filterUIDs []string) error {
	if len(filterUIDs) == 0 {
		return nil
	}

	for _, uid := range filterUIDs {
		filter := s.store.GetHTTPFilterByUID(uid)
		if filter == nil {
			return fmt.Errorf("http filter uid '%s' not found", uid)
		}

		vs.Spec.AdditionalHttpFilters = append(vs.Spec.AdditionalHttpFilters, &v1alpha1.ResourceRef{
			Name:      filter.Name,
			Namespace: &filter.Namespace,
		})
	}

	return nil
}

// setupTemplateOptions adds template options to the virtual service
func setupTemplateOptions(vs *v1alpha1.VirtualService, templateOptions []*v1.TemplateOption) {
	if len(templateOptions) == 0 {
		return
	}

	tOpts := make([]v1alpha1.TemplateOpts, 0, len(templateOptions))
	for _, opt := range templateOptions {
		tOpts = append(tOpts, v1alpha1.TemplateOpts{
			Field:    opt.Field,
			Modifier: ParseTemplateOptionModifier(opt.Modifier),
		})
	}

	vs.Spec.TemplateOptions = tOpts
}

// setupUseRemoteAddress sets the UseRemoteAddress field if provided
func setupUseRemoteAddress(vs *v1alpha1.VirtualService, useRemoteAddress *bool) {
	if useRemoteAddress != nil {
		vs.Spec.UseRemoteAddress = useRemoteAddress
	}
}

// setupExtraFields sets the ExtraFields if provided
func setupExtraFields(vs *v1alpha1.VirtualService, extraFields map[string]string) {
	if len(extraFields) > 0 {
		vs.Spec.ExtraFields = extraFields
	}
}

// setupTlsConfig adds TLS configuration to the virtual service
func setupTlsConfig(vs *v1alpha1.VirtualService, tlsConfig *commonv1.TLSConfig) {
	if tlsConfig == nil {
		return
	}

	tlsCfg := &v1alpha1.TlsConfig{
		AutoDiscovery: tlsConfig.AutoDiscovery,
	}

	if tlsConfig.SecretRef != nil {
		tlsCfg.SecretRef = &v1alpha1.ResourceRef{
			Name: tlsConfig.SecretRef.Name,
			// TODO: Namespace
		}
	}

	vs.Spec.TlsConfig = tlsCfg
}

// prepareResponse prepares the response for the FillTemplate method
func (s *VirtualServiceTemplateStore) prepareResponse(vs *v1alpha1.VirtualService, expandReferences bool) (*v1.FillTemplateResponse, error) {
	res := &v1.FillTemplateResponse{}

	var data []byte
	var err error

	if expandReferences {
		data, err = s.expandReferences(vs)
	} else {
		data, err = json.Marshal(vs.Spec)
	}

	if err != nil {
		return nil, err
	}

	res.Raw = string(data)
	return res, nil
}

func (s *VirtualServiceTemplateStore) FillTemplate(ctx context.Context, req *connect.Request[v1.FillTemplateRequest]) (*connect.Response[v1.FillTemplateResponse], error) {
	// Validate template and check access
	template, err := s.validateTemplateAccess(ctx, req.Msg.TemplateUid)
	if err != nil {
		return nil, err
	}

	// Initialize virtual service with template reference
	vs := &v1alpha1.VirtualService{}
	vs.Spec.Template = &v1alpha1.ResourceRef{
		Name:      template.Name,
		Namespace: &template.Namespace,
	}

	// Setup virtual service name
	setupVirtualServiceName(vs, req.Msg.Name, template.Name)

	// Setup listener reference if provided
	if err := s.setupListenerRef(vs, req.Msg.ListenerUid); err != nil {
		return nil, err
	}

	// Setup virtual host if provided
	if err := setupVirtualHost(vs, req.Msg.VirtualHost); err != nil {
		return nil, err
	}

	// Setup access log configurations if provided
	if err := s.setupAccessLogConfigs(vs, req.Msg.GetAccessLogConfigUids()); err != nil {
		return nil, err
	}

	// Setup additional routes if provided
	if err := s.setupAdditionalRoutes(vs, req.Msg.AdditionalRouteUids); err != nil {
		return nil, err
	}

	// Setup additional HTTP filters if provided
	if err := s.setupAdditionalHttpFilters(vs, req.Msg.AdditionalHttpFilterUids); err != nil {
		return nil, err
	}

	// Set UseRemoteAddress if provided
	setupUseRemoteAddress(vs, req.Msg.UseRemoteAddress)

	// Setup template options if provided
	setupTemplateOptions(vs, req.Msg.TemplateOptions)

	// Set extra fields if provided
	setupExtraFields(vs, req.Msg.ExtraFields)

	// Setup TLS configuration if provided
	setupTlsConfig(vs, req.Msg.TlsConfig)

	// Fill from template
	if err := vs.FillFromTemplate(template, vs.Spec.TemplateOptions...); err != nil {
		return nil, err
	}

	// Prepare response
	res, err := s.prepareResponse(vs, req.Msg.ExpandReferences)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(res), nil
}

// expandListenerReference adds listener reference to the result map
func (s *VirtualServiceTemplateStore) expandListenerReference(vs *v1alpha1.VirtualService, result map[string]interface{}) error {
	listener := vs.Spec.VirtualServiceCommonSpec.Listener
	if listener == nil {
		return nil
	}

	if listener.Namespace == nil {
		return fmt.Errorf("listener namespace is required")
	}

	l := s.store.GetListener(helpers.NamespacedName{
		Namespace: *listener.Namespace,
		Name:      listener.Name,
	})

	if l == nil {
		return fmt.Errorf("listener '%s' not found", listener.Name)
	}

	var listenerMap map[string]interface{}
	if err := yaml.Unmarshal(l.Spec.Raw, &listenerMap); err != nil {
		return fmt.Errorf("failed to unmarshal listener spec: %w", err)
	}

	result["listener"] = listenerMap
	return nil
}

// expandAccessLogConfigReference adds access log config reference to the result map
func (s *VirtualServiceTemplateStore) expandAccessLogConfigReference(vs *v1alpha1.VirtualService, result map[string]interface{}) error {
	accessLogConfig := vs.Spec.AccessLogConfig
	if accessLogConfig == nil {
		return nil
	}

	if accessLogConfig.Namespace == nil {
		return fmt.Errorf("access log config namespace is required")
	}

	alc := s.store.GetAccessLog(helpers.NamespacedName{
		Namespace: *accessLogConfig.Namespace,
		Name:      accessLogConfig.Name,
	})

	if alc == nil {
		return fmt.Errorf("accessLogConfig '%s' not found", accessLogConfig.Name)
	}

	var accessLogConfigMap map[string]interface{}
	if err := yaml.Unmarshal(alc.Spec.Raw, &accessLogConfigMap); err != nil {
		return fmt.Errorf("failed to unmarshal accessLogConfig spec: %w", err)
	}

	result["accessLogConfig"] = accessLogConfigMap
	return nil
}

// expandAccessLogConfigsReferences adds access log configs references to the result map
func (s *VirtualServiceTemplateStore) expandAccessLogConfigsReferences(vs *v1alpha1.VirtualService, result map[string]interface{}) error {
	if len(vs.Spec.AccessLogConfigs) == 0 {
		return nil
	}

	result["accessLogConfigs"] = make([]map[string]interface{}, 0, len(vs.Spec.AccessLogConfigs))

	for _, alc := range vs.Spec.AccessLogConfigs {
		if alc.Namespace == nil {
			return fmt.Errorf("access log config namespace is required")
		}

		accessLogConfig := s.store.GetAccessLog(helpers.NamespacedName{
			Namespace: *alc.Namespace,
			Name:      alc.Name,
		})

		if accessLogConfig == nil {
			return fmt.Errorf("accessLogConfig '%s' not found", alc.Name)
		}

		var accessLogConfigMap map[string]interface{}
		if err := yaml.Unmarshal(accessLogConfig.Spec.Raw, &accessLogConfigMap); err != nil {
			return fmt.Errorf("failed to unmarshal accessLogConfig spec: %w", err)
		}

		result["accessLogConfigs"] = append(result["accessLogConfigs"].([]map[string]interface{}), accessLogConfigMap)
	}

	return nil
}

// expandAdditionalRoutesReferences adds additional routes references to the result map
func (s *VirtualServiceTemplateStore) expandAdditionalRoutesReferences(vs *v1alpha1.VirtualService, result map[string]interface{}) error {
	if len(vs.Spec.AdditionalRoutes) == 0 {
		return nil
	}

	routes := make([]map[string]interface{}, 0)

	for _, route := range vs.Spec.AdditionalRoutes {
		if route.Namespace == nil {
			return fmt.Errorf("route namespace is required")
		}

		r := s.store.GetRoute(helpers.NamespacedName{
			Namespace: *route.Namespace,
			Name:      route.Name,
		})

		if r == nil {
			return fmt.Errorf("route '%s' not found", route.Name)
		}

		for _, rr := range r.Spec {
			var tmp map[string]interface{}
			if err := yaml.Unmarshal(rr.Raw, &tmp); err != nil {
				return fmt.Errorf("failed to unmarshal route spec: %w", err)
			}
			routes = append(routes, tmp)
		}
	}

	result["additionalRoutes"] = routes
	return nil
}

// expandAdditionalHttpFiltersReferences adds additional HTTP filters references to the result map
func (s *VirtualServiceTemplateStore) expandAdditionalHttpFiltersReferences(vs *v1alpha1.VirtualService, result map[string]interface{}) error {
	if len(vs.Spec.AdditionalHttpFilters) == 0 {
		return nil
	}

	httpFilters := make([]map[string]interface{}, 0)

	for _, httpFilter := range vs.Spec.AdditionalHttpFilters {
		if httpFilter.Namespace == nil {
			return fmt.Errorf("httpFilter namespace is required")
		}

		hf := s.store.GetHTTPFilter(helpers.NamespacedName{
			Namespace: *httpFilter.Namespace,
			Name:      httpFilter.Name,
		})

		if hf == nil {
			return fmt.Errorf("httpFilter '%s' not found", httpFilter.Name)
		}

		for _, h := range hf.Spec {
			var tmp map[string]interface{}
			if err := yaml.Unmarshal(h.Raw, &tmp); err != nil {
				return fmt.Errorf("failed to unmarshal httpFilter spec: %w", err)
			}
			httpFilters = append(httpFilters, tmp)
		}
	}

	result["additionalHttpFilters"] = httpFilters
	return nil
}

func (s *VirtualServiceTemplateStore) expandReferences(vs *v1alpha1.VirtualService) ([]byte, error) {
	// Marshal the VirtualServiceCommonSpec to JSON
	data, err := json.Marshal(vs.Spec.VirtualServiceCommonSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	// Unmarshal the JSON to a map
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal to map: %w", err)
	}

	// Expand listener reference
	if err := s.expandListenerReference(vs, result); err != nil {
		return nil, err
	}

	// Expand access log config reference
	if err := s.expandAccessLogConfigReference(vs, result); err != nil {
		return nil, err
	}

	// Expand access log configs references
	if err := s.expandAccessLogConfigsReferences(vs, result); err != nil {
		return nil, err
	}

	// Expand additional routes references
	if err := s.expandAdditionalRoutesReferences(vs, result); err != nil {
		return nil, err
	}

	// Expand additional HTTP filters references
	if err := s.expandAdditionalHttpFiltersReferences(vs, result); err != nil {
		return nil, err
	}

	// Marshal the result map to JSON
	resultData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal virtual service map: %v", err)
	}

	return resultData, nil
}
