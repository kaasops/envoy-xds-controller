package middlewares

import "testing"

func TestUserNamePrefersEmail(t *testing.T) {
	for _, tc := range []struct{ email, preferred, subject, want string }{
		{"alice@example.com", "alice", "id-1", "alice@example.com"},
		{"", "alice", "id-1", "alice"},
		{"", "", "id-1", "id-1"},
		{"", "", "", ""},
	} {
		if got := userName(tc.email, tc.preferred, tc.subject); got != tc.want {
			t.Errorf("userName(%q, %q, %q) = %q, want %q", tc.email, tc.preferred, tc.subject, got, tc.want)
		}
	}
}
