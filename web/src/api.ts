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

export type AutoUpdatePolicy = 'default' | 'on' | 'off';

export type UpdateState =
  | 'up_to_date'
  | 'outdated'
  | 'updating'
  | 'stalled'
  | 'disabled'
  | 'unknown';

export interface AgentUpdateRollup {
  server_version: string;
  counts: Partial<Record<UpdateState, number>>;
  paused: boolean;
  stalled: string[];
}

export interface AgentUpdateSettings {
  enabled: boolean;
  concurrency: number;
  stall_secs: number;
}

export interface Settings {
  signup_enabled: boolean;
  retention: Retention;
  smtp: SMTPSettings;
  google: GoogleSettings;
  agent_update: AgentUpdateSettings;
}

// The write shape differs from the read shape: secrets are write-only.
export interface SettingsInput {
  signup_enabled?: boolean;
  retention?: Retention;
  smtp?: SMTPInput;
  google?: GoogleInput;
  agent_update?: AgentUpdateSettings;
}

// Avatar upload rides a multipart form, so it bypasses the JSON request helper.
export async function uploadAvatar(file: File): Promise<User> {
  const form = new FormData();
  form.append('avatar', file);
  const res = await fetch('/api/v1/me/avatar', { method: 'POST', credentials: 'include', body: form });
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
  members: string[];
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

export type ChannelKind = 'generic' | 'webhook';

export type Severity = 'info' | 'warning' | 'error';

export interface Webhook {
  id: string;
  owner_type: 'tool' | 'group' | 'global';
  owner_id: string;
  url: string;
  enabled: boolean;
  format: ChannelKind;
  config: Record<string, string>;
  min_severity: Severity;
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
  tool_id: string;
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
  tool_id: string;
  requester_id: string;
  status: 'pending' | 'approved' | 'denied';
  note: string;
  created_at: string;
}

export interface RevealAuditEntry {
  id: string;
  credential_id: string;
  tool_id: string;
  user_id: string;
  source_ip: string;
  revealed_at: string;
}

export interface GrantAuditEntry {
  id: string;
  tool_id: string;
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

export interface Host {
  id: string;
  name: string;
  os: string;
  physical_location?: string;
  agent_version: string;
  status: 'online' | 'offline' | 'never';
  last_seen_at?: string;
  metrics?: HostMetrics;
  icon_url: string;
  thumbnail_url: string;
  latitude?: number;
  longitude?: number;
  auto_update: AutoUpdatePolicy;
  update_state: UpdateState;
}

export interface InventoryItem {
  source_type: string;
  source_ref: string;
  name: string;
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

// endpointString renders a tool's address as a copyable URL/host:port.
export function endpointString(t: Tool): string {
  if (t.url) {
    return t.url;
  }
  const scheme = t.scheme ? `${t.scheme}://` : '';
  const port = t.port ? `:${t.port}` : '';
  return `${scheme}${t.address}${port}`;
}
