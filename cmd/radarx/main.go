// Command radarx is the RadarX CLI (MVP). It scans a target once, diffs the
// result against the last stored snapshot, prints new/changed/removed assets,
// and persists the new snapshot. The Wails GUI will wrap this same engine.
//
// Usage:
//
//	radarx add   example.com --label "Acme BBP" --interval 60
//	radarx scan  example.com
//	radarx list
//
// Authorization: only monitor scopes you are explicitly permitted to test.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/americooo/radarx/internal/diff"
	"github.com/americooo/radarx/internal/engine"
	"github.com/americooo/radarx/internal/model"
	"github.com/americooo/radarx/internal/notify"
	"github.com/americooo/radarx/internal/report"
	"github.com/americooo/radarx/internal/scheduler"
	"github.com/americooo/radarx/internal/scope"
	"github.com/americooo/radarx/internal/store"
	"github.com/americooo/radarx/internal/web"
)

// version is injected at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	if os.Args[1] == "version" || os.Args[1] == "--version" {
		fmt.Printf("radarx %s\n", version)
		return
	}

	st, err := openStore()
	must(err)

	switch os.Args[1] {
	case "add":
		cmdAdd(st, os.Args[2:])
	case "scan":
		cmdScan(st, os.Args[2:])
	case "list":
		cmdList(st)
	case "watch":
		cmdWatch(st, os.Args[2:])
	case "serve":
		cmdServe(st, os.Args[2:])
	case "report":
		cmdReport(st, os.Args[2:])
	case "history":
		cmdHistory(st, os.Args[2:])
	case "test-telegram":
		cmdTestTelegram()
	default:
		usage()
		os.Exit(2)
	}
}

func openStore() (store.Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return store.NewSQLiteStore(filepath.Join(home, ".radarx", "radarx.db"))
}

// scopePath returns the path to the operator's scope file, ~/.radarx/scope.txt.
// This lives alongside the store's own ~/.radarx directory.
func scopePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".radarx", "scope.txt"), nil
}

// checkInScope is the single gate a scan must pass before it's allowed to
// touch the network: the scope file must exist and root must be covered by
// it. It's kept separate from cmdScan so it can be unit tested directly.
func checkInScope(path, root string) error {
	return scope.CheckPath(path, root)
}

// confirmScope asks the operator to explicitly authorize scanning root. It
// reads a single line from stdin and only treats "y"/"yes" (case-insensitive)
// as consent.
func confirmScope(root string) bool {
	fmt.Printf("Ushbu domenni (%s) scan qilishga ruxsatingiz bormi (o'zingizning infratuzilmangiz yoki ruxsat berilgan bug-bounty scope)? [y/N]: ", root)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes"
}

func cmdAdd(st store.Store, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: radarx add <root-domain> [--label ..] [--interval ..]")
		os.Exit(2)
	}
	// Take the domain as the first positional, then parse the rest as flags.
	// Go's flag package stops at the first non-flag token, so pulling the
	// domain out first lets flags appear in any order after it.
	root := strings.ToLower(args[0])

	fs := flag.NewFlagSet("add", flag.ExitOnError)
	label := fs.String("label", "", "human label / program name")
	interval := fs.Int("interval", 60, "scan interval in minutes")
	_ = fs.Parse(args[1:])

	sp, err := scopePath()
	must(err)

	if _, statErr := os.Stat(sp); os.IsNotExist(statErr) {
		if !confirmScope(root) {
			fmt.Println("bekor qilindi: scope tasdiqlanmadi, hech narsa saqlanmadi.")
			return
		}
		must(os.MkdirAll(filepath.Dir(sp), 0o755))
		must(os.WriteFile(sp, []byte(root+"\n"), 0o644))
	} else {
		sc, loadErr := scope.Load(sp)
		must(loadErr)
		if !sc.Allows(root) {
			fmt.Printf("%s joriy scope'da yo'q.\n", root)
			if !confirmScope(root) {
				fmt.Println("bekor qilindi: scope tasdiqlanmadi, hech narsa saqlanmadi.")
				return
			}
			f, openErr := os.OpenFile(sp, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
			must(openErr)
			_, writeErr := f.WriteString(root + "\n")
			must(writeErr)
			must(f.Close())
		}
	}

	t := model.Target{
		ID:        model.SlugID(root),
		Root:      root,
		Label:     *label,
		Enabled:   true,
		IntervalM: *interval,
		AddedAt:   time.Now().UTC(),
	}
	must(st.SaveTarget(t))
	fmt.Printf("added target %s (id=%s, interval=%dm)\n", t.Root, t.ID, t.IntervalM)
}

func cmdScan(st store.Store, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: radarx scan <root-domain> [--ct]")
		os.Exit(2)
	}
	root := strings.ToLower(args[0])
	id := model.SlugID(root)

	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	useCT := fs.Bool("ct", false, "also pull subdomains from Certificate Transparency (crt.sh)")
	ports := fs.Bool("ports", false, "run a light TCP port scan on each live host")
	_ = fs.Parse(args[1:])

	// Load target (or create an ephemeral one so `scan` works before `add`).
	targets, _ := st.ListTargets()
	var t model.Target
	found := false
	for _, x := range targets {
		if x.ID == id {
			t, found = x, true
			break
		}
	}
	if !found {
		t = model.Target{ID: id, Root: root, Enabled: true, IntervalM: 60, AddedAt: time.Now().UTC()}
	}

	sp, err := scopePath()
	must(err)
	if _, statErr := os.Stat(sp); os.IsNotExist(statErr) {
		fmt.Fprintf(os.Stderr, "scope hali sozlanmagan, avval `radarx add %s` orqali scope'ga qo'shing\n", t.Root)
		return
	}
	if err := checkInScope(sp, t.Root); err != nil {
		fmt.Fprintf(os.Stderr, "SCOPE VIOLATION: %s scope tashqarisida, scan qilinmaydi (%v)\n", t.Root, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Printf("scanning %s ...\n", root)
	start := time.Now()

	snap := model.Snapshot{TargetID: t.ID, Root: t.Root, TakenAt: time.Now().UTC()}
	for ev := range engine.ScanStream(ctx, t, engine.ScanOptions{Workers: 40, UseCertLogs: *useCT, ScanPorts: *ports}) {
		if ev.Asset != nil {
			fmt.Printf("  + %s\n", ev.Asset.Key)
			snap.Assets = append(snap.Assets, *ev.Asset)
		}
	}
	fmt.Printf("discovered %d assets in %s\n", len(snap.Assets), time.Since(start).Round(time.Millisecond))

	prev, hadPrev, err := st.LatestSnapshot(id)
	must(err)

	d := diff.Compare(prev, snap)
	printDiff(d, hadPrev)

	must(st.SaveSnapshot(snap))
	t.LastScanAt = snap.TakenAt
	must(st.SaveTarget(t))
}

func cmdList(st store.Store) {
	targets, err := st.ListTargets()
	must(err)
	if len(targets) == 0 {
		fmt.Println("no targets. add one:  radarx add example.com")
		return
	}
	for _, t := range targets {
		last := "never"
		if !t.LastScanAt.IsZero() {
			last = t.LastScanAt.Format(time.RFC3339)
		}
		fmt.Printf("%-30s interval=%-5s last=%s\n", t.Root, fmt.Sprintf("%dm", t.IntervalM), last)
	}
}

// cmdWatch runs the background scheduler until interrupted (Ctrl+C).
func cmdWatch(st store.Store, args []string) {
	fs := flag.NewFlagSet("watch", flag.ExitOnError)
	workers := fs.Int("workers", 40, "subdomain resolution concurrency")
	minPeriod := fs.Duration("min-period", 5*time.Minute, "minimum time between scans of any target")
	webAddr := fs.String("web", "", "also serve the dashboard at this address, e.g. 127.0.0.1:7777")
	_ = fs.Parse(args)

	if _, ok := notify.NewTelegramFromEnv(); ok {
		fmt.Println("telegram notifications: enabled")
	} else {
		fmt.Println("telegram notifications: disabled (set RADARX_TG_TOKEN and RADARX_TG_CHAT_ID)")
	}
	n := notify.Default()
	sch := scheduler.New(st, n, scheduler.Options{
		Workers:   *workers,
		MinPeriod: *minPeriod,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Optional dashboard alongside the scheduler.
	var httpSrv *http.Server
	if *webAddr != "" {
		httpSrv = &http.Server{Addr: *webAddr, Handler: web.New(st).Handler()}
		go func() {
			fmt.Printf("dashboard: http://%s\n", *webAddr)
			if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Fprintln(os.Stderr, "dashboard error:", err)
			}
		}()
	}

	fmt.Println("radarx watching — Ctrl+C to stop")
	err := sch.Run(ctx)

	if httpSrv != nil {
		shutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutCtx)
	}
	if err != nil && err != context.Canceled {
		must(err)
	}
	fmt.Println("stopped.")
}

// cmdServe starts the local web dashboard.
func cmdServe(st store.Store, args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:7777", "address to bind the dashboard")
	_ = fs.Parse(args)

	srv := web.New(st)
	httpSrv := &http.Server{Addr: *addr, Handler: srv.Handler()}

	go func() {
		fmt.Printf("dashboard: http://%s\n", *addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			must(err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutCtx)
	fmt.Println("dashboard stopped.")
}

// cmdReport writes a markdown asset inventory for a target.
func cmdReport(st store.Store, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: radarx report <root-domain> [--out file.md]")
		os.Exit(2)
	}
	root := strings.ToLower(args[0])
	id := model.SlugID(root)

	fs := flag.NewFlagSet("report", flag.ExitOnError)
	out := fs.String("out", "", "write to file instead of stdout")
	_ = fs.Parse(args[1:])

	targets, _ := st.ListTargets()
	var t model.Target
	for _, x := range targets {
		if x.ID == id {
			t = x
			break
		}
	}
	if t.ID == "" {
		t = model.Target{ID: id, Root: root}
	}

	snap, ok, err := st.LatestSnapshot(id)
	must(err)
	if !ok {
		fmt.Fprintf(os.Stderr, "no snapshot for %s yet — run: radarx scan %s\n", root, root)
		os.Exit(1)
	}

	md := report.Inventory(t, snap)
	if *out == "" {
		fmt.Print(md)
		return
	}
	must(os.WriteFile(*out, []byte(md), 0o644))
	fmt.Printf("wrote %s\n", *out)
}

// cmdHistory prints the change history for a target across all snapshots.
func cmdHistory(st store.Store, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: radarx history <root-domain>")
		os.Exit(2)
	}
	id := model.SlugID(args[0])
	snaps, err := st.ListSnapshots(id)
	must(err)
	if len(snaps) == 0 {
		fmt.Println("no history yet.")
		return
	}
	fmt.Printf("%d snapshots\n", len(snaps))
	for i := 1; i < len(snaps); i++ {
		d := diff.Compare(snaps[i-1], snaps[i])
		fmt.Printf("  %s  →  %d new, %d changes\n",
			snaps[i].TakenAt.Format("2006-01-02 15:04"), d.NewCount(), len(d.Changes))
	}
}

// cmdTestTelegram sends a test message so the operator can confirm their
// bot token and chat id are correct before relying on alerts.
func cmdTestTelegram() {
	tg, ok := notify.NewTelegramFromEnv()
	if !ok {
		fmt.Fprintln(os.Stderr, "set RADARX_TG_TOKEN and RADARX_TG_CHAT_ID first")
		os.Exit(2)
	}
	if err := tg.SendRaw("✅ RadarX test message — Telegram alerts are working."); err != nil {
		must(err)
	}
	fmt.Println("sent. check your Telegram.")
}

func printDiff(d model.DiffResult, hadPrev bool) {
	if !hadPrev {
		fmt.Printf("baseline snapshot recorded (%d assets). future scans will diff against this.\n", len(d.Changes))
		return
	}
	if !d.HasSignal() {
		fmt.Println("no changes since last scan.")
		return
	}
	fmt.Printf("\n=== CHANGES (%d new) ===\n", d.NewCount())
	for _, c := range d.Changes {
		switch c.Type {
		case model.ChangeNew:
			fmt.Printf("  [+] NEW      %-9s %s\n", c.Kind, describe(c.After))
		case model.ChangeModified:
			fmt.Printf("  [~] CHANGED  %-9s %s  (%s)\n", c.Kind, c.Key, strings.Join(c.Fields, ","))
		case model.ChangeRemoved:
			fmt.Printf("  [-] REMOVED  %-9s %s\n", c.Kind, c.Key)
		}
	}
}

func describe(a *model.Asset) string {
	if a == nil {
		return ""
	}
	switch a.Kind {
	case model.KindEndpoint:
		s := fmt.Sprintf("%s (%d)", a.Key, a.StatusCode)
		if a.Title != "" {
			s += " — " + a.Title
		}
		return s
	case model.KindCert:
		return fmt.Sprintf("%s CN=%s exp=%s", a.Host, a.CertCN, a.CertExpiry)
	default:
		return a.Key
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `RadarX — attack surface monitor

  radarx add     <root-domain> [--label ..] [--interval <min>]
  radarx scan    <root-domain> [--ct] [--ports]
  radarx watch   [--min-period 5m] [--workers 40] [--web 127.0.0.1:7777]
  radarx serve   [--addr 127.0.0.1:7777]
  radarx report  <root-domain> [--out file.md]
  radarx history <root-domain>
  radarx list
  radarx test-telegram

Telegram alerts (optional):
  export RADARX_TG_TOKEN=<bot token from @BotFather>
  export RADARX_TG_CHAT_ID=<your chat id>

Only monitor scopes you are authorized to test.`)
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
