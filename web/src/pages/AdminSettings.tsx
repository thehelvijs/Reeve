import { useEffect, useState } from 'react';
import { api, type GoogleInput, type Settings, type SettingsInput, type SMTPInput } from '../api';
import { Button, Card, ErrorText, Field, Form, Input } from '../components/ui';
import PageHeader from '../components/PageHeader';

// Retention is stored in seconds but only ever reasoned about in hours or days.
const RETENTION_CHOICES: { label: string; secs: number }[] = [
  { label: '6 hours', secs: 6 * 3600 },
  { label: '24 hours', secs: 24 * 3600 },
  { label: '48 hours', secs: 48 * 3600 },
  { label: '7 days', secs: 7 * 24 * 3600 },
  { label: '30 days', secs: 30 * 24 * 3600 },
  { label: '90 days', secs: 90 * 24 * 3600 },
  { label: '1 year', secs: 365 * 24 * 3600 },
];

export default function AdminSettings() {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    api
      .get<Settings>('/api/admin/settings')
      .then(setSettings)
      .catch((e) => setError(e instanceof Error ? e.message : 'could not load settings'));
  }, []);

  const save = async (patch: SettingsInput) => {
    setError('');
    try {
      setSettings(await api.put<Settings>('/api/admin/settings', patch));
    } catch (e) {
      setError(e instanceof Error ? e.message : 'could not save settings');
    }
  };

  if (!settings) {
    return (
      <div>
        <PageHeader title="Settings" subtitle="Instance-wide configuration." />
        <div className="mt-6">{error ? <ErrorText>{error}</ErrorText> : <p className="text-sm text-muted">Loading…</p>}</div>
      </div>
    );
  }

  return (
    <div>
      <PageHeader title="Settings" subtitle="Instance-wide configuration." />
      <div className="mt-3">
        <ErrorText>{error}</ErrorText>
      </div>

      <Section
        title="Sign-up"
        description="With sign-up open, anyone who can reach this server can create an account and see public services."
      >
        <div className="flex items-center justify-between gap-4">
          <div>
            <p className="text-sm text-content">{settings.signup_enabled ? 'Open' : 'Closed'}</p>
            <p className="text-xs text-muted">
              {settings.signup_enabled
                ? 'New people can register themselves as basic users.'
                : 'Only an admin can add accounts.'}
            </p>
          </div>
          <Button
            variant={settings.signup_enabled ? 'danger' : 'primary'}
            onClick={() => save({ signup_enabled: !settings.signup_enabled })}
          >
            {settings.signup_enabled ? 'Close sign-up' : 'Open sign-up'}
          </Button>
        </div>
      </Section>

      <EmailSection settings={settings} onSave={save} />
      <GoogleSection settings={settings} onSave={save} />
      <RetentionSection settings={settings} onSave={save} />
      <AgentUpdateSection settings={settings} onSave={save} />
    </div>
  );
}

function RetentionSection({
  settings,
  onSave,
}: {
  settings: Settings;
  onSave: (patch: SettingsInput) => Promise<void>;
}) {
  const [draft, setDraft] = useState(settings.retention);
  useEffect(() => setDraft(settings.retention), [settings.retention]);

  const dirty =
    draft.raw_secs !== settings.retention.raw_secs ||
    draft.fivemin_secs !== settings.retention.fivemin_secs ||
    draft.onehour_secs !== settings.retention.onehour_secs;

  return (
    <Section
      title="Metric retention"
      description="How long each resolution tier is kept before the rollup job prunes it. Coarser tiers must be kept at least as long as finer ones."
    >
      <Form onSubmit={() => onSave({ retention: draft })}>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
          <Field label="Raw samples">
            <RetentionSelect
              value={draft.raw_secs}
              onChange={(raw_secs) => setDraft({ ...draft, raw_secs })}
            />
          </Field>
          <Field label="5-minute rollups">
            <RetentionSelect
              value={draft.fivemin_secs}
              onChange={(fivemin_secs) => setDraft({ ...draft, fivemin_secs })}
            />
          </Field>
          <Field label="1-hour rollups">
            <RetentionSelect
              value={draft.onehour_secs}
              onChange={(onehour_secs) => setDraft({ ...draft, onehour_secs })}
            />
          </Field>
        </div>
        <div className="mt-4 flex justify-end">
          <Button type="submit" disabled={!dirty}>
            Save retention
          </Button>
        </div>
      </Form>
    </Section>
  );
}

// Sanitizes a possibly-NaN draft value for display; NaN means the operator cleared the field mid-edit.
function numOrEmpty(n: number): number | string {
  if (Number.isNaN(n)) {
    return '';
  }
  return n;
}

function AgentUpdateSection({
  settings,
  onSave,
}: {
  settings: Settings;
  onSave: (patch: SettingsInput) => Promise<void>;
}) {
  const [draft, setDraft] = useState(settings.agent_update);
  useEffect(() => setDraft(settings.agent_update), [settings.agent_update]);

  const dirty =
    draft.enabled !== settings.agent_update.enabled ||
    draft.concurrency !== settings.agent_update.concurrency ||
    draft.stall_secs !== settings.agent_update.stall_secs;

  const concurrencyValid = !Number.isNaN(draft.concurrency) && draft.concurrency >= 1 && draft.concurrency <= 100;
  const stallValid = !Number.isNaN(draft.stall_secs) && draft.stall_secs >= 1 && draft.stall_secs <= 86400;

  return (
    <Section
      title="Agent updates"
      description="The fleet default for agent self-updates. A host can override it, and a host running the agent with REEVE_AUTO_UPDATE=false always refuses."
    >
      <Form onSubmit={() => onSave({ agent_update: draft })}>
        <label className="flex items-center gap-2 text-sm text-content">
          <input
            type="checkbox"
            className="h-4 w-4 accent-accent"
            checked={draft.enabled}
            onChange={(e) => setDraft({ ...draft, enabled: e.target.checked })}
          />
          Roll out agent updates automatically
        </label>
        <p className="mt-1 text-xs text-muted">
          {draft.enabled
            ? 'Outdated hosts are told to update, a few at a time.'
            : 'No host updates unless it is pinned on for that host, or you press Update now.'}
        </p>

        <div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Hosts updating at once" hint="1–100. The first batch is effectively a canary.">
            <Input
              type="number"
              min={1}
              max={100}
              value={numOrEmpty(draft.concurrency)}
              onChange={(e) => setDraft({ ...draft, concurrency: e.target.valueAsNumber })}
            />
          </Field>
          <Field
            label="Stall timeout (seconds)"
            hint="A host that doesn't report the new version within this window stalls, and halts the whole rollout until an admin resumes it from the Hosts page."
          >
            <Input
              type="number"
              min={1}
              max={86400}
              value={numOrEmpty(draft.stall_secs)}
              onChange={(e) => setDraft({ ...draft, stall_secs: e.target.valueAsNumber })}
            />
          </Field>
        </div>

        <div className="mt-4 flex justify-end">
          <Button type="submit" disabled={!dirty || !concurrencyValid || !stallValid}>
            Save agent updates
          </Button>
        </div>
      </Form>
    </Section>
  );
}

// The read view carries password_set, which the server's strict JSON decoder
// rejects on the way back in, so the draft is built field by field.
function smtpDraft(s: Settings['smtp']): SMTPInput {
  return {
    enabled: s.enabled,
    host: s.host,
    port: s.port,
    username: s.username,
    from: s.from,
    tls: s.tls,
    password: '',
  };
}

function EmailSection({ settings, onSave }: { settings: Settings; onSave: (patch: SettingsInput) => Promise<void> }) {
  const [draft, setDraft] = useState<SMTPInput>(smtpDraft(settings.smtp));
  const [testing, setTesting] = useState('');
  useEffect(() => setDraft(smtpDraft(settings.smtp)), [settings.smtp]);

  const sendTest = async () => {
    setTesting('sending');
    try {
      const r = await api.post<{ sent_to: string }>('/api/admin/settings/test-email');
      setTesting(`sent to ${r.sent_to}`);
    } catch (e) {
      setTesting(e instanceof Error ? e.message : 'send failed');
    }
  };

  return (
    <Section
      title="Email"
      description="An SMTP relay lets people reset their own password. Without one, only an admin can set passwords."
    >
      <Form onSubmit={() => onSave({ smtp: draft })}>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Host">
            <Input value={draft.host} placeholder="smtp.example.com" onChange={(e) => setDraft({ ...draft, host: e.target.value })} />
          </Field>
          <Field label="Port">
            <Input type="number" value={draft.port} onChange={(e) => setDraft({ ...draft, port: Number(e.target.value) })} />
          </Field>
          <Field label="From address">
            <Input value={draft.from} placeholder="reeve@example.com" onChange={(e) => setDraft({ ...draft, from: e.target.value })} />
          </Field>
          <Field label="Encryption">
            <select
              className="w-full rounded-button border border-hairline-strong bg-canvas px-3 py-2 text-sm text-content focus:outline-none focus:ring-2 focus:ring-link"
              value={draft.tls}
              onChange={(e) => setDraft({ ...draft, tls: e.target.value as SMTPInput['tls'] })}
            >
              <option value="starttls">STARTTLS (587)</option>
              <option value="tls">TLS (465)</option>
              <option value="none">None</option>
            </select>
          </Field>
          <Field label="Username" hint="Leave empty for a relay that needs no auth">
            <Input value={draft.username} onChange={(e) => setDraft({ ...draft, username: e.target.value })} />
          </Field>
          <Field label="Password" hint={settings.smtp.password_set ? 'Stored; leave empty to keep it' : 'Not set'}>
            <Input
              type="password"
              value={draft.password}
              autoComplete="new-password"
              onChange={(e) => setDraft({ ...draft, password: e.target.value })}
            />
          </Field>
        </div>
        <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
          <label className="flex items-center gap-2 text-sm text-content">
            <input
              type="checkbox"
              className="h-4 w-4 accent-accent"
              checked={draft.enabled}
              onChange={(e) => setDraft({ ...draft, enabled: e.target.checked })}
            />
            Send email through this relay
          </label>
          <div className="flex items-center gap-3">
            {testing && <span className="text-xs text-muted">{testing}</span>}
            <Button variant="secondary" disabled={!settings.smtp.enabled} onClick={sendTest}>
              Send test email
            </Button>
            <Button type="submit">Save email</Button>
          </div>
        </div>
      </Form>
    </Section>
  );
}

function GoogleSection({ settings, onSave }: { settings: Settings; onSave: (patch: SettingsInput) => Promise<void> }) {
  const [draft, setDraft] = useState<GoogleInput>({
    enabled: settings.google.enabled,
    client_id: settings.google.client_id,
    allowed_domains: settings.google.allowed_domains,
    client_secret: '',
  });
  useEffect(
    () =>
      setDraft({
        enabled: settings.google.enabled,
        client_id: settings.google.client_id,
        allowed_domains: settings.google.allowed_domains,
        client_secret: '',
      }),
    [settings.google],
  );

  return (
    <Section
      title="Google sign-in"
      description="Let people sign in with a Google account. Create an OAuth 2.0 Web client in Google Cloud, then paste its credentials here."
    >
      {settings.google.redirect_url ? (
        <div>
          <p className="text-xs text-muted">Authorized redirect URI to register with Google</p>
          <p className="mt-1 select-all break-all rounded-button border border-hairline bg-surface-2 px-3 py-2 font-mono text-xs text-content">
            {settings.google.redirect_url}
          </p>
        </div>
      ) : (
        <p className="text-xs text-down">
          Set REEVE_PUBLIC_URL on the server first — Google matches the redirect URI exactly, so it cannot be
          derived per request.
        </p>
      )}

      <Form onSubmit={() => onSave({ google: draft })}>
        <div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Client ID">
            <Input value={draft.client_id} onChange={(e) => setDraft({ ...draft, client_id: e.target.value })} />
          </Field>
          <Field label="Client secret" hint={settings.google.secret_set ? 'Stored; leave empty to keep it' : 'Not set'}>
            <Input
              type="password"
              value={draft.client_secret}
              autoComplete="new-password"
              onChange={(e) => setDraft({ ...draft, client_secret: e.target.value })}
            />
          </Field>
        </div>
        <div className="mt-3">
          <Field
            label="Allowed email domains"
            hint="Comma separated; empty allows any. Listing a domain also lets its people in while sign-up is closed."
          >
            <Input
              value={draft.allowed_domains}
              placeholder="example.com"
              onChange={(e) => setDraft({ ...draft, allowed_domains: e.target.value })}
            />
          </Field>
        </div>
        <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
          <label className="flex items-center gap-2 text-sm text-content">
            <input
              type="checkbox"
              className="h-4 w-4 accent-accent"
              checked={draft.enabled}
              disabled={!settings.google.redirect_url}
              onChange={(e) => setDraft({ ...draft, enabled: e.target.checked })}
            />
            Offer Google sign-in on the login screen
          </label>
          <Button type="submit">Save Google</Button>
        </div>
      </Form>
    </Section>
  );
}

function RetentionSelect({ value, onChange }: { value: number; onChange: (secs: number) => void }) {
  const known = RETENTION_CHOICES.some((c) => c.secs === value);
  if (!known) {
    return (
      <Input
        type="number"
        value={value}
        min={600}
        onChange={(e) => onChange(Number(e.target.value))}
      />
    );
  }
  return (
    <select
      className="w-full rounded-button border border-hairline-strong bg-canvas px-3 py-2 text-sm text-content focus:outline-none focus:ring-2 focus:ring-link"
      value={value}
      onChange={(e) => onChange(Number(e.target.value))}
    >
      {RETENTION_CHOICES.map((c) => (
        <option key={c.secs} value={c.secs}>
          {c.label}
        </option>
      ))}
    </select>
  );
}

export function Section({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <Card className="mt-6 p-5">
      <h2 className="text-sm font-medium text-content">{title}</h2>
      <p className="mt-1 max-w-2xl text-xs text-muted">{description}</p>
      <div className="mt-4">{children}</div>
    </Card>
  );
}
