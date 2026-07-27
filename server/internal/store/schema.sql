CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('admin', 'basic')),
    active        INTEGER NOT NULL DEFAULT 1,
    display_name  TEXT NOT NULL DEFAULT '',
    avatar_path   TEXT NOT NULL DEFAULT '',
    -- The identity provider's immutable subject id, pinned on the first
    -- federated sign-in. An email address can be reassigned by whoever runs
    -- the domain; this cannot, so later sign-ins are matched against it.
    oauth_subject TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL
);

-- token_hash is the sha256 of the session cookie, never the cookie itself: the
-- cookie is a bearer token, so a readable database would otherwise be enough to
-- sign in as anyone holding a live session.
CREATE TABLE IF NOT EXISTS sessions (
    token_hash TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);

CREATE TABLE IF NOT EXISTS password_resets (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    used_at    TEXT,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_password_resets_user ON password_resets(user_id);

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS groups (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS group_members (
    group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id  TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_group_members_user ON group_members(user_id);

CREATE TABLE IF NOT EXISTS hosts (
    id                 TEXT PRIMARY KEY,
    name               TEXT NOT NULL,
    os                 TEXT NOT NULL DEFAULT 'linux',
    physical_location  TEXT NOT NULL DEFAULT '',
    latitude           REAL,
    longitude          REAL,
    enroll_token_hash  TEXT NOT NULL UNIQUE,
    -- The address the agent last reported, on the route from this host to the
    -- server. Tools on this host with no address of their own follow it.
    ip_address         TEXT NOT NULL DEFAULT '',
    agent_version      TEXT NOT NULL DEFAULT '',
    -- sha256 of the binary the agent last reported running. What "up to date"
    -- is decided against, since a version string does not identify a build.
    agent_checksum     TEXT NOT NULL DEFAULT '',
    auto_update        TEXT NOT NULL DEFAULT 'default'
                         CHECK (auto_update IN ('default','on','off')),
    auto_update_vetoed INTEGER NOT NULL DEFAULT 0,
    update_started_at  TEXT,
    -- Whether the slot above was granted by an operator pressing Update rather
    -- than by the paced rollout. The two are treated differently: a paced slot
    -- on a host whose policy has since turned off is dangling and must not
    -- count, while a forced one is exactly what the operator asked for.
    update_forced      INTEGER NOT NULL DEFAULT 0,
    last_seen_at       TEXT,
    offline_after_secs INTEGER NOT NULL DEFAULT 60,
    -- Whether the agent last said it will run commands. Never assumed: an
    -- agent that has not said so is never sent one.
    control_enabled    INTEGER NOT NULL DEFAULT 0,
    icon_path          TEXT NOT NULL DEFAULT '',
    thumbnail_path     TEXT NOT NULL DEFAULT '',
    created_at         TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tools (
    id                TEXT PRIMARY KEY,
    name              TEXT NOT NULL,
    -- The name in /go/<slug>. Unique so one short URL means one tool.
    slug              TEXT NOT NULL UNIQUE,
    description       TEXT NOT NULL DEFAULT '',
    tags              TEXT NOT NULL DEFAULT '[]',
    scheme            TEXT NOT NULL DEFAULT '',
    address           TEXT NOT NULL DEFAULT '',
    port              INTEGER NOT NULL DEFAULT 0,
    url               TEXT NOT NULL DEFAULT '',
    physical_location TEXT NOT NULL DEFAULT '',
    host_id           TEXT,
    source_type       TEXT NOT NULL DEFAULT 'manual'
                        CHECK (source_type IN ('manual', 'systemd', 'docker', 'cron')),
    source_ref        TEXT NOT NULL DEFAULT '',
    visibility        TEXT NOT NULL DEFAULT 'public'
                        CHECK (visibility IN ('public', 'restricted')),
    creator_id        TEXT NOT NULL REFERENCES users(id),
    log_alert_enabled INTEGER NOT NULL DEFAULT 0,
    icon_path         TEXT NOT NULL DEFAULT '',
    thumbnail_path    TEXT NOT NULL DEFAULT '',
    created_at        TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_tools_creator ON tools(creator_id);

CREATE TABLE IF NOT EXISTS collections (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    visibility  TEXT NOT NULL DEFAULT 'public'
                  CHECK (visibility IN ('public', 'restricted')),
    creator_id  TEXT NOT NULL REFERENCES users(id),
    icon_path   TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_collections_creator ON collections(creator_id);

CREATE TABLE IF NOT EXISTS collection_tools (
    collection_id TEXT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    tool_id       TEXT NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    PRIMARY KEY (collection_id, tool_id)
);
CREATE INDEX IF NOT EXISTS idx_collection_tools_tool ON collection_tools(tool_id);

CREATE TABLE IF NOT EXISTS collection_editors (
    collection_id  TEXT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    principal_type TEXT NOT NULL CHECK (principal_type IN ('user', 'group')),
    principal_id   TEXT NOT NULL,
    PRIMARY KEY (collection_id, principal_type, principal_id)
);
CREATE INDEX IF NOT EXISTS idx_collection_editors_principal
    ON collection_editors(principal_type, principal_id);

CREATE TABLE IF NOT EXISTS collection_visibility (
    collection_id  TEXT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    principal_type TEXT NOT NULL CHECK (principal_type IN ('user', 'group')),
    principal_id   TEXT NOT NULL,
    PRIMARY KEY (collection_id, principal_type, principal_id)
);
CREATE INDEX IF NOT EXISTS idx_collection_visibility_principal
    ON collection_visibility(principal_type, principal_id);

CREATE TABLE IF NOT EXISTS tool_visibility (
    tool_id        TEXT NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    principal_type TEXT NOT NULL CHECK (principal_type IN ('user', 'group')),
    principal_id   TEXT NOT NULL,
    PRIMARY KEY (tool_id, principal_type, principal_id)
);
CREATE INDEX IF NOT EXISTS idx_tool_visibility_principal ON tool_visibility(principal_type, principal_id);

CREATE TABLE IF NOT EXISTS credentials (
    id         TEXT PRIMARY KEY,
    host_id    TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    type       TEXT NOT NULL CHECK (type IN ('ssh_password', 'ssh_key', 'api_token', 'db', 'kv')),
    label      TEXT NOT NULL DEFAULT '',
    ciphertext BLOB NOT NULL,
    nonce      BLOB NOT NULL,
    created_by TEXT NOT NULL REFERENCES users(id),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_credentials_host ON credentials(host_id);

CREATE TABLE IF NOT EXISTS credential_access (
    host_id        TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    principal_type TEXT NOT NULL CHECK (principal_type IN ('user', 'group')),
    principal_id   TEXT NOT NULL,
    granted_by     TEXT NOT NULL,
    granted_at     TEXT NOT NULL,
    PRIMARY KEY (host_id, principal_type, principal_id)
);
CREATE INDEX IF NOT EXISTS idx_credential_access_principal ON credential_access(principal_type, principal_id);

CREATE TABLE IF NOT EXISTS access_requests (
    id           TEXT PRIMARY KEY,
    host_id      TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    requester_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status       TEXT NOT NULL DEFAULT 'pending'
                   CHECK (status IN ('pending', 'approved', 'denied')),
    note         TEXT NOT NULL DEFAULT '',
    decided_by   TEXT,
    decided_at   TEXT,
    created_at   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_access_requests_host ON access_requests(host_id);
CREATE INDEX IF NOT EXISTS idx_access_requests_requester ON access_requests(requester_id);

CREATE TABLE IF NOT EXISTS reveal_audit (
    id            TEXT PRIMARY KEY,
    credential_id TEXT NOT NULL,
    host_id       TEXT NOT NULL,
    user_id       TEXT NOT NULL,
    source_ip     TEXT NOT NULL DEFAULT '',
    revealed_at   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_reveal_audit_user ON reveal_audit(user_id);
CREATE INDEX IF NOT EXISTS idx_reveal_audit_host ON reveal_audit(host_id);

CREATE TABLE IF NOT EXISTS grant_audit (
    id             TEXT PRIMARY KEY,
    host_id        TEXT NOT NULL,
    principal_type TEXT NOT NULL,
    principal_id   TEXT NOT NULL,
    action         TEXT NOT NULL CHECK (action IN ('grant', 'revoke')),
    actor_id       TEXT NOT NULL,
    at             TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_grant_audit_host ON grant_audit(host_id);

CREATE TABLE IF NOT EXISTS service_status (
    host_id      TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    unit         TEXT NOT NULL,
    active_state TEXT NOT NULL DEFAULT '',
    sub_state    TEXT NOT NULL DEFAULT '',
    updated_at   TEXT NOT NULL,
    PRIMARY KEY (host_id, unit)
);

CREATE TABLE IF NOT EXISTS container_status (
    host_id      TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    container_id TEXT NOT NULL,
    name         TEXT NOT NULL DEFAULT '',
    image        TEXT NOT NULL DEFAULT '',
    state        TEXT NOT NULL DEFAULT '',
    health       TEXT NOT NULL DEFAULT '',
    updated_at   TEXT NOT NULL,
    PRIMARY KEY (host_id, container_id)
);

CREATE TABLE IF NOT EXISTS cron_jobs (
    host_id     TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    schedule    TEXT NOT NULL DEFAULT '',
    last_run_at TEXT,
    last_exit   INTEGER,
    updated_at  TEXT NOT NULL,
    PRIMARY KEY (host_id, name)
);

-- Remote actions queued for an agent, collected on its next push. requested_by
-- has no foreign key on purpose: deleting the account must not rewrite who
-- rebooted a machine, the same reason reveal_audit keeps a bare user id.
CREATE TABLE IF NOT EXISTS host_commands (
    id           TEXT PRIMARY KEY,
    host_id      TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    action       TEXT NOT NULL,
    target       TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'pending'
                   CHECK (status IN ('pending','sent','done','failed','expired')),
    output       TEXT NOT NULL DEFAULT '',
    requested_by TEXT NOT NULL,
    requested_at TEXT NOT NULL,
    sent_at      TEXT,
    finished_at  TEXT
);
CREATE INDEX IF NOT EXISTS idx_host_commands_host ON host_commands(host_id, status);

CREATE TABLE IF NOT EXISTS log_events (
    id      TEXT PRIMARY KEY,
    host_id TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    tool_id TEXT,
    source  TEXT NOT NULL DEFAULT '',
    level   TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL DEFAULT '',
    at      TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_log_events_host ON log_events(host_id, at);

CREATE TABLE IF NOT EXISTS metric_samples (
    host_id       TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    ts            TEXT NOT NULL,
    resolution    TEXT NOT NULL CHECK (resolution IN ('raw', '5m', '1h')),
    cpu_pct       REAL NOT NULL DEFAULT 0,
    mem_used      INTEGER NOT NULL DEFAULT 0,
    mem_total     INTEGER NOT NULL DEFAULT 0,
    disk_used     INTEGER NOT NULL DEFAULT 0,
    disk_total    INTEGER NOT NULL DEFAULT 0,
    disk_read     INTEGER NOT NULL DEFAULT 0,
    disk_write    INTEGER NOT NULL DEFAULT 0,
    net_rx        INTEGER NOT NULL DEFAULT 0,
    net_tx        INTEGER NOT NULL DEFAULT 0,
    uptime_secs   INTEGER NOT NULL DEFAULT 0,
    temps         TEXT NOT NULL DEFAULT '{}',
    load1         REAL NOT NULL DEFAULT 0,
    load5         REAL NOT NULL DEFAULT 0,
    load15        REAL NOT NULL DEFAULT 0,
    gpu_util      REAL NOT NULL DEFAULT 0,
    gpu_mem_used  INTEGER NOT NULL DEFAULT 0,
    gpu_mem_total INTEGER NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_metric_samples_lookup ON metric_samples(host_id, resolution, ts);

-- One replaceable row per host: the latest process snapshot, not a time series.
CREATE TABLE IF NOT EXISTS host_processes (
    host_id TEXT PRIMARY KEY REFERENCES hosts(id) ON DELETE CASCADE,
    ts      TEXT NOT NULL,
    procs   TEXT NOT NULL DEFAULT '[]'
);

-- One replaceable row per host: every mounted filesystem's capacity as of the
-- last push. A snapshot, not a series — the root filesystem is already charted
-- in metric_samples, and what this answers is "which of this machine's disks is
-- filling up", which only needs the current answer.
CREATE TABLE IF NOT EXISTS host_disks (
    host_id TEXT PRIMARY KEY REFERENCES hosts(id) ON DELETE CASCADE,
    ts      TEXT NOT NULL,
    disks   TEXT NOT NULL DEFAULT '[]'
);

-- Per-command usage accumulated into 5-minute buckets, so an average over any
-- window is SUM(cpu_sum)/SUM(samples) and no per-push row has to be kept. Keyed
-- by command, not pid: a process that restarts is the same thing to whoever is
-- hunting for what a machine spends its day on.
CREATE TABLE IF NOT EXISTS process_usage (
    host_id  TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    bucket   TEXT NOT NULL,
    command  TEXT NOT NULL,
    samples  INTEGER NOT NULL DEFAULT 0,
    cpu_sum  REAL NOT NULL DEFAULT 0,
    cpu_max  REAL NOT NULL DEFAULT 0,
    mem_sum  REAL NOT NULL DEFAULT 0,
    mem_max  INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (host_id, bucket, command)
);
CREATE INDEX IF NOT EXISTS idx_process_usage_window ON process_usage(host_id, bucket);

CREATE TABLE IF NOT EXISTS container_stats (
    host_id      TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    container_id TEXT NOT NULL,
    ts           TEXT NOT NULL,
    resolution   TEXT NOT NULL CHECK (resolution IN ('raw', '5m', '1h')),
    cpu_pct      REAL NOT NULL DEFAULT 0,
    mem_used     INTEGER NOT NULL DEFAULT 0,
    mem_limit    INTEGER NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_container_stats_lookup ON container_stats(host_id, container_id, resolution, ts);

CREATE TABLE IF NOT EXISTS webhooks (
    id           TEXT PRIMARY KEY,
    owner_type   TEXT NOT NULL CHECK (owner_type IN ('tool', 'group', 'global')),
    owner_id     TEXT NOT NULL DEFAULT '',
    url          TEXT NOT NULL,
    format       TEXT NOT NULL DEFAULT 'generic',
    config       TEXT NOT NULL DEFAULT '{}',
    min_severity TEXT NOT NULL DEFAULT 'info',
    enabled      INTEGER NOT NULL DEFAULT 1,
    created_at   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_webhooks_owner ON webhooks(owner_type, owner_id);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id              TEXT PRIMARY KEY,
    webhook_id      TEXT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    payload         TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending', 'sent', 'failed', 'dead')),
    attempts        INTEGER NOT NULL DEFAULT 0,
    last_error      TEXT NOT NULL DEFAULT '',
    next_attempt_at TEXT NOT NULL,
    created_at      TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_due ON webhook_deliveries(status, next_attempt_at);

CREATE TABLE IF NOT EXISTS alert_state (
    subject_key   TEXT PRIMARY KEY,
    state         TEXT NOT NULL CHECK (state IN ('ok', 'pending', 'firing')),
    pending_since TEXT,
    open_event_id TEXT
);

CREATE TABLE IF NOT EXISTS alert_events (
    id          TEXT PRIMARY KEY,
    subject_key TEXT NOT NULL,
    tool_id     TEXT,
    host_id     TEXT,
    type        TEXT NOT NULL CHECK (type IN ('down', 'agent_offline', 'log_error',
                  'cpu_high', 'mem_high', 'disk_high', 'temp_high', 'load_high', 'net_high')),
    severity    TEXT NOT NULL DEFAULT 'error',
    message     TEXT NOT NULL DEFAULT '',
    fired_at    TEXT NOT NULL,
    resolved_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_alert_events_subject ON alert_events(subject_key);
CREATE INDEX IF NOT EXISTS idx_alert_events_fired ON alert_events(fired_at);

CREATE TABLE IF NOT EXISTS alert_thresholds (
    host_id   TEXT NOT NULL DEFAULT '',
    metric    TEXT NOT NULL,
    enabled   INTEGER NOT NULL DEFAULT 1,
    threshold REAL NOT NULL,
    PRIMARY KEY (host_id, metric)
);
INSERT OR IGNORE INTO alert_thresholds(host_id, metric, enabled, threshold) VALUES
    ('', 'cpu', 1, 90),
    ('', 'mem', 1, 90),
    ('', 'disk', 1, 90),
    ('', 'temp', 1, 80),
    ('', 'load', 0, 4),
    ('', 'net', 0, 104857600);
