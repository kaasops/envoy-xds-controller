package xds

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cachev3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resourcev3 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"github.com/kaasops/envoy-xds-controller/pkg/xds/cache"
	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// https://github.com/envoyproxy/go-control-plane/issues/390
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/tls_inspector/v3"
)

const (
	nodeIDAnnotation = "envoy.kaasops.io/node-id"
	defaultNodeID    = "main"
)

var (
	resourceTypes = []string{
		resourcev3.EndpointType,
		resourcev3.ClusterType,
		resourcev3.RouteType,
		resourcev3.ScopedRouteType,
		resourcev3.VirtualHostType,
		resourcev3.ListenerType,
		resourcev3.SecretType,
		resourcev3.ExtensionConfigType,
		resourcev3.RouteType,
	}

	ErrNotSupported = errors.New("not supported type for create or update kubernetes resource")
)

func Ensure(ctx context.Context, cache cache.Cache, obj client.Object) error {
	resource, resourceType, err := unmarshal(ctx, obj)
	if err != nil {
		return err
	}

	nodeID := getNodeID(obj)

	snap, err := newSnapshotWithResource(cache, nodeID, resource, obj.GetName(), resourceType)
	if err != nil {
		return err
	}

	return cache.SnapshotCache.SetSnapshot(context.TODO(), nodeID, snap)
}

func unmarshal(ctx context.Context, obj client.Object) (types.Resource, string, error) {
	unmarshaler := &protojson.UnmarshalOptions{
		AllowPartial: false,
		// DiscardUnknown: true,
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
		return resource, resourcev3.EndpointType, nil
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
		return resource, resourcev3.ListenerType, nil
	case *v1alpha1.Secret:
		resource := &tlsv3.Secret{}
		time.Sleep(5 * time.Second)
		if err := unmarshaler.Unmarshal(o.Spec.Raw, resource); err != nil {
			return nil, "", err
		}
		return resource, resourcev3.SecretType, nil
	default:
		return nil, "", fmt.Errorf("%w.\n %+v", ErrNotSupported, obj)
	}
}

func newSnapshotWithResource(
	cache cache.Cache,
	nodeID string,
	resource types.Resource,
	resourceName string,
	resourceType string,
) (*cachev3.Snapshot, error) {
	version := 0

	// Create map for new snapshot
	resources := make(map[string][]types.Resource, 0)
	for _, t := range resourceTypes {
		// resourceCache := snap.GetResources(t)
		resourceCache, rVersionStr, err := cache.GetResourceFromCache(t, nodeID)
		if err != nil {
			return nil, err
		}

		// Get max version from resources
		if rVersionStr != "" {
			rVersion, err := strconv.Atoi(rVersionStr)
			if err != nil {
				return nil, err
			}
			if rVersion > version {
				version = rVersion
			}
		}

		// if our resource - update
		if t == resourceType {
			resourceCache[resourceName] = resource
		}

		res := make([]types.Resource, 0)

		for _, r := range resourceCache {
			res = append(res, r)
		}

		fmt.Printf("TYPE: %+v\nRES: %+v\n", t, res)
		resources[t] = res
	}

	// Increment version
	version++

	return cachev3.NewSnapshot(strconv.Itoa(version), resources)
}

func getNodeID(obj client.Object) string {
	annotations := obj.GetAnnotations()

	nodeID, ok := annotations[nodeIDAnnotation]
	if !ok {
		return defaultNodeID
	}

	return nodeID
}

// func getResourceFromCache(cache cachev3.SnapshotCache, resourceType string, nodeID string) (map[string]types.Resource, string, error) {
// 	snap, err := cache.GetSnapshot(nodeID)
// 	if err == nil {
// 		if snap.GetResources(resourceType) == nil {
// 			return make(map[string]types.Resource), snap.GetVersion(resourceType), nil
// 		}
// 		return snap.GetResources(resourceType), snap.GetVersion(resourceType), nil
// 	}
// 	if strings.Contains(err.Error(), "no snapshot found for node") {
// 		return map[string]types.Resource{}, "", nil
// 	}
// 	return nil, "", err
// }
