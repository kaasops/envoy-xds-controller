package virtualservice

import (
	"context"
	"fmt"
	"maps"
	"sort"

	"connectrpc.com/connect"
	api "github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/internal/grpcapi"
	v1 "github.com/kaasops/envoy-xds-controller/pkg/api/grpc/virtual_service/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *VirtualServiceStore) ListVirtualServices(ctx context.Context, r *connect.Request[v1.ListVirtualServicesRequest]) (*connect.Response[v1.ListVirtualServicesResponse], error) {
	if r.Msg.AccessGroup == "" {
		return nil, fmt.Errorf("access group is required")
	}

	authorizer := grpcapi.GetAuthorizerFromContext(ctx)

	// The manager cache receives status-only changes from Kubernetes even when
	// the controller skips rebuilding snapshots. Its list is read-only.
	var services api.VirtualServiceList
	if err := s.client.List(ctx, &services, client.UnsafeDisableDeepCopy); err != nil {
		return nil, err
	}
	list := make([]*v1.VirtualServiceListItem, 0, len(services.Items))
	for i := range services.Items {
		v := &services.Items[i]
		vsAccessGroup := v.GetAccessGroup()
		if r.Msg.AccessGroup != vsAccessGroup {
			continue
		}

		isAllowed, err := authorizer.Authorize(vsAccessGroup, v.Name)
		if err != nil {
			return nil, err
		}
		if !isAllowed {
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
			ExtraFields: maps.Clone(v.Spec.ExtraFields),
		}
		if v.Spec.Template != nil {
			vs.Template = resolveRef(v.Spec.Template, v.Namespace, s.store.GetVirtualServiceTemplate)
		}
		vs.IsEditable = v.IsEditable()
		list = append(list, vs)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return connect.NewResponse(&v1.ListVirtualServicesResponse{Items: list}), nil
}
