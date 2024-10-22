package v1alpha1

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
