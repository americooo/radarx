package scope

import "testing"

func TestAllows(t *testing.T) {
	s := New([]string{"example.com", "Other-Example.org"})

	cases := []struct {
		host string
		want bool
	}{
		{"example.com", true},
		{"api.example.com", true},
		{"deep.api.example.com", true},
		{"notexample.com", false},
		{"example.com.evil.com", false},
		{"other-example.org", true}, // case-insensitive
		{"api.OTHER-EXAMPLE.ORG", true},
		{"evil.com", false},
		{"example.com.", true}, // trailing dot normalized
	}
	for _, c := range cases {
		if got := s.Allows(c.host); got != c.want {
			t.Errorf("Allows(%q) = %v, want %v", c.host, got, c.want)
		}
	}
}
