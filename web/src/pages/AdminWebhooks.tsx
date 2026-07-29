import { useEffect, useState } from 'react';
import ConfirmModal from '../components/ConfirmModal';
import {
  api,
  type ChannelFormats,
  type ChannelKind,
  type ChannelScope,
  type Host,
  type Principals,
  type Severity,
  type Tool,
  type Webhook,
} from '../api';
import { Button, Card, ErrorText, Field, Form, Input, Pill } from '../components/ui';
import PageHeader from '../components/PageHeader';
import { matchesQuery } from '../lib/search';
import { EVENT_LABEL, FORMAT_LABEL, extraConfigKey, formatNeedsTemplate } from '../lib/channelFormat';

const selectClass =
  'rounded-button border border-hairline-strong bg-canvas px-3 py-2 text-sm text-content focus:outline-none focus-visible:ring-2 focus-visible:ring-link';
const textareaClass =
  'w-full rounded-button border border-hairline-strong bg-canvas px-3 py-2 font-mono text-xs text-content placeholder:text-muted focus:outline-none focus-visible:ring-2 focus-visible:ring-link';

// SCOPES is what a channel can watch, broadest first. Global receives every
// event; a narrower channel adds a receiver rather than taking traffic off the
// broad one, so an operator can page one team about one host and keep the
// instance-wide feed intact.
const SCOPES: { value: ChannelScope; label: string; noun: string }[] = [
  { value: 'global', label: 'Everything', noun: '' },
  { value: 'host', label: 'One host', noun: 'Host' },
  { value: 'tool', label: 'One service', noun: 'Service' },
  { value: 'group', label: 'What a group can reach', noun: 'Group' },
];

// scopeTargets is every entity a channel can be pointed at, so the form offers
// names instead of asking an operator to paste an id.
interface ScopeTargets {
  hosts: Host[];
  tools: Tool[];
  groups: { id: string; name: string }[];
}

const NO_TARGETS: ScopeTargets = { hosts: [], tools: [], groups: [] };

function targetsFor(scope: ChannelScope, t: ScopeTargets): { id: string; name: string }[] {
  if (scope === 'host') {
    return t.hosts;
  }
  if (scope === 'tool') {
    return t.tools;
  }
  if (scope === 'group') {
    return t.groups;
  }
  return [];
}

// A channel as the form holds it: the shape both the create card and a row's
// editor post, so the two cannot drift.
interface Draft {
  url: string;
  format: ChannelKind;
  minSeverity: Severity;
  events: string[];
  token: string;
  template: string;
  extra: string;
}

const EMPTY_DRAFT: Draft = {
  url: '',
  format: 'auto',
  minSeverity: 'info',
  events: [],
  token: '',
  template: '',
  extra: '',
};

function draftFrom(h: Webhook): Draft {
  return {
    url: h.url,
    format: h.format,
    minSeverity: h.min_severity,
    events: h.events ?? [],
    token: '',
    template: h.config.template ?? '',
    extra: h.config[extraConfigKey(h.format) ?? ''] ?? '',
  };
}

// Only a value the operator typed is sent: a blank token or routing key leaves
// the stored one alone, because every read redacts it.
function draftConfig(d: Draft): Record<string, string> {
  const config: Record<string, string> = {};
  if (d.token.trim()) {
    config.token = d.token.trim();
  }
  if (formatNeedsTemplate(d.format)) {
    config.template = d.template;
  }
  const key = extraConfigKey(d.format);
  if (key && d.extra.trim()) {
    config[key] = d.extra.trim();
  }
  return config;
}

export default function AdminWebhooks() {
  const [hooks, setHooks] = useState<Webhook[]>([]);
  const [meta, setMeta] = useState<ChannelFormats | null>(null);
  const [targets, setTargets] = useState<ScopeTargets>(NO_TARGETS);
  const [ownerType, setOwnerType] = useState<ChannelScope>('global');
  const [ownerId, setOwnerId] = useState('');
  const [draft, setDraft] = useState<Draft>(EMPTY_DRAFT);
  const [error, setError] = useState('');
  const [search, setSearch] = useState('');
  const [editing, setEditing] = useState<string | null>(null);
  // One result per row plus one for the form, keyed by webhook id and 'new' for
  // the unsaved one, so testing a second channel does not clear the first verdict.
  const [tested, setTested] = useState<Record<string, string>>({});
  const [testing, setTesting] = useState<string | null>(null);

  // A saved channel tests by id: its token is redacted in every read, so only the
  // server can put the real one on the wire.
  const test = async (key: string, body: Record<string, unknown>) => {
    setTesting(key);
    setTested((t) => ({ ...t, [key]: '' }));
    try {
      const out = await api.post<{ ok: boolean; error?: string }>('/api/admin/webhooks/test', body);
      if (out.ok) {
        setTested((t) => ({ ...t, [key]: 'delivered' }));
      } else {
        setTested((t) => ({ ...t, [key]: out.error || 'the receiver rejected it' }));
      }
    } catch (e) {
      setTested((t) => ({ ...t, [key]: e instanceof Error ? e.message : 'test failed' }));
    }
    setTesting(null);
  };

  const testDraft = () =>
    test('new', { url: draft.url.trim(), format: draft.format, config: draftConfig(draft) });

  const load = () => api.get<Webhook[]>('/api/admin/webhooks').then((h) => setHooks(h ?? []));
  useEffect(() => {
    load();
    api.get<ChannelFormats>('/api/admin/webhook-formats').then(setMeta).catch(() => setMeta(null));
    Promise.all([
      api.get<Host[]>('/api/hosts').catch(() => []),
      api.get<Tool[]>('/api/tools').catch(() => []),
      api.get<Principals>('/api/principals').catch(() => null),
    ]).then(([hosts, tools, principals]) =>
      setTargets({ hosts: hosts ?? [], tools: tools ?? [], groups: principals?.groups ?? [] }),
    );
  }, []);

  const create = async () => {
    setError('');
    try {
      await api.post('/api/admin/webhooks', {
        owner_type: ownerType,
        owner_id: ownerId,
        url: draft.url,
        format: draft.format,
        min_severity: draft.minSeverity,
        events: draft.events,
        config: draftConfig(draft),
      });
      setDraft(EMPTY_DRAFT);
      setOwnerId('');
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'failed');
    }
  };
  const [confirming, setConfirming] = useState<Webhook | null>(null);

  const remove = async (id: string) => {
    await api.del(`/api/admin/webhooks/${id}`);
    load();
  };

  const shown = hooks.filter((h) =>
    matchesQuery(search, h.url, h.owner_type, h.owner_id, h.min_severity, h.format),
  );

  return (
    <div>
      <PageHeader
        title="Webhooks"
        search={{ value: search, onChange: setSearch, placeholder: 'Search webhooks…' }}
      />

      <Card className="mt-6 p-5">
        <Form onSubmit={create}>
          <ScopeFields
            scope={ownerType}
            ownerId={ownerId}
            targets={targets}
            onChange={(scope, id) => {
              setOwnerType(scope);
              setOwnerId(id);
            }}
          />

          <ChannelFields draft={draft} onChange={setDraft} meta={meta} />

          <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
            <p className="text-xs text-muted">The token is encrypted at rest and never shown again after saving.</p>
            <div className="flex shrink-0 items-center gap-2">
              <TestResult result={tested.new} />
              <Button
                type="button"
                variant="secondary"
                disabled={!draft.url.trim() || testing === 'new'}
                onClick={testDraft}
              >
                {testing === 'new' ? 'Testing…' : 'Test'}
              </Button>
              <Button type="submit" disabled={!draft.url.trim() || (ownerType !== 'global' && !ownerId)}>
                Add webhook
              </Button>
            </div>
          </div>
          <ErrorText>{error}</ErrorText>
        </Form>
      </Card>

      <div className="mt-6 space-y-2">
        {hooks.length === 0 && <p className="text-sm text-muted">No webhooks configured.</p>}
        {hooks.length > 0 && shown.length === 0 && (
          <p className="text-sm text-muted">No webhook matches the search.</p>
        )}
        {shown.map((h) => (
          <Card key={h.id} className="px-4 py-3">
            <div className="flex items-center justify-between gap-3">
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <Pill>{scopeLabel(h, targets)}</Pill>
                  <Pill>{h.min_severity}</Pill>
                  <span className="text-xs text-muted">{receiverLabel(h)}</span>
                </div>
                {h.url && <p className="mt-1 truncate font-mono text-xs text-muted">{h.url}</p>}
              </div>
              <div className="flex shrink-0 items-center gap-2">
                <TestResult result={tested[h.id]} />
                <Button
                  variant="secondary"
                  disabled={testing === h.id}
                  onClick={() => test(h.id, { id: h.id })}
                >
                  {testing === h.id ? 'Testing…' : 'Test'}
                </Button>
                <Button variant="secondary" onClick={() => setEditing(editing === h.id ? null : h.id)}>
                  {editing === h.id ? 'Close' : 'Edit'}
                </Button>
                <Button variant="danger" onClick={() => setConfirming(h)}>
                  Delete
                </Button>
              </div>
            </div>
            {editing === h.id && (
              <ChannelEditor
                hook={h}
                meta={meta}
                onClose={() => setEditing(null)}
                onSaved={() => {
                  setEditing(null);
                  load();
                }}
              />
            )}
          </Card>
        ))}
      </div>

      {confirming && (
        <ConfirmModal
          title="Delete this webhook?"
          body={`Alerts stop being delivered to ${confirming.url || 'it'}. Nothing already delivered is affected.`}
          confirmLabel="Delete"
          onConfirm={() => remove(confirming.id)}
          onClose={() => setConfirming(null)}
        />
      )}
    </div>
  );
}

// scopeLabel names what a channel watches. An id whose entity has since been
// deleted still reads as the id, because a channel pointing at nothing is worth
// seeing rather than hiding.
function scopeLabel(h: Webhook, targets: ScopeTargets): string {
  if (h.owner_type === 'global') {
    return 'everything';
  }
  const name = targetsFor(h.owner_type, targets).find((t) => t.id === h.owner_id)?.name;
  return `${h.owner_type}: ${name ?? h.owner_id}`;
}

// receiverLabel says what the payload will be shaped as. An 'auto' channel names
// what the URL resolved to, because that is the answer the operator wants.
function receiverLabel(h: Webhook): string {
  if (h.format === 'auto') {
    return `auto · ${FORMAT_LABEL[h.detected] ?? h.detected}`;
  }
  return FORMAT_LABEL[h.format] ?? h.format;
}

// ScopeFields picks what a channel watches. Only the create card renders it: a
// channel's scope never changes, because a global hook that should have been a
// host hook is a different channel, not an edit.
function ScopeFields({
  scope,
  ownerId,
  targets,
  onChange,
}: {
  scope: ChannelScope;
  ownerId: string;
  targets: ScopeTargets;
  onChange: (scope: ChannelScope, ownerId: string) => void;
}) {
  const noun = SCOPES.find((s) => s.value === scope)?.noun ?? '';
  const options = targetsFor(scope, targets);
  return (
    <div className="flex flex-wrap items-end gap-3">
      <Field label="Watches">
        <select
          value={scope}
          onChange={(e) => onChange(e.target.value as ChannelScope, '')}
          className={selectClass}
        >
          {SCOPES.map((s) => (
            <option key={s.value} value={s.value}>
              {s.label}
            </option>
          ))}
        </select>
      </Field>
      {noun && (
        <Field label={noun}>
          <select
            value={ownerId}
            onChange={(e) => onChange(scope, e.target.value)}
            className={selectClass}
          >
            <option value="">Choose a {noun.toLowerCase()}…</option>
            {options.map((o) => (
              <option key={o.id} value={o.id}>
                {o.name}
              </option>
            ))}
          </select>
        </Field>
      )}
    </div>
  );
}

// ChannelFields is the URL, receiver, severity gate and whatever that receiver
// needs on top. Both the create card and a row's editor render it.
function ChannelFields({
  draft,
  onChange,
  meta,
}: {
  draft: Draft;
  onChange: (d: Draft) => void;
  meta: ChannelFormats | null;
}) {
  const formats = meta?.formats ?? (['auto', 'generic', 'custom'] as ChannelKind[]);
  const extraKey = extraConfigKey(draft.format);
  return (
    <>
      <div className="mt-3 flex flex-wrap items-end gap-3">
        <div className="min-w-64 grow">
          <Field label="URL">
            <Input
              value={draft.url}
              onChange={(e) => onChange({ ...draft, url: e.target.value })}
              placeholder="https://sink.example.com/hook"
            />
          </Field>
        </div>
        <Field label="Receiver">
          <select
            value={draft.format}
            onChange={(e) => onChange({ ...draft, format: e.target.value as ChannelKind })}
            className={selectClass}
          >
            {formats.map((f) => (
              <option key={f} value={f}>
                {FORMAT_LABEL[f] ?? f}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Min severity">
          <select
            value={draft.minSeverity}
            onChange={(e) => onChange({ ...draft, minSeverity: e.target.value as Severity })}
            className={selectClass}
          >
            <option value="info">info</option>
            <option value="warning">warning</option>
            <option value="error">error</option>
          </select>
        </Field>
      </div>

      <EventFields
        selected={draft.events}
        all={meta?.events ?? []}
        onChange={(events) => onChange({ ...draft, events })}
      />

      {draft.format === 'auto' && (
        <p className="mt-1.5 text-xs text-muted">
          Read from the URL. Discord, Slack, Google Chat, Teams, Webex, ntfy, Telegram and PagerDuty
          are recognised. A Mattermost or Rocket.Chat server runs on your own domain, so pick it by
          name.
        </p>
      )}

      <div className="mt-3">
        <Field label="Bearer token (optional)">
          <Input
            type="password"
            value={draft.token}
            onChange={(e) => onChange({ ...draft, token: e.target.value })}
            placeholder="Leave blank to keep the stored one"
          />
        </Field>
      </div>

      {extraKey && (
        <div className="mt-3">
          <Field
            label={extraKey === 'chat_id' ? 'Telegram chat id' : 'PagerDuty routing key'}
            hint={
              extraKey === 'chat_id'
                ? 'The chat the bot posts to. Telegram rejects a message without it.'
                : 'The integration key of the service this alerts. It goes in the body, not a header.'
            }
          >
            <Input
              value={draft.extra}
              onChange={(e) => onChange({ ...draft, extra: e.target.value })}
              placeholder={extraKey === 'chat_id' ? '-1001234567890' : ''}
            />
          </Field>
        </div>
      )}

      {formatNeedsTemplate(draft.format) && (
        <div className="mt-3">
          <Field
            label="Body template"
            hint="JSON posted as-is, with each {{name}} replaced by that field of the alert. Values are escaped for you."
          >
            <textarea
              rows={5}
              value={draft.template}
              onChange={(e) => onChange({ ...draft, template: e.target.value })}
              className={textareaClass}
              placeholder={'{\n  "text": "[{{severity}}] {{tool}}: {{message}}"\n}'}
              spellCheck={false}
            />
          </Field>
          <p className="mt-1.5 text-xs text-muted">
            Variables:{' '}
            {(meta?.variables ?? []).map((v, i) => (
              <span key={v}>
                {i > 0 && ', '}
                <code className="font-mono">{`{{${v}}}`}</code>
              </span>
            ))}
          </p>
        </div>
      )}
    </>
  );
}

// EventFields chooses which events reach a channel. An empty selection means
// every type, so a channel nobody narrowed keeps receiving a type added after it
// was saved; that is why every box reads as checked when the list is empty. The
// last checked box cannot be cleared, because a channel that receives nothing is
// a channel to disable, not to configure.
function EventFields({
  selected,
  all,
  onChange,
}: {
  selected: string[];
  all: string[];
  onChange: (events: string[]) => void;
}) {
  if (all.length === 0) {
    return null;
  }
  const checked = (e: string) => selected.length === 0 || selected.includes(e);
  const count = all.filter(checked).length;
  const toggle = (e: string) => {
    const next = all.filter((x) => (x === e ? !checked(x) : checked(x)));
    onChange(next.length === all.length ? [] : next);
  };
  return (
    <fieldset className="mt-3">
      <legend className="text-xs font-semibold text-content">Triggers on</legend>
      <div className="mt-1.5 grid grid-cols-1 gap-x-4 gap-y-1.5 sm:grid-cols-2 lg:grid-cols-3">
        {all.map((e) => (
          <label key={e} className="flex items-center gap-2 text-sm text-content">
            <input
              type="checkbox"
              className="size-4 rounded-button accent-link"
              checked={checked(e)}
              disabled={checked(e) && count === 1}
              onChange={() => toggle(e)}
            />
            {EVENT_LABEL[e] ?? e}
          </label>
        ))}
      </div>
      <p className="mt-1.5 text-xs text-muted">
        {count === all.length
          ? 'Everything, including any event type added later.'
          : `${count} of ${all.length} event types.`}
      </p>
    </fieldset>
  );
}

// ChannelEditor saves a row in place. A channel's scope never changes: a global
// hook that should have been a tool hook is a different channel, not an edit.
function ChannelEditor({
  hook,
  meta,
  onClose,
  onSaved,
}: {
  hook: Webhook;
  meta: ChannelFormats | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [draft, setDraft] = useState<Draft>(() => draftFrom(hook));
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  const save = async () => {
    setError('');
    setSaving(true);
    try {
      await api.patch(`/api/admin/webhooks/${hook.id}`, {
        url: draft.url,
        format: draft.format,
        min_severity: draft.minSeverity,
        events: draft.events,
        enabled: hook.enabled,
        config: draftConfig(draft),
      });
      onSaved();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'could not save the channel');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="mt-3 border-t border-hairline pt-3">
      <Form onSubmit={save}>
        <ChannelFields draft={draft} onChange={setDraft} meta={meta} />
        <div className="mt-4 flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={saving || !draft.url.trim()}>
            {saving ? 'Saving…' : 'Save'}
          </Button>
        </div>
        <ErrorText>{error}</ErrorText>
      </Form>
    </div>
  );
}

// The verdict sits beside the button that produced it: a receiver that answers
// badly is the common case, and its own words are more use than "failed".
function TestResult({ result }: { result?: string }) {
  if (!result) {
    return null;
  }
  if (result === 'delivered') {
    return <span className="text-xs text-up">Delivered</span>;
  }
  return <span className="max-w-64 truncate text-xs text-down" title={result}>{result}</span>;
}
