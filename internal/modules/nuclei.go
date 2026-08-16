package modules

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"time"

	"github.com/americooo/radarx/internal/model"
)

// NucleiModule wraps the external `nuclei` binary (projectdiscovery.io) as a
// vuln-signal module: it runs nuclei's default template set against a single
// endpoint asset and turns each JSONL match into a Finding.
//
// RadarX has otherwise been zero-dependency Go — nuclei is the one
// deliberate exception. It is optional: if the binary isn't on PATH, Run
// silently emits nothing (no error, no noise). Users who want this signal
// install nuclei themselves: https://github.com/projectdiscovery/nuclei
//
// lookPath and runNuclei are injectable so tests never exec a real binary.
type NucleiModule struct {
	lookPath  func(file string) (string, error)
	runNuclei func(ctx context.Context, targetURL string) (io.Reader, error)
}

func init() {
	Register(&NucleiModule{
		lookPath:  exec.LookPath,
		runNuclei: defaultRunNuclei,
	})
}

func (m *NucleiModule) Name() string       { return "nuclei" }
func (m *NucleiModule) Category() Category { return CategoryVulnSignal }
func (m *NucleiModule) Trigger() Trigger   { return TriggerNewAssetsOnly }

// nucleiResult mirrors the subset of nuclei's JSONL output we care about —
// one JSON object per line, one line per matched template.
type nucleiResult struct {
	TemplateID string `json:"template-id"`
	Info       struct {
		Name        string `json:"name"`
		Severity    string `json:"severity"`
		Description string `json:"description"`
	} `json:"info"`
	MatchedAt string `json:"matched-at"`
}

// Run only applies to KindEndpoint assets (HTTP-probed, asset.Key is a full
// URL) — every other asset kind is skipped immediately.
func (m *NucleiModule) Run(ctx context.Context, target model.Target, asset model.Asset, state State) (<-chan model.Finding, error) {
	out := make(chan model.Finding, 1)

	go func() {
		defer close(out)

		if asset.Kind != model.KindEndpoint || asset.Key == "" {
			return
		}

		if _, err := m.lookPath("nuclei"); err != nil {
			// nuclei isn't installed — skip quietly, never error.
			return
		}

		r, err := m.runNuclei(ctx, asset.Key)
		if err != nil || r == nil {
			return
		}

		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := bytes.TrimSpace(scanner.Bytes())
			if len(line) == 0 {
				continue
			}

			var res nucleiResult
			if err := json.Unmarshal(line, &res); err != nil {
				continue
			}

			finding := model.Finding{
				Module:      m.Name(),
				Severity:    mapSeverity(res.Info.Severity),
				Asset:       asset,
				Title:       res.Info.Name,
				Description: res.Info.Description,
				Evidence:    res.MatchedAt,
				TakenAt:     time.Now().UTC(),
			}
			select {
			case out <- finding:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

// mapSeverity translates nuclei's severity strings to model.Severity.
// The names match 1:1; anything unrecognized defaults to info.
func mapSeverity(s string) model.Severity {
	switch model.Severity(s) {
	case model.SeverityLow, model.SeverityMedium, model.SeverityHigh, model.SeverityCritical:
		return model.Severity(s)
	default:
		return model.SeverityInfo
	}
}

// defaultRunNuclei shells out to the real nuclei binary. Flags:
//   - -u <url>       scan a single target URL
//   - -jsonl         newline-delimited JSON output (older nuclei releases
//     use -json-export <file> instead; -jsonl/-silent is
//     the most widely supported streaming-output combo)
//   - -silent        suppress banner/progress noise from stdout
//   - -rate-limit 10 stay a good citizen — matches RadarX's default
//     "never hammer the target" posture
//   - -timeout 8     per-request timeout, seconds
//   - -no-color      keep output parseable
//
// No -t is passed, so nuclei runs its default template set rather than a
// hand-picked aggressive/exploit category.
func defaultRunNuclei(ctx context.Context, targetURL string) (io.Reader, error) {
	cmd := exec.CommandContext(ctx, "nuclei",
		"-u", targetURL,
		"-jsonl",
		"-silent",
		"-rate-limit", "10",
		"-timeout", "8",
		"-no-color",
	)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	// nuclei's own errors/progress go to stderr; ignore rather than fail
	// the module on non-zero exit (nuclei exits non-zero on "no matches").
	_ = cmd.Run()

	return &buf, nil
}
