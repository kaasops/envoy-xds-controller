package v1alpha1

// type Message struct {
// 	S string
// }

type Message string

func (m *Message) Add(s string) {
	if m == nil {
		return
	}

	if *m == "" {
		*m = Message(s)
		return
	}

	*m = *m + "; " + Message(s)
}
