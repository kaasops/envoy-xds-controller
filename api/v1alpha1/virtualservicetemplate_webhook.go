package v1alpha1

import (
	"context"
	"fmt"
	"github.com/kaasops/envoy-xds-controller/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (vst *VirtualServiceTemplate) ValidateDelete(ctx context.Context, cl client.Client) error {
	virtualServices := &VirtualServiceList{}

	if err := cl.List(ctx, virtualServices); err != nil {
		return fmt.Errorf("%v. %w", errors.GetFromKubernetesMessage, err)
	}

	for _, vs := range virtualServices.Items {
		if vs.Spec.Template == nil {
			continue
		}

		if vs.Spec.Template.Name == vst.Name &&
			((vs.Spec.Template.Namespace != nil && *vs.Spec.Template.Namespace == vst.Namespace) || vst.Namespace == vs.Namespace) {
			return fmt.Errorf("%v. It used in Virtual Service %v/%v", errors.DeleteInKubernetesMessage, vs.Namespace, vs.Name)
		}
	}
	return nil
}
