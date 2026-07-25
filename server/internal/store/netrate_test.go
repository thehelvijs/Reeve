package store

import (
	"testing"
	"time"

	"github.com/thehelvijs/Reeve/contracts"
)

func TestHostNetRate(t *testing.T) {
	db := openTemp(t)
	host, err := db.CreateHost("h", "linux", "", "hh", 600)
	if err != nil {
		t.Fatal(err)
	}
	base := time.Now().UTC()

	// Single sample: no rate.
	if err := db.InsertHostMetric(host.ID, contracts.HostMetrics{NetRx: 1000, NetTx: 2000}, base); err != nil {
		t.Fatal(err)
	}
	if _, ok := db.HostNetRate(host.ID); ok {
		t.Error("single sample should return ok=false")
	}

	// Second sample 10s later, +1 MiB rx.
	if err := db.InsertHostMetric(host.ID, contracts.HostMetrics{NetRx: 1000 + 1048576, NetTx: 2000}, base.Add(10*time.Second)); err != nil {
		t.Fatal(err)
	}
	rate, ok := db.HostNetRate(host.ID)
	if !ok {
		t.Fatal("two samples should return ok=true")
	}
	if rate < 104857.5 || rate > 104857.7 {
		t.Errorf("rate = %v, want ~104857.6", rate)
	}

	// Counter reset: newest rx below previous, tx flat -> rate 0.
	if err := db.InsertHostMetric(host.ID, contracts.HostMetrics{NetRx: 5, NetTx: 2000}, base.Add(20*time.Second)); err != nil {
		t.Fatal(err)
	}
	rate, ok = db.HostNetRate(host.ID)
	if !ok {
		t.Fatal("ok=true expected after reset")
	}
	if rate != 0 {
		t.Errorf("rate after counter reset = %v, want 0", rate)
	}
}
