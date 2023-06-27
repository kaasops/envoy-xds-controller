package resources

import (
	"context"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"github.com/kaasops/envoy-xds-controller/api/v1alpha1"
	"google.golang.org/protobuf/encoding/protojson"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterController interface {
	GetCluster(ctx context.Context) (*clusterv3.Cluster, error)
}

type clusterController struct {
	client    client.Client
	clusterCR v1alpha1.Cluster
}

func NewClusterController(c client.Client, cr v1alpha1.Cluster) ClusterController {
	return &clusterController{
		client:    c,
		clusterCR: cr,
	}
}

func (cc *clusterController) EnsureCluster() error {
	return nil
}

func (cc *clusterController) GetCluster(ctx context.Context) (*clusterv3.Cluster, error) {

	cluster := &clusterv3.Cluster{}

	unmarshaler := &protojson.UnmarshalOptions{
		AllowPartial:   false,
		DiscardUnknown: true,
	}

	if err := unmarshaler.Unmarshal(cc.clusterCR.Spec.Raw, cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}
