package xds

import (
	"context"
	"errors"
	"fmt"

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

func Ensure(ctx context.Context, cache Cache, obj client.Object) error {
	resource, resourceType, err := unmarshal(ctx, obj)
	if err != nil {
		return err
	}

	nodeID := getNodeID(obj)

	// Get exists Resources fron cache
	cachedResources, err := cache.GetResource(resourceType, nodeID)
	if err != nil {
		return err
	}

	cachedResources[obj.GetName()] = resource

	for _, cr := range cachedResources {

		snapNew, _ := cachev3.NewSnapshot("1", map[resourcev3.Type][]types.Resource{
			resourceType: {cr},
		})

		cache.SetSnaphot(context.Background(), nodeID, snapNew)
	}

	return nil
}

func unmarshal(ctx context.Context, obj client.Object) (types.Resource, string, error) {
	unmarshaler := &protojson.UnmarshalOptions{
		AllowPartial:   false,
		DiscardUnknown: true,
	}

	switch obj.(type) {
	case *v1alpha1.Cluster:
		resource := &clusterv3.Cluster{}
		dObj := obj.(*v1alpha1.Cluster)
		if err := unmarshaler.Unmarshal(dObj.Spec.Raw, resource); err != nil {
			return nil, "", err
		}
		return resource, resourcev3.ClusterType, nil
	case *v1alpha1.Endpoint:
		resource := &endpointv3.Endpoint{}
		dObj := obj.(*v1alpha1.Endpoint)
		if err := unmarshaler.Unmarshal(dObj.Spec.Raw, resource); err != nil {
			return nil, "", err
		}
		return resource, resourcev3.ListenerType, nil
	case *v1alpha1.Route:
		resource := &routev3.Route{}
		dObj := obj.(*v1alpha1.Route)
		if err := unmarshaler.Unmarshal(dObj.Spec.Raw, resource); err != nil {
			return nil, "", err
		}
		return resource, resourcev3.RouteType, nil
	case *v1alpha1.Listener:
		resource := &listenerv3.Listener{}
		dObj := obj.(*v1alpha1.Listener)
		if err := unmarshaler.Unmarshal(dObj.Spec.Raw, resource); err != nil {
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
