package virtualservice

import (
	"context"
	"fmt"
	"sort"

	"connectrpc.com/connect"
	"github.com/kaasops/envoy-xds-controller/internal/grpcapi"
	"github.com/kaasops/envoy-xds-controller/internal/helpers"
	commonv1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/common/v1"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service/v1"
)

func (s *VirtualServiceStore) ListVirtualServices(ctx context.Context, r *connect.Request[v1.ListVirtualServicesRequest]) (*connect.Response[v1.ListVirtualServicesResponse], error) {
	m := s.store.MapVirtualServices()
	list := make([]*v1.VirtualServiceListItem, 0, len(m))

	if r.Msg.AccessGroup == "" {
		return nil, fmt.Errorf("access group is required")
	}

	authorizer := grpcapi.GetAuthorizerFromContext(ctx)

	for _, v := range m {
		vsAccessGroup := v.GetAccessGroup()
		if vsAccessGroup == "" {
			vsAccessGroup = grpcapi.GeneralAccessGroup
		}

		isAllowed, err := authorizer.Authorize(vsAccessGroup, v.Name)
		if err != nil {
			return nil, err
		}
		if !isAllowed {
			continue
		}

		if r.Msg.AccessGroup != vsAccessGroup {
			continue
		}
		vs := &v1.VirtualServiceListItem{
			Uid:         string(v.UID),
			Name:        v.GetLabelName(),
			NodeIds:     v.GetNodeIDs(),
			AccessGroup: vsAccessGroup,
			Description: v.GetDescription(),
			Status: &v1.Status{
				Invalid: v.Status.Invalid,
				Message: v.Status.Message,
			},
			ExtraFields: v.Spec.ExtraFields,
		}
		if v.Spec.Template != nil {
			template := s.store.GetVirtualServiceTemplate(helpers.NamespacedName{
				Namespace: helpers.GetNamespace(v.Spec.Template.Namespace, v.Namespace),
				Name:      v.Spec.Template.Name,
			})
			if template == nil {
				// TODO: log
				continue
			}
			vs.Template = &commonv1.ResourceRef{
				Uid:  string(template.UID),
				Name: template.Name,
			}
		}
		vs.IsEditable = v.IsEditable()
		list = append(list, vs)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return connect.NewResponse(&v1.ListVirtualServicesResponse{Items: list}), nil
}
