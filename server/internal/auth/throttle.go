package auth

import (
	"sync"
	"time"
)

// Defaults for login throttling: five misses inside the window locks the key
// out for the lockout period. Chosen so a human typo streak recovers quickly
// while an online guessing run is capped at ~20 attempts an hour per key.
const (
	DefaultThrottleMax     = 5
	DefaultThrottleWindow  = 15 * time.Minute
	DefaultThrottleLockout = 15 * time.Minute
)

// pruneAt bounds the entry map so unbounded distinct keys (spoofed emails or
// XFF values) cannot grow it without limit.
const pruneAt = 4096

type throttleEntry struct {
	failures    int
	first       time.Time
	lockedUntil time.Time
}

// Throttle rate-limits repeated failures per key (an IP or an email). It is
// safe for concurrent use and takes an injected clock so tests stay
// deterministic.
type Throttle struct {
	mu      sync.Mutex
	max     int
	window  time.Duration
	lockout time.Duration
	now     func() time.Time
	entries map[string]*throttleEntry
}

// NewThrottle builds a Throttle. A nil now uses the wall clock.
func NewThrottle(max int, window, lockout time.Duration, now func() time.Time) *Throttle {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Throttle{
		max:     max,
		window:  window,
		lockout: lockout,
		now:     now,
		entries: map[string]*throttleEntry{},
	}
}

// NewLoginThrottle builds a Throttle with the default login policy.
func NewLoginThrottle() *Throttle {
	return NewThrottle(DefaultThrottleMax, DefaultThrottleWindow, DefaultThrottleLockout, nil)
}

// RetryAfter returns how long key must wait before its next attempt, and
// whether it is currently locked out.
func (t *Throttle) RetryAfter(key string) (time.Duration, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	e, ok := t.entries[key]
	if !ok {
		return 0, false
	}
	now := t.now()
	if e.lockedUntil.After(now) {
		return e.lockedUntil.Sub(now), true
	}
	return 0, false
}

// Fail records a failed attempt for key, locking it out once the threshold is
// reached inside the window.
func (t *Throttle) Fail(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	if len(t.entries) >= pruneAt {
		t.pruneLocked(now)
	}
	e, ok := t.entries[key]
	if !ok || now.Sub(e.first) > t.window {
		t.entries[key] = &throttleEntry{failures: 1, first: now}
		return
	}
	// A locked-out key stays on its existing deadline; further misses during
	// the lockout must not extend it, or an attacker could lock a user out
	// indefinitely.
	if e.lockedUntil.After(now) {
		return
	}
	e.failures++
	if e.failures >= t.max {
		e.lockedUntil = now.Add(t.lockout)
	}
}

// Reset clears a key's failures after a successful attempt.
func (t *Throttle) Reset(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.entries, key)
}

// pruneLocked drops entries whose window and lockout have both elapsed.
func (t *Throttle) pruneLocked(now time.Time) {
	for k, e := range t.entries {
		if e.lockedUntil.After(now) {
			continue
		}
		if now.Sub(e.first) > t.window {
			delete(t.entries, k)
		}
	}
}
