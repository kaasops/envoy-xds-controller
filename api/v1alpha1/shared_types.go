package v1alpha1

// type Message struct {
// 	S string
// }

type Message string

type ResourceRef struct {
	Name      string  `json:"name,omitempty"`
	Namespace *string `json:"namespace,omitempty"`
}
