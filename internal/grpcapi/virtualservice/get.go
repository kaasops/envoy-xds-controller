package virtualservice

import (
	"context"
	"encoding/json"
	"fmt"

	api "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"connectrpc.com/connect"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/kaasops/envoy-xds-controller/internal/grpcapi"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
	commonv1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/common/v1"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service/v1"
	virtual_service_templatev1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service_template/v1"
)

func (s *VirtualServiceStore) GetVirtualService(ctx context.Context, req *connect.Request[v1.GetVirtualServiceRequest]) (*connect.Response[v1.GetVirtualServiceResponse], error) {
	if req.Msg.Uid == "" {
		return nil, fmt.Errorf("uid is required")
	}
	var matches api.VirtualServiceList
	if err := s.client.List(ctx, &matches, client.MatchingFields{uidIndex: req.Msg.Uid}); err != nil {
		return nil, err
	}
	if len(matches.Items) != 1 {
		return nil, fmt.Errorf("virtual service uid '%s' not found", req.Msg.Uid)
	}
	vs := &matches.Items[0]
	allowed, err := grpcapi.GetAuthorizerFromContext(ctx).Authorize(vs.GetAccessGroup(), vs.Name)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not authorized to get virtual service"))
	}
	resp := &v1.GetVirtualServiceResponse{
		Uid:         string(vs.UID),
		Name:        vs.GetLabelName(),
		NodeIds:     vs.GetNodeIDs(),
		AccessGroup: vs.GetAccessGroup(),
		IsEditable:  vs.IsEditable(),
		Description: vs.GetDescription(),
		Raw:         string(vs.Raw()),
		Status: &v1.Status{
			Invalid: vs.Status.Invalid,
			Message: vs.Status.Message,
		},
		ExtraFields: vs.Spec.ExtraFields,
	}
	if vs.Spec.Template != nil {
		resp.Template = resolveRef(vs.Spec.Template, vs.Namespace, s.store.GetVirtualServiceTemplate)
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
		resp.Listener = resolveRef(vs.Spec.Listener, vs.Namespace, s.store.GetListener)
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
		ref := resolveRef(vs.Spec.AccessLogConfig, vs.Namespace, s.store.GetAccessLog)
		resp.AccessLog = &v1.GetVirtualServiceResponse_AccessLogConfigs{
			AccessLogConfigs: &commonv1.ResourceRefs{Refs: []*commonv1.ResourceRef{ref}},
		}
	}
	if len(vs.Spec.AccessLogConfigs) > 0 {
		accessLogConfigs := make([]*commonv1.ResourceRef, 0, len(vs.Spec.AccessLogConfigs))
		for _, aclRef := range vs.Spec.AccessLogConfigs {
			ref := resolveRef(aclRef, vs.Namespace, s.store.GetAccessLog)
			accessLogConfigs = append(accessLogConfigs, ref)
		}
		resp.AccessLog = &v1.GetVirtualServiceResponse_AccessLogConfigs{
			AccessLogConfigs: &commonv1.ResourceRefs{
				Refs: accessLogConfigs,
			},
		}
	}
	if vs.Spec.AccessLog != nil {
		resp.AccessLog = &v1.GetVirtualServiceResponse_AccessLogConfigRaw{
			AccessLogConfigRaw: rawJSONString(vs.Spec.AccessLog),
		}
	}
	if len(vs.Spec.AccessLogs) > 0 {
		resp.AccessLog = &v1.GetVirtualServiceResponse_AccessLogConfigRaw{
			AccessLogConfigRaw: rawJSONString(vs.Spec.AccessLogs),
		}
	}
	if vs.Spec.AdditionalRoutes != nil {
		resp.AdditionalRoutes = make([]*commonv1.ResourceRef, 0, len(vs.Spec.AdditionalRoutes))
		for _, route := range vs.Spec.AdditionalRoutes {
			ref := resolveRef(route, vs.Namespace, s.store.GetRoute)
			resp.AdditionalRoutes = append(resp.AdditionalRoutes, ref)
		}
	}
	if vs.Spec.AdditionalHttpFilters != nil {
		resp.AdditionalHttpFilters = make([]*commonv1.ResourceRef, 0, len(vs.Spec.AdditionalHttpFilters))
		for _, filter := range vs.Spec.AdditionalHttpFilters {
			ref := resolveRef(filter, vs.Namespace, s.store.GetHTTPFilter)
			resp.AdditionalHttpFilters = append(resp.AdditionalHttpFilters, ref)
		}
	}
	if vs.Spec.UseRemoteAddress != nil {
		resp.UseRemoteAddress = vs.Spec.UseRemoteAddress
	}
	if vs.Spec.TlsConfig != nil {
		tlsConfig := &commonv1.TLSConfig{}
		if vs.Spec.TlsConfig.SecretRef != nil {
			tlsConfig.SecretRef = &commonv1.ResourceRef{
				Name: vs.Spec.TlsConfig.SecretRef.Name,
			}
		}
		tlsConfig.AutoDiscovery = vs.Spec.TlsConfig.AutoDiscovery
		resp.TlsConfig = tlsConfig
	}
	return connect.NewResponse(resp), nil
}

// A missing dependency can be the reason publication failed. Keep its name
// in the response even when there is no corresponding UID.
func resolveRef[T interface {
	client.Object
	comparable
}](ref *api.ResourceRef, namespace string, lookup func(helpers.NamespacedName) T) *commonv1.ResourceRef {
	result := &commonv1.ResourceRef{Name: ref.Name}
	obj := lookup(helpers.NamespacedName{Namespace: helpers.GetNamespace(ref.Namespace, namespace), Name: ref.Name})
	var missing T
	if obj != missing {
		result.Uid = string(obj.GetUID())
	}
	return result
}

func rawJSONString(input any) string {
	data, err := json.Marshal(input)
	if err != nil {
		return ""
	}
	return string(data)
}
