package notify

import (
	"strings"
	"testing"
	"time"

	"github.com/americooo/radarx/internal/model"
)

func sampleFinding() model.Finding {
	return model.Finding{
		Module:      "subdomain-takeover",
		Severity:    model.SeverityHigh,
		Asset:       model.Asset{Kind: model.KindSubdomain, Key: "assets.example.com", Host: "assets.example.com"},
		Title:       "Possible Amazon S3 subdomain takeover",
		Description: "assets.example.com has a CNAME pointing at a dangling S3 bucket.",
		Evidence:    "CNAME=dangling.s3.amazonaws.com body=\"NoSuchBucket\"",
		TakenAt:     time.Now().UTC(),
	}
}

func TestFormatFindingContainsKeyFields(t *testing.T) {
	f := sampleFinding()
	out := FormatFinding(f)

	for _, want := range []string{f.Module, string(f.Severity), f.Title, f.Asset.Key, f.Evidence} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected FormatFinding output to contain %q, got:\n%s", want, out)
		}
	}
}

// recordingNotifier captures whatever it's asked to notify, so tests can
// assert on fan-out behaviour without touching the network or the desktop.
type recordingNotifier struct {
	name        string
	failFinding bool
	findings    []model.Finding
}

func (r *recordingNotifier) Name() string { return r.name }

func (r *recordingNotifier) Notify(d model.DiffResult) error { return nil }

func (r *recordingNotifier) NotifyFinding(f model.Finding) error {
	if r.failFinding {
		return errTest
	}
	r.findings = append(r.findings, f)
	return nil
}

var errTest = &testError{"boom"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

func TestConsoleNotifyFinding(t *testing.T) {
	if err := (Console{}).NotifyFinding(sampleFinding()); err != nil {
		t.Fatalf("Console.NotifyFinding returned error: %v", err)
	}
}

func TestMultiNotifyFindingFansOutAndCollectsErrors(t *testing.T) {
	ok1 := &recordingNotifier{name: "ok1"}
	failing := &recordingNotifier{name: "failing", failFinding: true}
	ok2 := &recordingNotifier{name: "ok2"}

	m := Multi{Notifiers: []Notifier{ok1, failing, ok2}}
	f := sampleFinding()

	err := m.NotifyFinding(f)
	if err == nil {
		t.Fatal("expected an error because one notifier failed")
	}
	if !strings.Contains(err.Error(), "failing") {
		t.Fatalf("expected error to mention the failing notifier, got: %v", err)
	}

	if len(ok1.findings) != 1 || len(ok2.findings) != 1 {
		t.Fatalf("expected both healthy notifiers to receive the finding, got ok1=%d ok2=%d", len(ok1.findings), len(ok2.findings))
	}
}
