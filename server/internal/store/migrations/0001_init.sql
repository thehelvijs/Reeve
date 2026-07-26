CREATE TABLE users (
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

CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX idx_sessions_user ON sessions(user_id);

CREATE TABLE api_tokens (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    token_hash   TEXT NOT NULL UNIQUE,
    created_at   TEXT NOT NULL,
    last_used_at TEXT,
    revoked_at   TEXT
);
CREATE INDEX idx_api_tokens_user ON api_tokens(user_id);

CREATE TABLE password_resets (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    used_at    TEXT,
    created_at TEXT NOT NULL
);
CREATE INDEX idx_password_resets_user ON password_resets(user_id);

CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE groups (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL
);

CREATE TABLE group_members (
    group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id  TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, user_id)
);
CREATE INDEX idx_group_members_user ON group_members(user_id);

CREATE TABLE hosts (
    id                 TEXT PRIMARY KEY,
    name               TEXT NOT NULL,
    os                 TEXT NOT NULL DEFAULT 'linux',
    physical_location  TEXT NOT NULL DEFAULT '',
    latitude           REAL,
    longitude          REAL,
    enroll_token_hash  TEXT NOT NULL UNIQUE,
    agent_version      TEXT NOT NULL DEFAULT '',
    auto_update        TEXT NOT NULL DEFAULT 'default'
                         CHECK (auto_update IN ('default','on','off')),
    auto_update_vetoed INTEGER NOT NULL DEFAULT 0,
    update_started_at  TEXT,
    last_seen_at       TEXT,
    offline_after_secs INTEGER NOT NULL DEFAULT 60,
    icon_path          TEXT NOT NULL DEFAULT '',
    thumbnail_path     TEXT NOT NULL DEFAULT '',
    created_at         TEXT NOT NULL
);

CREATE TABLE tools (
    id                TEXT PRIMARY KEY,
    name              TEXT NOT NULL,
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
CREATE INDEX idx_tools_creator ON tools(creator_id);

CREATE TABLE collections (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    visibility  TEXT NOT NULL DEFAULT 'public'
                  CHECK (visibility IN ('public', 'restricted')),
    creator_id  TEXT NOT NULL REFERENCES users(id),
    icon_path   TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL
);
CREATE INDEX idx_collections_creator ON collections(creator_id);

CREATE TABLE collection_tools (
    collection_id TEXT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    tool_id       TEXT NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    PRIMARY KEY (collection_id, tool_id)
);
CREATE INDEX idx_collection_tools_tool ON collection_tools(tool_id);

CREATE TABLE collection_editors (
    collection_id  TEXT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    principal_type TEXT NOT NULL CHECK (principal_type IN ('user', 'group')),
    principal_id   TEXT NOT NULL,
    PRIMARY KEY (collection_id, principal_type, principal_id)
);
CREATE INDEX idx_collection_editors_principal
    ON collection_editors(principal_type, principal_id);

CREATE TABLE collection_visibility (
    collection_id  TEXT NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    principal_type TEXT NOT NULL CHECK (principal_type IN ('user', 'group')),
    principal_id   TEXT NOT NULL,
    PRIMARY KEY (collection_id, principal_type, principal_id)
);
CREATE INDEX idx_collection_visibility_principal
    ON collection_visibility(principal_type, principal_id);

CREATE TABLE tool_visibility (
    tool_id        TEXT NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    principal_type TEXT NOT NULL CHECK (principal_type IN ('user', 'group')),
    principal_id   TEXT NOT NULL,
    PRIMARY KEY (tool_id, principal_type, principal_id)
);
CREATE INDEX idx_tool_visibility_principal ON tool_visibility(principal_type, principal_id);

CREATE TABLE credentials (
    id         TEXT PRIMARY KEY,
    tool_id    TEXT NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    type       TEXT NOT NULL CHECK (type IN ('ssh_password', 'ssh_key', 'api_token', 'db', 'kv')),
    label      TEXT NOT NULL DEFAULT '',
    ciphertext BLOB NOT NULL,
    nonce      BLOB NOT NULL,
    created_by TEXT NOT NULL REFERENCES users(id),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX idx_credentials_tool ON credentials(tool_id);

CREATE TABLE credential_access (
    tool_id        TEXT NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    principal_type TEXT NOT NULL CHECK (principal_type IN ('user', 'group')),
    principal_id   TEXT NOT NULL,
    granted_by     TEXT NOT NULL,
    granted_at     TEXT NOT NULL,
    PRIMARY KEY (tool_id, principal_type, principal_id)
);
CREATE INDEX idx_credential_access_principal ON credential_access(principal_type, principal_id);

CREATE TABLE access_requests (
    id           TEXT PRIMARY KEY,
    tool_id      TEXT NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    requester_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status       TEXT NOT NULL DEFAULT 'pending'
                   CHECK (status IN ('pending', 'approved', 'denied')),
    note         TEXT NOT NULL DEFAULT '',
    decided_by   TEXT,
    decided_at   TEXT,
    created_at   TEXT NOT NULL
);
CREATE INDEX idx_access_requests_tool ON access_requests(tool_id);
CREATE INDEX idx_access_requests_requester ON access_requests(requester_id);

CREATE TABLE reveal_audit (
    id            TEXT PRIMARY KEY,
    credential_id TEXT NOT NULL,
    tool_id       TEXT NOT NULL,
    user_id       TEXT NOT NULL,
    source_ip     TEXT NOT NULL DEFAULT '',
    revealed_at   TEXT NOT NULL
);
CREATE INDEX idx_reveal_audit_user ON reveal_audit(user_id);
CREATE INDEX idx_reveal_audit_tool ON reveal_audit(tool_id);

CREATE TABLE grant_audit (
    id             TEXT PRIMARY KEY,
    tool_id        TEXT NOT NULL,
    principal_type TEXT NOT NULL,
    principal_id   TEXT NOT NULL,
    action         TEXT NOT NULL CHECK (action IN ('grant', 'revoke')),
    actor_id       TEXT NOT NULL,
    at             TEXT NOT NULL
);
CREATE INDEX idx_grant_audit_tool ON grant_audit(tool_id);

CREATE TABLE service_status (
    host_id      TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    unit         TEXT NOT NULL,
    active_state TEXT NOT NULL DEFAULT '',
    sub_state    TEXT NOT NULL DEFAULT '',
    updated_at   TEXT NOT NULL,
    PRIMARY KEY (host_id, unit)
);

CREATE TABLE container_status (
    host_id      TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    container_id TEXT NOT NULL,
    name         TEXT NOT NULL DEFAULT '',
    image        TEXT NOT NULL DEFAULT '',
    state        TEXT NOT NULL DEFAULT '',
    health       TEXT NOT NULL DEFAULT '',
    updated_at   TEXT NOT NULL,
    PRIMARY KEY (host_id, container_id)
);

CREATE TABLE cron_jobs (
    host_id     TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    schedule    TEXT NOT NULL DEFAULT '',
    last_run_at TEXT,
    last_exit   INTEGER,
    updated_at  TEXT NOT NULL,
    PRIMARY KEY (host_id, name)
);

CREATE TABLE log_events (
    id      TEXT PRIMARY KEY,
    host_id TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    tool_id TEXT,
    source  TEXT NOT NULL DEFAULT '',
    level   TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL DEFAULT '',
    at      TEXT NOT NULL
);
CREATE INDEX idx_log_events_host ON log_events(host_id, at);

CREATE TABLE metric_samples (
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
CREATE UNIQUE INDEX idx_metric_samples_lookup ON metric_samples(host_id, resolution, ts);

CREATE TABLE container_stats (
    host_id      TEXT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    container_id TEXT NOT NULL,
    ts           TEXT NOT NULL,
    resolution   TEXT NOT NULL CHECK (resolution IN ('raw', '5m', '1h')),
    cpu_pct      REAL NOT NULL DEFAULT 0,
    mem_used     INTEGER NOT NULL DEFAULT 0,
    mem_limit    INTEGER NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX idx_container_stats_lookup ON container_stats(host_id, container_id, resolution, ts);

CREATE TABLE webhooks (
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
CREATE INDEX idx_webhooks_owner ON webhooks(owner_type, owner_id);

CREATE TABLE webhook_deliveries (
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
CREATE INDEX idx_webhook_deliveries_due ON webhook_deliveries(status, next_attempt_at);

CREATE TABLE alert_state (
    subject_key   TEXT PRIMARY KEY,
    state         TEXT NOT NULL CHECK (state IN ('ok', 'pending', 'firing')),
    pending_since TEXT,
    open_event_id TEXT
);

CREATE TABLE alert_events (
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
CREATE INDEX idx_alert_events_subject ON alert_events(subject_key);
CREATE INDEX idx_alert_events_fired ON alert_events(fired_at);

CREATE TABLE alert_thresholds (
    host_id   TEXT NOT NULL DEFAULT '',
    metric    TEXT NOT NULL,
    enabled   INTEGER NOT NULL DEFAULT 1,
    threshold REAL NOT NULL,
    PRIMARY KEY (host_id, metric)
);
INSERT INTO alert_thresholds(host_id, metric, enabled, threshold) VALUES
    ('', 'cpu', 1, 90),
    ('', 'mem', 1, 90),
    ('', 'disk', 1, 90),
    ('', 'temp', 1, 80),
    ('', 'load', 0, 4),
    ('', 'net', 0, 104857600);
