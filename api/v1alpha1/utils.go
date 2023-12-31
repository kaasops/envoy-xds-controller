package v1alpha1

import (
	"k8s.io/apimachinery/pkg/types"
)

func (rf ResourceRef) NamespacedName(namespace string) types.NamespacedName {
	return types.NamespacedName{
		Name:      rf.Name,
		Namespace: namespace,
	}
}
