package helpers

import (
	"fmt"
	"strings"
)

type NamespacedName struct {
	Namespace string
	Name      string
}

// NamespacedNameFromString parses the "namespace/name" form produced by String().
func NamespacedNameFromString(s string) (NamespacedName, error) {
	ns, name, found := strings.Cut(s, "/")
	if !found || ns == "" || name == "" {
		return NamespacedName{}, fmt.Errorf("invalid namespaced name %q", s)
	}
	return NamespacedName{Namespace: ns, Name: name}, nil
}

func (n *NamespacedName) String() string {
	if n.Namespace == "" {
		return "default/" + n.Name
	}
	return n.Namespace + "/" + n.Name
}

func GetNamespace(ns *string, defaultNs string) string {
	if ns != nil {
		return *ns
	}
	return defaultNs
}

func BoolFromPtr(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
