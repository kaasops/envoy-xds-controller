package v1alpha1

import (
	"bytes"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	"github.com/kaasops/envoy-xds-controller/internal/protoutil"
)

func (c *Cluster) UnmarshalV3() (*cluster.Cluster, error) {
	return c.unmarshalV3()
}

func (c *Cluster) UnmarshalV3AndValidate() (*cluster.Cluster, error) {
	clusterV3, err := c.unmarshalV3()
	if err != nil {
		return nil, err
	}
	if err := clusterV3.ValidateAll(); err != nil {
		return nil, err
	}
	return clusterV3, nil
}

func (c *Cluster) unmarshalV3() (*cluster.Cluster, error) {
	if c.Spec == nil {
		return nil, ErrSpecNil
	}
	var clusterV3 cluster.Cluster
	if err := protoutil.Unmarshaler.Unmarshal(c.Spec.Raw, &clusterV3); err != nil {
		return nil, err
	}
	return &clusterV3, nil
}

func (c *Cluster) IsEqual(other *Cluster) bool {
	if c == nil && other == nil {
		return true
	}
	if c == nil || other == nil {
		return false
	}
	if c.Spec == nil && other.Spec == nil {
		return true
	}
	if c.Spec == nil || other.Spec == nil {
		return false
	}
	if c.Spec.Raw == nil && other.Spec.Raw == nil {
		return true
	}
	if c.Spec.Raw == nil || other.Spec.Raw == nil {
		return false
	}
	if len(c.Spec.Raw) != len(other.Spec.Raw) {
		return false
	}
	if !bytes.Equal(c.Spec.Raw, other.Spec.Raw) {
		return false
	}
	return true
}

func (c *Cluster) GetDescription() string {
	return c.Annotations[annotationDescription]
}
