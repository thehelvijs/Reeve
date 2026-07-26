# Agent version tracking and server-driven rollout

## Problem

The agent already self-updates: hourly it compares its own sha256 against
`/dl/agent-linux-<arch>.sha256`, verifies an Ed25519 release signature, swaps the
binary atomically and exits for systemd to relaunch it
(`agent/update.go:114`, `agent/main.go:82`). Each push records the reporting
version in `hosts.agent_version` (`server/internal/store/telemetry.go:41`).

None of that is visible or controllable from the UI. The Hosts list prints the
version as grey text (`web/src/pages/Hosts.tsx:70`) and nothing compares it to
what the server publishes. An operator cannot see which hosts are behind, cannot
tell that a host has `REEVE_AUTO_UPDATE=false` and will never update, cannot
pause a rollout that is going badly, and cannot hurry one that matters.

Self-registration is already toggle-able and is out of scope. It is the
`signup_enabled` setting (`server/auth_handlers.go:15`), surfaced at
Settings → Sign-up (`web/src/pages/AdminSettings.tsx:54`), with an allowed-domain
list that admits listed domains while sign-up is closed.

## Outcome

An admin can answer "is my fleet on the current agent?" from the Hosts page
without leaving it, and can hold, resume, or force a rollout. The measurable
signal is the fleet rollup counts (`up_to_date` / `outdated` / `updating` /
`stalled` / `disabled`) served by `GET /api/v1/admin/agent-updates`: a healthy
upgrade drives `outdated` to zero within one rollout, and a bad one is visible as
a non-zero `stalled` with the rollout paused instead of a silently bricked fleet.

## Decisions

| Question | Decision |
| --- | --- |
| Policy scope | Fleet default plus a per-host override |
| Local `REEVE_AUTO_UPDATE=false` | Hard veto; the server can never force an update onto a host whose owner refused |
| Update trigger | Ingest ack carries `check_now`; updates land within one push interval (~15s) |
| Rollout pacing | Server-paced, N concurrent, and a stall halts the fleet |
| UI location | Existing Hosts / host detail / Settings pages, no new nav |

Precedence, highest first: local veto → per-host `on`/`off` → fleet default.

## Protocol

`contracts.PushProtocolVersion` goes 1 → 2. Producer, consumer and tests move in
the same commit; the server hard-rejects a mismatch at ingest.

```go
// contracts.Push gains:
AutoUpdateVetoed bool `json:"auto_update_vetoed"`

// New. Ingest replies 200 with this body instead of 204 No Content.
type PushAck struct {
    CheckNow bool `json:"check_now"`
}
```

### Why the hourly ticker stays

A protocol bump means that the moment the server upgrades, every v1 agent is
rejected at ingest with `400` and no ack body. If the server were the only
update trigger, those agents would never be told to check, never update, and
never speak v2 again — the upgrade would strand the entire fleet and every host
would need a hand reinstall.

So the ticker survives, demoted to a recovery path. It fires only when no valid
ack has arrived in `2 × REEVE_UPDATE_INTERVAL`. A healthy agent is server-paced
and the ticker never runs, so it cannot bypass rollout pacing. A rejected or
orphaned agent recovers itself within the hour. `REEVE_UPDATE_INTERVAL` keeps
its meaning; no existing behavior is removed.

Agent logic:

- On ack with `check_now` and no local veto: `go runSelfUpdate(...)`. The
  existing `updating` compare-and-swap guard drops overlapping runs.
- On ticker: return early if the last ack is fresher than
  `2 × UpdateInterval`, or if `cfg.AutoUpdate` is false. Otherwise run as today.
- A malformed or unreadable ack body is treated as no ack: it must not wedge the
  push loop, and it must not count as contact for the ticker's freshness check.

## Schema

`server/internal/store/migrations/0001_init.sql` is edited in place — the
project is pre-release and does not carry migrations yet.

```sql
-- hosts
auto_update        TEXT NOT NULL DEFAULT 'default'
                     CHECK (auto_update IN ('default','on','off')),
auto_update_vetoed INTEGER NOT NULL DEFAULT 0,
update_started_at  TEXT,
```

`update_started_at` non-null means the host holds a rollout slot. It is cleared
when the host reports the target version.

Settings rows, read through the existing `GetSetting` helpers:

| Key | Default | Bounds |
| --- | --- | --- |
| `agent_update.enabled` | `true` | — |
| `agent_update.concurrency` | `3` | 1–100 |
| `agent_update.stall_secs` | `900` | 1–86400 |

`stall_secs` is allowed down to 1 so the e2e suite can drive the stall path
without a fifteen-minute wait.

## Rollout state machine

New file `server/agent_update.go`, called from `handleIngest` after `storePush`.

1. Persist the reported `auto_update_vetoed`.
2. `check_now` is false when any of these hold: the host vetoed locally, the
   effective policy is off, the reported version already equals `a.cfg.Version`,
   or either version is `dev`. A dev build is never chased.
3. Otherwise the host is outdated and eligible. Live slots are hosts with a
   non-null `update_started_at` younger than `stall_secs`. If the host already
   holds a live slot, re-send `check_now` — the agent's guard dedupes. If the
   count of live slots is under `concurrency`, stamp `update_started_at` and
   send `check_now`.
4. A slot releases when the host reports the target version.
5. **A stall halts the fleet.** A host whose `update_started_at` is older than
   `stall_secs` while still on the old version is `stalled`, and while any host
   is stalled the server grants no new slots. That is what makes the first batch
   a genuine canary: if it never comes back, nothing else moves. Recovery is
   explicit — per-host "Update now", or fleet-wide "Resume rollout".

Derived `update_state` per host: `up_to_date`, `outdated`, `updating`,
`stalled`, `disabled` (vetoed or policy off), `unknown` (never reported, or a
`dev` version on either side).

## API

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/v1/hosts` | Gains `auto_update` and derived `update_state` |
| `PUT` | `/api/v1/admin/hosts/{id}/auto-update` | `{policy: "default"\|"on"\|"off"}` |
| `POST` | `/api/v1/admin/hosts/{id}/update-now` | Force-grants a slot, ignoring cap and pause |
| `GET` | `/api/v1/admin/agent-updates` | `{server_version, counts, paused, stalled}` |
| `POST` | `/api/v1/admin/agent-updates/resume` | Nulls `update_started_at` on every stalled host, unpausing the fleet |
| `GET`/`PUT` | `/api/v1/admin/settings` | Gains `agent_update: {enabled, concurrency, stall_secs}` |

The settings addition follows the existing partial-update pattern in
`handlePutSettings`, so one page section cannot clobber another. Every mutation
is admin-only, matching the rest of `/api/v1/admin/*`.

The anonymous portal payload is unchanged. It already omits `agent_version`
(`contracts/API.md:135`) and gains neither new field.

## UI

Follows `DESIGN.md`; no new colors or components.

**`web/src/pages/Hosts.tsx`** — a rollup line under the page header
("14 hosts · 12 on 0.2.0 · 1 updating · 1 stalled"), with a `Resume rollout`
button when the rollout is paused, admin-only. Per row, `agent 0.1.0` stays
plain grey text when the host is current and becomes a `Pill` otherwise:
`muted` for outdated and updating, `down` for stalled.

**`web/src/pages/HostInventory.tsx`** — a `Card` titled "Agent": reported
version, state, a policy select (Default / On / Off), an `Update now` button,
and a line reading "auto-update is disabled on this host" when the agent
reports a local veto, so the UI never promises an update that cannot happen.

**`web/src/pages/AdminSettings.tsx`** — an "Agent updates" section with the
fleet default toggle, concurrency, and stall timeout, matching the existing
Retention section's shape.

## Tests

Gate tests, in the same commits as the code.

Go:

- `contracts/contracts_test.go` — protocol version is 2; `PushAck` round-trips.
- `agent/push_test.go` — ack parsing; a malformed ack body is ignored and does
  not count as contact.
- `agent/update_test.go` — the local veto blocks a `check_now`; the ticker stays
  quiet while acks are fresh; the ticker fires once acks go stale.
- `server/ingest_handlers_test.go` — the ack body is served; a v1 push is
  rejected.
- `server/agent_update_test.go` (new) — slot grant; the concurrency cap holds;
  a stall pauses the whole fleet; resume clears it; per-host override beats the
  fleet default; a local veto beats both; a `dev` version on either side is a
  no-op.
- `server/internal/store/hosts_test.go` — the new columns and the slot queries.

### End-to-end

New `web/e2e/zz-agent-updates.spec.ts`, named to run after the specs that seed
the admin account and before `zz-profile.spec.ts`, which deletes it. The suite is
serial over one shared database.

Two edits to `web/playwright.config.ts` make this testable:

1. The Go server is started with `go run -ldflags "-X main.version=9.9.9"
   ./server`. Without it `a.cfg.Version` is `dev`, every host is `unknown`, and
   none of the rollout logic can run under test.
2. Nothing else changes; the throwaway DB and fixed master key stay as they are.

The spec drives a real agent over the real protocol rather than mocking it:
create a host through the UI, take the one-time enroll token from the modal, and
`POST /api/v1/ingest` with a chosen `agent_version` and `auto_update_vetoed`.
That exercises ingest, the ack, and the state machine exactly as a real agent
would. Coverage:

- A push reporting `9.9.9` renders as up to date, with plain text and no pill.
- A push reporting `0.1.0` renders the outdated pill, and the ack carries
  `check_now: true`.
- A push with `auto_update_vetoed: true` renders "disabled on host", and the ack
  carries `check_now: false` even though the version is behind.
- Setting `concurrency` to 1 and pushing from two outdated hosts grants the slot
  to the first only.
- Setting `stall_secs` to 1, waiting, then pushing the old version again renders
  the stalled pill and the paused-rollout banner; `Resume rollout` clears both.
- Setting the per-host policy to Off renders "disabled" and stops the ack from
  asking for a check.

`web/e2e/PLAYWRIGHT-CHECKLIST.md` gains an "Agent updates" section, and the
change ends with the full screenshot sweep that file requires — including the
pages this work does not touch.

There is no eval suite. The repo has no eval harness and this feature makes no
LLM call; it is deterministic code covered by gate tests.

## Out of scope

- No audit table for policy changes or update triggers. The existing audit
  tables are specific to credential reveals and grants, and adding a third is
  wider than this work. Server-side `log.Printf` only.
- No rollback. Downgrading an agent is a release action, not a fleet action.
- Self-registration, which already works.
