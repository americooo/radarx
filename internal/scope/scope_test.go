package scope

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scope.txt")
	content := "# comment\nexample.com\n\n  api.other.com  \n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !s.Allows("sub.example.com") {
		t.Error("expected sub.example.com to be in scope")
	}
	if !s.Allows("api.other.com") {
		t.Error("expected api.other.com (trimmed) to be in scope")
	}
	if s.Allows("random.com") {
		t.Error("expected random.com to be out of scope")
	}
}

func TestLoadEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scope.txt")
	if err := os.WriteFile(path, []byte("# only comments\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("expected error for scope file with no entries")
	}
}

func TestLoadMissing(t *testing.T) {
	if _, err := Load("/nonexistent/scope.txt"); err == nil {
		t.Error("expected error for missing scope file")
	}
}

func TestCheckPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scope.txt")
	content := "example.com\nOther-Example.org\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CheckPath(path, "api.example.com"); err != nil {
		t.Errorf("expected api.example.com to be in scope, got error: %v", err)
	}
	if err := CheckPath(path, "sub.other-example.org"); err != nil {
		t.Errorf("expected sub.other-example.org to be in scope, got error: %v", err)
	}
	if err := CheckPath(path, "evil.com"); err == nil {
		t.Error("expected evil.com to be rejected as out of scope")
	}
}

func TestCheckPathMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.txt")
	if err := CheckPath(path, "example.com"); err == nil {
		t.Error("expected error when scope file doesn't exist")
	}
}
