package helpers

import (
	"fmt"
	"strings"
)

type NamespacedName struct {
	Namespace string
	Name      string
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

func SplitNamespacedName(resourceName string) (namespace, name string, err error) {
	splitResourceName := strings.Split(resourceName, "/")

	if len(splitResourceName) < 2 {
		return "", "", fmt.Errorf("invalid resource name %s", resourceName)
	}

	namespace = splitResourceName[0]
	name = splitResourceName[1]

	return
}
