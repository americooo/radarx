package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/americooo/radarx/internal/model"
)

// maxSeenCertIDs bounds how many previously-seen crt.sh certificate IDs we
// remember — enough to dedupe across scan cycles without the state value
// growing unbounded forever.
const maxSeenCertIDs = 2000

// crtshEntry is one row from crt.sh's JSON output. internal/engine has its
// own copy of this shape (certtransparency.go) — modules never imports
// engine, so this is a small local duplicate of the same wire format, plus
// the fields (ID, IssuerName) this module additionally needs for dedup and
// attribution.
type crtshEntry struct {
	ID         int64  `json:"id"`
	IssuerName string `json:"issuer_name"`
	CommonName string `json:"common_name"`
	NameValue  string `json:"name_value"`
	EntryTS    string `json:"entry_timestamp"`
	NotBefore  string `json:"not_before"`
	NotAfter   string `json:"not_after"`
}

// CTMonitorModule watches crt.sh's Certificate Transparency log index for
// certificates newly issued under a target's root domain — a purely passive
// signal (no active scan, no DNS resolution): "this subdomain name showed up
// in a CT log," nothing more. Whether it actually resolves/serves anything
// is left to the next ordinary scan cycle.
//
// fetch is injectable so tests never touch the real crt.sh service.
type CTMonitorModule struct {
	fetch func(ctx context.Context, url string) ([]byte, error)
}

func init() {
	Register(&CTMonitorModule{
		fetch: defaultCTFetch,
	})
}

func (m *CTMonitorModule) Name() string       { return "ct-monitor" }
func (m *CTMonitorModule) Category() Category { return CategoryDiscovery }
func (m *CTMonitorModule) Trigger() Trigger   { return TriggerScheduled }

// Run queries crt.sh once for target.Root, and emits a Finding for every
// certificate ID it hasn't reported before. asset is ignored (zero-value) —
// TriggerScheduled modules operate on the whole target, not one asset.
func (m *CTMonitorModule) Run(ctx context.Context, target model.Target, asset model.Asset, state State) (<-chan model.Finding, error) {
	out := make(chan model.Finding, 1)

	go func() {
		defer close(out)

		if target.Root == "" {
			return
		}

		reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()

		url := fmt.Sprintf("https://crt.sh/?q=%%25.%s&output=json", target.Root)
		body, err := m.fetch(reqCtx, url)
		if err != nil {
			return
		}
		if len(strings.TrimSpace(string(body))) == 0 {
			return
		}

		var entries []crtshEntry
		if err := json.Unmarshal(body, &entries); err != nil {
			return
		}
		if len(entries) == 0 {
			return
		}

		seen := loadSeenCertIDs(state)

		var newIDs []int64
		for _, e := range entries {
			if e.ID == 0 {
				continue
			}
			if _, ok := seen[e.ID]; ok {
				continue
			}
			newIDs = append(newIDs, e.ID)
			seen[e.ID] = struct{}{}

			issuer := strings.TrimSpace(e.IssuerName)
			if issuer == "" {
				issuer = "unknown issuer"
			}

			for _, name := range strings.Split(e.NameValue, "\n") {
				host := strings.ToLower(strings.TrimSpace(name))
				host = strings.TrimPrefix(host, "*.")
				if host == "" {
					continue
				}

				finding := model.Finding{
					Module:      m.Name(),
					Severity:    model.SeverityInfo,
					Asset:       model.Asset{Kind: model.KindSubdomain, Key: host, Host: host},
					Title:       "New certificate observed in CT logs",
					Description: fmt.Sprintf("%s issued a certificate covering %s", issuer, host),
					Evidence:    fmt.Sprintf("crt.sh id=%d", e.ID),
					TakenAt:     time.Now().UTC(),
				}
				select {
				case out <- finding:
				case <-ctx.Done():
					return
				}
			}
		}

		if len(newIDs) > 0 {
			saveSeenCertIDs(state, seen)
		}
	}()

	return out, nil
}

// loadSeenCertIDs decodes the module's remembered certificate IDs from
// state, returning an empty set if nothing has been stored yet (or the
// stored value is unreadable).
func loadSeenCertIDs(state State) map[int64]struct{} {
	seen := make(map[int64]struct{})

	raw, ok, err := state.Get("seen_cert_ids")
	if err != nil || !ok || raw == "" {
		return seen
	}

	var ids []int64
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return seen
	}
	for _, id := range ids {
		seen[id] = struct{}{}
	}
	return seen
}

// saveSeenCertIDs persists the set of seen certificate IDs, keeping only the
// most recent maxSeenCertIDs entries so the stored value doesn't grow
// forever.
func saveSeenCertIDs(state State, seen map[int64]struct{}) {
	ids := make([]int64, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	if len(ids) > maxSeenCertIDs {
		ids = ids[len(ids)-maxSeenCertIDs:]
	}

	encoded, err := json.Marshal(ids)
	if err != nil {
		return
	}
	_ = state.Set("seen_cert_ids", string(encoded))
}

func defaultCTFetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "RadarX/0.1 (+asset-monitor)")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("crt.sh returned %d", resp.StatusCode)
	}

	return io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
}
