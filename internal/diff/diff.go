// Package diff compares two snapshots of a target and reports what changed.
// This is the heart of RadarX: being first to see a NEW asset is the whole
// competitive edge in bug bounty and continuous recon.
package diff

import (
	"sort"

	"github.com/americooo/radarx/internal/model"
)

// Compare diffs an old snapshot against a new one and returns the changes.
// A nil or empty `old` means every asset in `new` is reported as new (first run).
func Compare(old, new model.Snapshot) model.DiffResult {
	res := model.DiffResult{
		TargetID: new.TargetID,
		Root:     new.Root,
		At:       new.TakenAt,
	}

	oldByKey := indexByKey(old.Assets)
	newByKey := indexByKey(new.Assets)

	// New + modified: walk everything in the new snapshot.
	for key, na := range newByKey {
		oa, existed := oldByKey[key]
		if !existed {
			asset := na
			res.Changes = append(res.Changes, model.Change{
				Type:  model.ChangeNew,
				Kind:  na.Kind,
				Key:   na.Key,
				After: &asset,
			})
			continue
		}
		if fields := changedFields(oa, na); len(fields) > 0 {
			before, after := oa, na
			res.Changes = append(res.Changes, model.Change{
				Type:   model.ChangeModified,
				Kind:   na.Kind,
				Key:    na.Key,
				Before: &before,
				After:  &after,
				Fields: fields,
			})
		}
	}

	// Removed: anything in old that's gone from new.
	for key, oa := range oldByKey {
		if _, still := newByKey[key]; !still {
			before := oa
			res.Changes = append(res.Changes, model.Change{
				Type:   model.ChangeRemoved,
				Kind:   oa.Kind,
				Key:    oa.Key,
				Before: &before,
			})
		}
	}

	// Deterministic ordering: new first (most important), then modified, then
	// removed; alphabetical within each group. Keeps notifications stable.
	sort.SliceStable(res.Changes, func(i, j int) bool {
		pi, pj := priority(res.Changes[i].Type), priority(res.Changes[j].Type)
		if pi != pj {
			return pi < pj
		}
		return res.Changes[i].Key < res.Changes[j].Key
	})

	return res
}

func indexByKey(assets []model.Asset) map[string]model.Asset {
	m := make(map[string]model.Asset, len(assets))
	for _, a := range assets {
		// Namespace the key by kind so a subdomain and a port can't collide.
		m[string(a.Kind)+"|"+a.Key] = a
	}
	return m
}

// changedFields returns the names of enrichment fields that differ between two
// assets with the same identity. We only compare fields that meaningfully
// signal a change worth an operator's attention.
func changedFields(a, b model.Asset) []string {
	var f []string
	if a.StatusCode != b.StatusCode {
		f = append(f, "status_code")
	}
	if a.Title != b.Title {
		f = append(f, "title")
	}
	if a.Server != b.Server {
		f = append(f, "server")
	}
	if a.Scheme != b.Scheme {
		f = append(f, "scheme")
	}
	if a.CertCN != b.CertCN {
		f = append(f, "cert_cn")
	}
	if a.CertExpiry != b.CertExpiry {
		f = append(f, "cert_expiry")
	}
	if !sameStrings(a.IP, b.IP) {
		f = append(f, "ip")
	}
	return f
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ca := append([]string(nil), a...)
	cb := append([]string(nil), b...)
	sort.Strings(ca)
	sort.Strings(cb)
	for i := range ca {
		if ca[i] != cb[i] {
			return false
		}
	}
	return true
}

func priority(t model.ChangeType) int {
	switch t {
	case model.ChangeNew:
		return 0
	case model.ChangeModified:
		return 1
	case model.ChangeRemoved:
		return 2
	default:
		return 3
	}
}
