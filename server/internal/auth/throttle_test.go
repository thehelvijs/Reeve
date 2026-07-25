package auth

import (
	"fmt"
	"testing"
	"time"
)

// clock is a manually advanced clock so throttle tests never touch wall time.
type clock struct{ t time.Time }

func (c *clock) now() time.Time      { return c.t }
func (c *clock) add(d time.Duration) { c.t = c.t.Add(d) }

func newTestThrottle() (*Throttle, *clock) {
	c := &clock{t: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	return NewThrottle(3, 10*time.Minute, 5*time.Minute, c.now), c
}

func TestThrottleLocksAfterMaxFailures(t *testing.T) {
	th, _ := newTestThrottle()
	for i := 0; i < 2; i++ {
		th.Fail("ip")
		if _, locked := th.RetryAfter("ip"); locked {
			t.Fatalf("locked out after %d failures, want lockout only at 3", i+1)
		}
	}
	th.Fail("ip")
	retry, locked := th.RetryAfter("ip")
	if !locked {
		t.Fatal("not locked out after 3 failures")
	}
	if retry != 5*time.Minute {
		t.Errorf("retry after = %v, want 5m", retry)
	}
}

func TestThrottleUnlocksWhenLockoutElapses(t *testing.T) {
	th, c := newTestThrottle()
	for i := 0; i < 3; i++ {
		th.Fail("ip")
	}
	c.add(5 * time.Minute)
	if _, locked := th.RetryAfter("ip"); locked {
		t.Fatal("still locked out once the lockout elapsed")
	}
}

func TestThrottleForgetsFailuresOlderThanWindow(t *testing.T) {
	th, c := newTestThrottle()
	th.Fail("ip")
	th.Fail("ip")
	c.add(11 * time.Minute)
	th.Fail("ip")
	if _, locked := th.RetryAfter("ip"); locked {
		t.Fatal("stale failures outside the window still counted toward lockout")
	}
}

func TestThrottleResetClearsFailures(t *testing.T) {
	th, _ := newTestThrottle()
	th.Fail("ip")
	th.Fail("ip")
	th.Reset("ip")
	th.Fail("ip")
	if _, locked := th.RetryAfter("ip"); locked {
		t.Fatal("locked out after a reset plus one failure")
	}
}

// Failures during a lockout must not extend it, otherwise anyone could keep a
// victim's account locked forever by guessing on a loop.
func TestThrottleLockoutIsNotExtendedByMoreFailures(t *testing.T) {
	th, c := newTestThrottle()
	for i := 0; i < 3; i++ {
		th.Fail("user@example.com")
	}
	c.add(4 * time.Minute)
	th.Fail("user@example.com")
	retry, locked := th.RetryAfter("user@example.com")
	if !locked {
		t.Fatal("expected still locked one minute before the deadline")
	}
	if retry != time.Minute {
		t.Errorf("retry after = %v, want 1m (deadline unchanged)", retry)
	}
}

func TestThrottleKeysAreIndependent(t *testing.T) {
	th, _ := newTestThrottle()
	for i := 0; i < 3; i++ {
		th.Fail("a")
	}
	if _, locked := th.RetryAfter("b"); locked {
		t.Fatal("locking key a also locked key b")
	}
}

func TestThrottlePrunesStaleEntries(t *testing.T) {
	th, c := newTestThrottle()
	for i := 0; i < pruneAt; i++ {
		th.Fail(fmt.Sprintf("key-%d", i))
	}
	c.add(11 * time.Minute)
	th.Fail("fresh")
	th.mu.Lock()
	n := len(th.entries)
	th.mu.Unlock()
	if n > 1 {
		t.Errorf("entries after prune = %d, want 1 (only the fresh key)", n)
	}
}
