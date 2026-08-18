// Thin fetch wrapper for the Reeve API. Cookies ride along for sessions;
// errors surface as ApiError with the server's code and message.

export class ApiError extends Error {
  code: string;
  status: number;
  // Full parsed body, for errors that carry more than a message (an SSH
  // install returns the remote output alongside the failure).
  body: unknown;
  constructor(status: number, code: string, message: string, body?: unknown) {
    super(message);
    this.code = code;
    this.status = status;
    this.body = body;
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    credentials: 'include',
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
  if (res.status === 204) {
    return undefined as T;
  }
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) {
    const code = data?.code ?? 'error';
    const message = data?.message ?? `request failed (${res.status})`;
    throw new ApiError(res.status, code, message, data);
  }
  return data as T;
}

export const api = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
  patch: <T>(path: string, body?: unknown) => request<T>('PATCH', path, body),
  put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
  del: <T>(path: string, body?: unknown) => request<T>('DELETE', path, body),
};

export interface User {
  id: string;
  email: string;
  role: 'admin' | 'basic';
  display_name: string;
  avatar_url: string;
}

export interface AdminUser extends User {
  active: boolean;
}

export interface Retention {
  raw_secs: number;
  fivemin_secs: number;
  onehour_secs: number;
}

export interface SMTPSettings {
  enabled: boolean;
  host: string;
  port: number;
  username: string;
  from: string;
  tls: 'starttls' | 'tls' | 'none';
  password_set: boolean;
}

export interface SMTPInput extends Omit<SMTPSettings, 'password_set'> {
  password: string;
}

export interface SSHTarget {
  address: string;
  port: number;
  username: string;
  password: string;
  private_key: string;
  passphrase: string;
  sudo_password: string;
  fingerprint: string;
}

export interface GoogleSettings {
  enabled: boolean;
  client_id: string;
  allowed_domains: string;
  secret_set: boolean;
  redirect_url: string;
}

export interface GoogleInput extends Omit<GoogleSettings, 'secret_set' | 'redirect_url'> {
  client_secret: string;
}

// One connected forge. GitLab is gitlab.com or any instance you run; GitHub is
// github.com or an Enterprise Server. Only the URL tells them apart, and an
// empty URL means the hosted one.
export interface ForgeSettings {
  url: string;
  token_set: boolean;
}

export interface ForgeInput extends Omit<ForgeSettings, 'token_set'> {
  token: string;
}

// Which forge a pipeline group's paths belong to.
export type Provider = 'gitlab' | 'github';

// The state of a repo's latest run, lowercased. Both forges are reported in
// GitLab's vocabulary, so one status has one meaning. Empty means never run.
export type PipelineStatus = string;

export interface PipelineProject {
  name: string;
  path: string;
  url: string;
  status: PipelineStatus;
  ref?: string;
  updated_at?: string;
  pipeline_url?: string;
  // This project alone could not be read; the rest of its group still renders.
  error?: string;
}

// A group as the pipelines page sees it: the operator's name with each member's
// latest pipeline attached.
export interface PipelineGroup {
  id: string;
  name: string;
  provider: Provider;
  projects: PipelineProject[];
  // This group's own failure; the other groups still rendered.
  error?: string;
}

// A group as it is stored: the membership an admin edits, with no call to
// GitLab behind it.
export interface PipelineGroupRecord {
  id: string;
  name: string;
  provider: Provider;
  projects: string[];
}

export interface PipelineOverview {
  configured: boolean;
  groups: PipelineGroup[];
}

export interface StalledHost {
  id: string;
  name: string;
}

export interface AgentUpdateRollup {
  server_version: string;
  counts: Record<UpdateState, number>;
  paused: boolean;
  stalled: StalledHost[];
}

export interface AgentUpdateSettings {
  enabled: boolean;
  concurrency: number;
  stall_secs: number;
}

// Which stream of builds this server follows. The tag each one resolves to is
// the server's business, not the browser's.
export type UpdateChannel = 'release' | 'develop' | 'main';

export interface ServerUpdateSettings {
  channel: UpdateChannel;
  // Read-only: the build running now, and when it replaced a different one.
  version?: string;
  updated_at?: string;
  // Read-only: whether an updater watches this deployment. False means the
  // channel is recorded and nothing acts on it.
  managed?: boolean;
}

export interface Settings {
  signup_enabled: boolean;
  heartbeat_url: string;
  retention: Retention;
  smtp: SMTPSettings;
  google: GoogleSettings;
  gitlab: ForgeSettings;
  github: ForgeSettings;
  agent_update: AgentUpdateSettings;
  server_update: ServerUpdateSettings;
}

// The write shape differs from the read shape: secrets are write-only.
export interface SettingsInput {
  signup_enabled?: boolean;
  heartbeat_url?: string;
  retention?: Retention;
  smtp?: SMTPInput;
  google?: GoogleInput;
  gitlab?: ForgeInput;
  github?: ForgeInput;
  agent_update?: AgentUpdateSettings;
  server_update?: ServerUpdateSettings;
}

// Avatar upload rides a multipart form, so it bypasses the JSON request helper.
export async function uploadAvatar(file: File): Promise<User> {
  const form = new FormData();
  form.append('avatar', file);
  const res = await fetch('/api/me/avatar', { method: 'POST', credentials: 'include', body: form });
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) {
    throw new ApiError(res.status, data?.code ?? 'error', data?.message ?? 'upload failed');
  }
  return data as User;
}

// uploadIcon posts an entity icon (tools/hosts) and returns the new icon URL.
export async function uploadIcon(path: string, file: File): Promise<string> {
  const form = new FormData();
  form.append('icon', file);
  const res = await fetch(path, { method: 'POST', credentials: 'include', body: form });
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) {
    throw new ApiError(res.status, data?.code ?? 'error', data?.message ?? 'upload failed');
  }
  return (data?.icon_url as string) ?? '';
}

export interface Group {
  id: string;
  name: string;
  members: GroupMember[];
}

// A membership carries the person's name, so a moderator can read their own group
// without being handed the instance's user list.
export interface GroupMember {
  user_id: string;
  email: string;
  display_name: string;
  avatar_url: string;
  role: 'moderator' | 'member';
}

export interface Invite {
  user: AdminUser;
  emailed: boolean;
  invite_link?: string;
}

// The compact collection shape embedded in a tool payload.
export interface CollectionRef {
  id: string;
  name: string;
  icon_url: string;
}

export interface Collection {
  id: string;
  name: string;
  description: string;
  visibility: 'public' | 'restricted';
  creator_id: string;
  icon_url: string;
  tool_count: number;
  can_edit: boolean;
  created_at: string;
}

export interface CollectionDetail extends Collection {
  tool_ids: string[];
}

export interface Principals {
  users: { id: string; display_name: string; email: string }[];
  groups: { id: string; name: string }[];
}

export type ToolStatus = 'up' | 'down' | 'agent_offline' | 'unknown';

export interface Tool {
  id: string;
  name: string;
  // The name in /go/<slug>, the link that survives a host address change.
  slug: string;
  description: string;
  collections: CollectionRef[];
  tags: string[];
  scheme: string;
  address: string;
  port?: number;
  url?: string;
  physical_location?: string;
  host_id?: string;
  source_type: string;
  source_ref: string;
  status: ToolStatus;
  visibility: 'public' | 'restricted';
  creator_id: string;
  can_edit: boolean;
  log_alert_enabled: boolean;
  icon_url: string;
  thumbnail_url: string;
}

// The receiver a channel posts to. 'auto' reads it off the URL, 'generic' sends
// Reeve's own JSON, 'custom' renders the channel's template. The server owns this
// vocabulary; /api/admin/webhook-formats is the list the form offers.
export type ChannelKind =
  | 'auto'
  | 'discord'
  | 'slack'
  | 'mattermost'
  | 'rocketchat'
  | 'googlechat'
  | 'teams'
  | 'teamsflow'
  | 'webex'
  | 'ntfy'
  | 'gotify'
  | 'telegram'
  | 'pagerduty'
  | 'generic'
  | 'custom';

export interface ChannelFormats {
  formats: ChannelKind[];
  events: string[];
  variables: string[];
}

export type Severity = 'info' | 'warning' | 'error';

// What a channel watches. Global receives everything; the rest add to it.
export type ChannelScope = 'global' | 'host' | 'tool' | 'group';

export interface Webhook {
  id: string;
  owner_type: ChannelScope;
  owner_id: string;
  url: string;
  enabled: boolean;
  format: ChannelKind;
  detected: ChannelKind;
  config: Record<string, string>;
  min_severity: Severity;
  // The event types this channel wants. Empty is all of them.
  events: string[];
}

export interface AlertEvent {
  id: string;
  tool_id?: string;
  host_id?: string;
  type:
    | 'down'
    | 'agent_offline'
    | 'log_error'
    | `${ThresholdMetric}_high`;
  severity: string;
  message: string;
  fired_at: string;
  resolved_at?: string;
}

export type ThresholdMetric = 'cpu' | 'mem' | 'disk' | 'temp' | 'load' | 'net';

export interface ThresholdSetting {
  enabled: boolean;
  threshold: number;
}

export interface ThresholdsPayload {
  window_secs?: number;
  thresholds: Partial<Record<ThresholdMetric, ThresholdSetting>>;
}

export interface Delivery {
  id: string;
  webhook_id: string;
  status: 'pending' | 'sent' | 'failed' | 'dead';
  attempts: number;
  last_error: string;
}

export interface VisibilityGrant {
  principal_type: 'user' | 'group';
  principal_id: string;
}

export type CredentialType = 'ssh_password' | 'ssh_key' | 'api_token' | 'db' | 'kv';

export interface Credential {
  id: string;
  host_id: string;
  type: CredentialType;
  label: string;
  can_reveal: boolean;
}

export interface RevealedCredential {
  id: string;
  type: CredentialType;
  label: string;
  secret: Record<string, string>;
}

export interface AccessRequest {
  id: string;
  host_id: string;
  host_name: string;
  requester_id: string;
  requester_name: string;
  status: 'pending' | 'approved' | 'denied';
  note: string;
  created_at: string;
}

export interface RevealAuditEntry {
  id: string;
  credential_id: string;
  host_id: string;
  user_id: string;
  source_ip: string;
  revealed_at: string;
}

export interface GrantAuditEntry {
  id: string;
  host_id: string;
  principal_type: string;
  principal_id: string;
  action: 'grant' | 'revoke';
  actor_id: string;
  at: string;
}

export interface HostMetrics {
  cpu_pct: number;
  mem_used: number;
  mem_total: number;
  disk_used: number;
  disk_total: number;
  at: string;
}

export type AutoUpdatePolicy = 'default' | 'on' | 'off';

export type UpdateState =
  | 'up_to_date'
  | 'outdated'
  | 'updating'
  | 'stalled'
  | 'disabled'
  | 'unknown';

export interface Host {
  id: string;
  name: string;
  // What this machine is for, in an operator's words.
  description?: string;
  os: string;
  physical_location?: string;
  // Where the agent last reported this host to be. Empty until it first pushes.
  ip_address: string;
  agent_version: string;
  status: 'online' | 'offline' | 'never';
  last_seen_at?: string;
  metrics?: HostMetrics;
  icon_url: string;
  thumbnail_url: string;
  latitude?: number;
  longitude?: number;
  // #rrggbb for this host's map pin, absent for the brand accent.
  pin_color?: string;
  auto_update: AutoUpdatePolicy;
  // The machine's own refusal (REEVE_AUTO_UPDATE=false). No server-side policy
  // or operator override outranks it: the agent ignores the ack.
  auto_update_vetoed: boolean;
  update_state: UpdateState;
  control_enabled: boolean;
  // The machine Reeve itself runs on. It reports itself instead of through an
  // agent, so nothing agent-shaped is offered on it.
  is_server?: boolean;
}

export type CommandStatus = 'pending' | 'sent' | 'done' | 'failed' | 'expired';

export interface HostCommand {
  id: string;
  host_id: string;
  action: string;
  target: string;
  status: CommandStatus;
  output: string;
  requested_by: string;
  requested_by_name: string;
  requested_at: string;
  finished_at?: string;
}

export interface InventoryItem {
  source_type: string;
  source_ref: string;
  name: string;
  // What the machine says this is doing now — a systemd active state or a
  // container state. Empty for a cron job, which has no running state.
  state: string;
  detail: string;
  linked: boolean;
}

export interface HostInventory {
  services: InventoryItem[];
  containers: InventoryItem[];
  cron_jobs: InventoryItem[];
}

export interface UptimeSummary {
  range: string;
  uptime_pct: number;
}

// Field templates per credential type; kv is free-form (handled separately).
export const CREDENTIAL_FIELDS: Record<CredentialType, string[]> = {
  ssh_password: ['username', 'password'],
  ssh_key: ['username', 'private_key'],
  api_token: ['token'],
  db: ['host', 'port', 'username', 'password', 'database'],
  kv: [],
};

// What each type is called on screen. `ssh_password` and `kv` are storage keys;
// both the picker and the stored list used to print them raw.
export const CREDENTIAL_LABEL: Record<CredentialType, string> = {
  ssh_password: 'SSH password',
  ssh_key: 'SSH key',
  api_token: 'API token',
  db: 'Database login',
  kv: 'Key/value pairs',
};

// endpointString renders a tool's address as a copyable URL/host:port. A tool
// with no address of its own follows its host, and only the server knows where
// that host currently is, so this renders a placeholder rather than a wrong
// answer; goURL is what to link to.
export function endpointString(t: Tool): string {
  if (t.url) {
    return t.url;
  }
  if (!t.address) {
    return t.host_id ? 'follows this host' : '';
  }
  const scheme = t.scheme ? `${t.scheme}://` : '';
  const port = t.port ? `:${t.port}` : '';
  return `${scheme}${t.address}${port}`;
}

// goURL is the durable link to a tool: the server resolves it at click time,
// so it keeps working after the host's address changes.
export function goURL(t: Pick<Tool, 'slug'>): string {
  return `/go/${t.slug}`;
}
