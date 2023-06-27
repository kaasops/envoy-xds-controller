package xds

import (
	"context"
	"errors"
	"fmt"
	"strings"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	nodeIDAnnotation = "envoy.kaasops.io/node-id"
	defaultNodeID    = "default"
)

var (
	ErrNotSupported = errors.New("not supported type for create or update kubernetes resource")
)

func Ensure(ctx context.Context, cache cachev3.SnapshotCache, obj client.Object) error {
	resource, resourceType, err := unmarshal(ctx, obj)
	if err != nil {
		return err
	}

	nodeID := getNodeID(obj)

	cachedResources, err := getResourceFromCache(cache, resourceType, nodeID)
	if err != nil {
		return err
	}

	cachedResources[obj.GetName()] = resource

	for _, cr := range cachedResources {

		snapNew, _ := cachev3.NewSnapshot("1", map[resourcev3.Type][]types.Resource{
			resourceType: {cr},
		})

		cache.SetSnapshot(context.Background(), nodeID, snapNew)
	}

	return nil
}

func unmarshal(ctx context.Context, obj client.Object) (types.Resource, string, error) {
	unmarshaler := &protojson.UnmarshalOptions{
		AllowPartial:   false,
		DiscardUnknown: true,
	}

	switch o := obj.(type) {
	case *v1alpha1.Cluster:
		resource := &clusterv3.Cluster{}
		if err := unmarshaler.Unmarshal(o.Spec.Raw, resource); err != nil {
			return nil, "", err
		}
		return resource, resourcev3.ClusterType, nil
	case *v1alpha1.Endpoint:
		resource := &endpointv3.Endpoint{}
		if err := unmarshaler.Unmarshal(o.Spec.Raw, resource); err != nil {
			return nil, "", err
		}
		return resource, resourcev3.ListenerType, nil
	case *v1alpha1.Route:
		resource := &routev3.Route{}
		if err := unmarshaler.Unmarshal(o.Spec.Raw, resource); err != nil {
			return nil, "", err
		}
		return resource, resourcev3.RouteType, nil
	case *v1alpha1.Listener:
		resource := &listenerv3.Listener{}
		if err := unmarshaler.Unmarshal(o.Spec.Raw, resource); err != nil {
			return nil, "", err
		}
		return resource, resourcev3.SecretType, nil
	default:
		return nil, "", fmt.Errorf("%w.\n %+v", ErrNotSupported, obj)
	}
}

func getNodeID(obj client.Object) string {
	annotations := obj.GetAnnotations()

	nodeID, ok := annotations[nodeIDAnnotation]
	if !ok {
		return defaultNodeID
	}

	return nodeID
}

func getResourceFromCache(cache cachev3.SnapshotCache, resourceType string, nodeID string) (map[string]types.Resource, error) {
	snap, err := cache.GetSnapshot(nodeID)
	if err == nil {
		return snap.GetResources(resourceType), nil
	}
	if strings.Contains(err.Error(), "no snapshot found for node") {
		return map[string]types.Resource{}, nil
	}
	return nil, err
}
