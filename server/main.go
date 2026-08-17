// Command server is the Reeve central server: catalog, auth, ingest,
// alerting, and the embedded web UI. LAN-only by design.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/crypto"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

// version is set at build time via -ldflags; defaults for dev.
var version = "dev"

// shutdownGrace bounds how long in-flight requests get to finish on SIGTERM.
const shutdownGrace = 15 * time.Second

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// `server users …` is a subcommand, parsed before the server's own flags.
	if len(os.Args) > 1 && os.Args[1] == "users" {
		return runUsersCLI(os.Stdout, envOr("REEVE_DB", "reeve.db"), os.Args[2:])
	}

	resetEmail := flag.String("reset-password", "", "set this account's password, then exit")
	resetValue := flag.String("password", "", "the new password for -reset-password")
	healthcheck := flag.Bool("healthcheck", false, "probe a running server's /healthz and exit")
	flag.Parse()

	cfg := config{
		Addr:         envOr("REEVE_ADDR", "127.0.0.1:7338"),
		PublicURL:    envOr("REEVE_PUBLIC_URL", ""),
		DBPath:       envOr("REEVE_DB", "reeve.db"),
		AvatarDir:    os.Getenv("REEVE_AVATAR_DIR"),
		IconDir:      os.Getenv("REEVE_ICON_DIR"),
		SessionTTL:   30 * 24 * time.Hour,
		CookieSecure: os.Getenv("REEVE_COOKIE_SECURE") == "true",
		Version:      version,
		LogRequests:  os.Getenv("REEVE_LOG_REQUESTS") != "false",

		TrustProxyHeaders: os.Getenv("REEVE_TRUST_PROXY") == "true",
		AllowedOrigins:    splitList(os.Getenv("REEVE_ALLOWED_ORIGINS")),
		SelfUpdate:        os.Getenv("REEVE_SELF_UPDATE") == "true",
	}

	// The container healthcheck runs this binary rather than adding curl to the
	// image, so it must answer before any database or key work.
	if *healthcheck {
		return probeHealth(cfg.Addr)
	}

	// Recovery runs before the master key is required: a locked-out admin must
	// be able to get back in even from a shell that lacks the key.
	if *resetEmail != "" {
		return resetPasswordCLI(cfg.DBPath, *resetEmail, *resetValue)
	}

	cipher, err := crypto.NewFromEnv()
	if err != nil {
		return fmt.Errorf("startup: %w (generate with: openssl rand -base64 32)", err)
	}
	if err := store.ApplyStagedRestore(cfg.DBPath); err != nil {
		return fmt.Errorf("apply staged restore: %w", err)
	}
	db, err := store.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	db.SetAuditMAC(cipher.MAC)
	a := &app{db: db, cipher: cipher, cfg: cfg, startedAt: time.Now().UTC(), agentFS: agentDistFS(), scriptFS: installScripts}
	if err := db.EnsureServerHost(runtime.GOOS); err != nil {
		return fmt.Errorf("startup: could not register self host: %w", err)
	}
	if err := a.syncSelfAgentToken(); err != nil {
		return fmt.Errorf("startup: could not publish the self-agent token: %w", err)
	}
	a.syncChannelFile()
	a.recordServerVersion()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var loops sync.WaitGroup
	for _, loop := range []func(context.Context){a.runRollupLoop, a.runAlertLoop, a.runDispatchLoop, a.runServerSampleLoop} {
		loops.Add(1)
		go func(f func(context.Context)) {
			defer loops.Done()
			f(ctx)
		}(loop)
	}

	srv := &http.Server{Addr: cfg.Addr, Handler: a.routes()}
	log.Printf("Reeve server %s listening on %s", version, cfg.Addr)
	serveErr := serve(ctx, srv)
	loops.Wait()
	return serveErr
}

// probeHealth GETs /healthz on a locally running server. A bind address of
// 0.0.0.0 or an empty host is dialed on loopback.
func probeHealth(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("healthcheck: bad address %q: %w", addr, err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://" + net.JoinHostPort(host, port) + "/healthz")
	if err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck: /healthz returned %d", resp.StatusCode)
	}
	return nil
}

// serve runs srv until ctx is cancelled, then drains in-flight requests within
// shutdownGrace so an upgrade never severs a request mid-write.
func serve(ctx context.Context, srv *http.Server) error {
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	log.Print("shutting down, draining in-flight requests")
	shutCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		return err
	}
	return nil
}

// every runs fn on an interval until ctx is cancelled.
func every(ctx context.Context, d time.Duration, fn func()) {
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fn()
		}
	}
}

// runServerSampleLoop records the server's own machine metrics, and heartbeats
// its host row, until an agent is installed on that machine.
//
// It stands down for good once one has reported: an agent reads the machine
// itself, while this reads it through the server's container, and two writers
// would fight over one filesystem snapshot. A machine whose agent is later
// removed then reads offline like any other, which is the honest answer — put
// the agent back to fix it.
func (a *app) runServerSampleLoop(ctx context.Context) {
	sample := func() {
		if h, err := a.db.GetHost(store.ServerHostID); err == nil && h.AgentVersion != "" {
			return
		}
		now := time.Now().UTC()
		m := a.sampleHost()
		if err := a.db.InsertHostMetric(store.ServerHostID, m, now); err != nil {
			log.Printf("server sample: %v", err)
		}
		// Nothing pushes for this row yet, so its filesystem snapshot has to be
		// written here or the page shows charts and no disks.
		if err := a.db.ReplaceHostDisks(store.ServerHostID, m.Disks, now); err != nil {
			log.Printf("server sample disks: %v", err)
		}
		if err := a.db.MarkHostSeen(store.ServerHostID); err != nil {
			log.Printf("server sample touch: %v", err)
		}
	}
	sample()
	every(ctx, 30*time.Second, sample)
}

// runRollupLoop periodically aggregates and prunes metric time-series, and
// closes out commands no agent ever answered.
func (a *app) runRollupLoop(ctx context.Context) {
	every(ctx, time.Minute, func() {
		if err := a.db.RollupAndPrune(time.Now().UTC(), a.db.EffectiveRetention()); err != nil {
			log.Printf("rollup: %v", err)
		}
		n, err := a.db.ExpireStaleCommands(time.Now().UTC())
		if err != nil {
			log.Printf("expiring stale commands: %v", err)
		}
		if n > 0 {
			log.Printf("expired %d command(s) no agent collected or answered in %s", n, store.CommandStaleAfter)
		}
	})
}

// runAlertLoop evaluates alert conditions on a ticker.
func (a *app) runAlertLoop(ctx context.Context) {
	every(ctx, 20*time.Second, func() { a.evaluateAlerts(time.Now().UTC()) })
}

// runDispatchLoop delivers due webhook payloads on a ticker.
func (a *app) runDispatchLoop(ctx context.Context) {
	every(ctx, 10*time.Second, func() { a.dispatchDue(time.Now().UTC()) })
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// splitList parses a comma-separated environment value, dropping blanks.
func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
