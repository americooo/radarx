// Package scope enforces that RadarX only ever touches hosts the operator is
// authorized to test. This is the safety gate required before any scan runs:
// a target outside scope must be refused, not silently skipped.
package scope

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Scope is a set of authorized root domains. A host is in scope if it equals
// a root exactly or is a subdomain of one (e.g. root "example.com" covers
// "api.example.com" but not "notexample.com" or "example.com.evil.com").
type Scope struct {
	roots []string
}

// Load reads a scope file: one root domain per line, blank lines and lines
// starting with '#' are ignored. Domains are lowercased and trimmed.
func Load(path string) (*Scope, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("scope: open %s: %w", path, err)
	}
	defer f.Close()

	var roots []string
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		roots = append(roots, strings.ToLower(line))
	}
	if err := scan.Err(); err != nil {
		return nil, fmt.Errorf("scope: read %s: %w", path, err)
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("scope: %s has no scope entries", path)
	}
	return &Scope{roots: roots}, nil
}

// New builds a Scope directly from a list of root domains (e.g. for tests or
// programmatic construction), applying the same normalization as Load.
func New(roots []string) *Scope {
	s := &Scope{}
	for _, r := range roots {
		r = strings.ToLower(strings.TrimSpace(r))
		if r != "" {
			s.roots = append(s.roots, r)
		}
	}
	return s
}

// Allows reports whether host is within scope: an exact match on a root, or
// a subdomain of one.
func (s *Scope) Allows(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	for _, root := range s.roots {
		if host == root || strings.HasSuffix(host, "."+root) {
			return true
		}
	}
	return false
}

// Roots returns the authorized root domains (read-only copy).
func (s *Scope) Roots() []string {
	out := make([]string, len(s.roots))
	copy(out, s.roots)
	return out
}
