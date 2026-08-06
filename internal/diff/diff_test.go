package diff

import (
	"testing"
	"time"

	"github.com/americooo/radarx/internal/model"
)

func sub(host string) model.Asset {
	return model.Asset{Kind: model.KindSubdomain, Key: host, Host: host}
}

func ep(host string, code int, title string) model.Asset {
	return model.Asset{Kind: model.KindEndpoint, Key: "https://" + host, Host: host, Scheme: "https", StatusCode: code, Title: title}
}

func TestFirstRunBaseline(t *testing.T) {
	newSnap := model.Snapshot{TargetID: "x", Root: "example.com", TakenAt: time.Now(),
		Assets: []model.Asset{sub("api.example.com"), sub("www.example.com")}}
	d := Compare(model.Snapshot{}, newSnap)
	if d.NewCount() != 2 {
		t.Fatalf("baseline: expected 2 new, got %d", d.NewCount())
	}
}

func TestDetectsNewSubdomain(t *testing.T) {
	old := model.Snapshot{Assets: []model.Asset{sub("www.example.com")}}
	new := model.Snapshot{Assets: []model.Asset{sub("www.example.com"), sub("admin.example.com")}}
	d := Compare(old, new)
	if d.NewCount() != 1 {
		t.Fatalf("expected 1 new, got %d", d.NewCount())
	}
	if d.Changes[0].Type != model.ChangeNew || d.Changes[0].Key != "admin.example.com" {
		t.Fatalf("expected new admin.example.com, got %+v", d.Changes[0])
	}
}

func TestDetectsModifiedStatus(t *testing.T) {
	old := model.Snapshot{Assets: []model.Asset{ep("api.example.com", 403, "Forbidden")}}
	new := model.Snapshot{Assets: []model.Asset{ep("api.example.com", 200, "Welcome")}}
	d := Compare(old, new)
	if len(d.Changes) != 1 || d.Changes[0].Type != model.ChangeModified {
		t.Fatalf("expected 1 modified, got %+v", d.Changes)
	}
	// 403 -> 200 with a title change is exactly the "a new service went live" signal.
	if !contains(d.Changes[0].Fields, "status_code") || !contains(d.Changes[0].Fields, "title") {
		t.Fatalf("expected status_code+title changed, got %v", d.Changes[0].Fields)
	}
}

func TestDetectsRemoved(t *testing.T) {
	old := model.Snapshot{Assets: []model.Asset{sub("old.example.com"), sub("www.example.com")}}
	new := model.Snapshot{Assets: []model.Asset{sub("www.example.com")}}
	d := Compare(old, new)
	if len(d.Changes) != 1 || d.Changes[0].Type != model.ChangeRemoved {
		t.Fatalf("expected 1 removed, got %+v", d.Changes)
	}
}

func TestOrderingNewBeforeRemoved(t *testing.T) {
	old := model.Snapshot{Assets: []model.Asset{sub("gone.example.com")}}
	new := model.Snapshot{Assets: []model.Asset{sub("fresh.example.com")}}
	d := Compare(old, new)
	if d.Changes[0].Type != model.ChangeNew {
		t.Fatalf("expected NEW ordered first, got %s", d.Changes[0].Type)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
