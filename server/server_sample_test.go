package main

import (
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/server/internal/store"
)

// The server measures its own machine only while nothing else does. An agent
// that stops reporting has to hand the row back: the machine is plainly up —
// this code is what serves the page saying so — and a row that reads offline
// there pages somebody for nothing.
func TestServerSampleLoopYieldsToALiveAgentAndTakesTheRowBack(t *testing.T) {
	ts := newTestServer(t)
	if err := ts.app.db.EnsureServerHost("linux"); err != nil {
		t.Fatal(err)
	}
	// metric_samples is keyed (host, ts, resolution) at second granularity and
	// written with INSERT OR REPLACE, so the ticks have to be spelled out apart
	// rather than left to land in whatever second the test runs in.
	tick := time.Now().UTC()
	next := tick.Add(30 * time.Second)
	count := func() int {
		var n int
		ts.app.db.SQL().QueryRow(`SELECT COUNT(*) FROM metric_samples WHERE host_id = ?`,
			store.ServerHostID).Scan(&n)
		return n
	}

	// An agent reporting now owns the row.
	if err := ts.app.db.TouchHost(store.ServerHostID, "0.9.9"); err != nil {
		t.Fatal(err)
	}
	ts.app.sampleServerHost(tick)
	if count() != 0 {
		t.Fatal("the server sampled a row a live agent is reporting for")
	}

	// The same agent, gone quiet well past the row's offline window.
	h, err := ts.app.db.GetHost(store.ServerHostID)
	if err != nil {
		t.Fatal(err)
	}
	stale := tick.Add(-time.Duration(h.OfflineAfterSecs)*time.Second - time.Minute)
	if _, err := ts.app.db.SQL().Exec(`UPDATE hosts SET last_seen_at = ? WHERE id = ?`,
		stale.Format(time.RFC3339Nano), store.ServerHostID); err != nil {
		t.Fatal(err)
	}
	ts.app.sampleServerHost(tick)
	if count() != 1 {
		t.Fatalf("samples after the agent went quiet = %d, want 1", count())
	}

	// Taking the row over drops the version, so the next tick keeps sampling
	// rather than reading the dead agent as live again.
	h, err = ts.app.db.GetHost(store.ServerHostID)
	if err != nil {
		t.Fatal(err)
	}
	if h.AgentVersion != "" {
		t.Errorf("agent_version = %q, want it cleared", h.AgentVersion)
	}
	ts.app.sampleServerHost(next)
	if count() != 2 {
		t.Fatalf("samples on the tick after taking over = %d, want 2", count())
	}
}
