package k8s

import (
	"strings"

	"github.com/kaasops/envoy-xds-controller/pkg/options"
	"golang.org/x/exp/slices"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NodeIDs(obj client.Object) []string {
	annotation := NodeIDsAnnotation(obj)
	if annotation == "" {
		return nil
	}
	return strings.Split(NodeIDsAnnotation(obj), ",")
}

func NodeIDsAnnotation(obj client.Object) string {
	annotations := obj.GetAnnotations()

	annotation, ok := annotations[options.NodeIDAnnotation]
	if !ok {
		return ""
	}

	return annotation
}

func NodeIDsContains(s1, s2 []string) bool {

	if len(s1) > len(s2) {
		return false
	}

	for _, e := range s1 {
		if !slices.Contains(s2, e) {
			return false
		}
	}

	return true
}
