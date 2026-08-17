import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  api,
  type PipelineGroupRecord,
  type PipelineProject,
  type Provider,
  type Settings,
} from '../api';
import { Button, Card, ErrorText, Form, Input, Pill, Section } from '../components/ui';
import PageHeader from '../components/PageHeader';
import EmptyState from '../components/EmptyState';
import ConfirmModal from '../components/ConfirmModal';

// How long the picker waits after the last keystroke before asking the forge.
// Long enough that typing a repo name is one search, short enough to feel live.
const SEARCH_DEBOUNCE_MS = 250;

const PROVIDER_LABEL: Record<Provider, string> = { gitlab: 'GitLab', github: 'GitHub' };

export default function PipelineGroups() {
  const [groups, setGroups] = useState<PipelineGroupRecord[]>([]);
  const [connected, setConnected] = useState<Provider[]>([]);
  const [name, setName] = useState('');
  const [provider, setProvider] = useState<Provider>('gitlab');
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    try {
      const [rows, settings] = await Promise.all([
        api.get<PipelineGroupRecord[]>('/api/pipeline-groups'),
        api.get<Settings>('/api/admin/settings'),
      ]);
      setGroups(rows);
      const live: Provider[] = [];
      if (settings.gitlab.token_set) {
        live.push('gitlab');
      }
      if (settings.github.token_set) {
        live.push('github');
      }
      setConnected(live);
      setProvider((p) => {
        if (live.includes(p) || live.length === 0) {
          return p;
        }
        return live[0];
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : 'could not load groups');
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const create = async () => {
    if (!name.trim()) {
      return;
    }
    setError('');
    try {
      await api.post('/api/admin/pipeline-groups', { name: name.trim(), provider });
      setName('');
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'could not create the group');
    }
  };

  return (
    <div>
      <PageHeader
        title="Pipeline groups"
        subtitle="Your own sets of repos. A group is watched together on the Pipelines page, whatever GitLab group or GitHub org each repo actually lives in."
      />
      <div className="mt-3">
        <ErrorText>{error}</ErrorText>
      </div>

      {connected.length === 0 ? (
        <div className="mt-6">
          <EmptyState
            title="No forge connected"
            description="Connect GitLab or GitHub before building a group — the picker searches the forge for repos, so it needs a token first."
            action={
              <Link to="/admin/settings">
                <Button>Open settings</Button>
              </Link>
            }
          />
        </div>
      ) : (
        <Card className="mt-6 p-5">
          <Form onSubmit={create}>
            <div className="flex flex-wrap items-end gap-3">
              <div className="min-w-64 flex-1">
                <Input
                  value={name}
                  placeholder="Firmware"
                  aria-label="New group name"
                  onChange={(e) => setName(e.target.value)}
                />
              </div>
              {connected.length > 1 && (
                <div className="flex items-center gap-4">
                  {connected.map((p) => (
                    <label key={p} className="flex items-center gap-2 text-sm text-content">
                      <input
                        type="radio"
                        name="new-group-provider"
                        className="h-4 w-4 accent-accent"
                        checked={provider === p}
                        onChange={() => setProvider(p)}
                      />
                      {PROVIDER_LABEL[p]}
                    </label>
                  ))}
                </div>
              )}
              <Button type="submit" disabled={!name.trim()}>
                New {connected.length === 1 ? PROVIDER_LABEL[connected[0]] : ''} group
              </Button>
            </div>
          </Form>
        </Card>
      )}

      <div className="mt-8 space-y-8">
        {groups.length === 0 && connected.length > 0 && (
          <EmptyState
            title="No groups yet"
            description="Name a group above, then search the forge for the repos that belong in it."
          />
        )}
        {groups.map((g) => (
          <GroupEditor key={g.id} group={g} onChanged={load} />
        ))}
      </div>
    </div>
  );
}

function GroupEditor({ group, onChanged }: { group: PipelineGroupRecord; onChanged: () => Promise<void> }) {
  const [name, setName] = useState(group.name);
  const [confirming, setConfirming] = useState(false);
  const [error, setError] = useState('');
  useEffect(() => setName(group.name), [group.name]);

  const run = async (fn: () => Promise<unknown>) => {
    setError('');
    try {
      await fn();
      await onChanged();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'that did not work');
    }
  };

  const rename = () => run(() => api.patch(`/api/admin/pipeline-groups/${group.id}`, { name: name.trim() }));
  const remove = (path: string) =>
    run(() => api.del(`/api/admin/pipeline-groups/${group.id}/projects`, { path }));

  return (
    <Section
      title={group.name}
      count={group.projects.length}
      action={
        <>
          <Pill>{PROVIDER_LABEL[group.provider]}</Pill>
          <Button variant="danger" onClick={() => setConfirming(true)}>
            Delete group
          </Button>
        </>
      }
    >
      <Card className="p-5">
        <ErrorText>{error}</ErrorText>
        <Form onSubmit={rename}>
          <div className="flex flex-wrap items-end gap-3">
            <div className="min-w-64 flex-1">
              <Input value={name} aria-label="Group name" onChange={(e) => setName(e.target.value)} />
            </div>
            <Button type="submit" variant="secondary" disabled={!name.trim() || name.trim() === group.name}>
              Rename
            </Button>
          </div>
        </Form>

        <div className="mt-5">
          <ProjectPicker
            group={group}
            onAdd={(path) => run(() => api.post(`/api/admin/pipeline-groups/${group.id}/projects`, { path }))}
          />
        </div>

        <ul className="mt-5 divide-y divide-hairline border-t border-hairline">
          {group.projects.length === 0 && (
            <li className="py-3 text-sm text-muted">No repos in this group yet.</li>
          )}
          {group.projects.map((path) => (
            <li key={path} className="flex items-center justify-between gap-3 py-2">
              <span className="truncate font-mono text-xs text-content">{path}</span>
              <Button variant="secondary" onClick={() => remove(path)}>
                Remove
              </Button>
            </li>
          ))}
        </ul>
      </Card>

      {confirming && (
        <ConfirmModal
          title={`Delete ${group.name}?`}
          body="The group and its list of repos go away. Nothing on the forge is touched."
          confirmLabel="Delete group"
          onConfirm={() => run(() => api.del(`/api/admin/pipeline-groups/${group.id}`))}
          onClose={() => setConfirming(false)}
        />
      )}
    </Section>
  );
}

// ProjectPicker searches the group's own forge rather than filtering a list Reeve
// holds: there is no local copy of what repos exist, and typing a full path from
// memory is exactly what this page is here to avoid.
function ProjectPicker({
  group,
  onAdd,
}: {
  group: PipelineGroupRecord;
  onAdd: (path: string) => Promise<void>;
}) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<PipelineProject[]>([]);
  const [error, setError] = useState('');
  const [searching, setSearching] = useState(false);

  useEffect(() => {
    let live = true;
    setSearching(true);
    const id = window.setTimeout(() => {
      api
        .get<PipelineProject[]>(
          `/api/admin/repo-search?provider=${group.provider}&q=${encodeURIComponent(query)}`,
        )
        .then((r) => {
          if (live) {
            setResults(r);
            setError('');
          }
        })
        .catch((e) => {
          if (live) {
            setResults([]);
            setError(e instanceof Error ? e.message : 'search failed');
          }
        })
        .finally(() => {
          if (live) {
            setSearching(false);
          }
        });
    }, SEARCH_DEBOUNCE_MS);
    return () => {
      live = false;
      window.clearTimeout(id);
    };
  }, [query, group.provider]);

  const inGroup = new Set(group.projects);
  const offered = results.filter((p) => !inGroup.has(p.path));

  // A search can come up empty for reasons the picker cannot fix: a token that
  // cannot list, a repo past the search cap, an instance that answers slowly.
  // The path is what a group stores, so typing one has to keep working.
  const typedPath = query.trim().replace(/^\/+|\/+$/g, '');
  const canAddTyped =
    !searching &&
    typedPath.includes('/') &&
    !inGroup.has(typedPath) &&
    !offered.some((p) => p.path === typedPath);

  let note = '';
  if (searching) {
    note = 'Searching…';
  } else if (offered.length === 0 && !error) {
    note = `Nothing new matches. ${PROVIDER_LABEL[group.provider]} lists the most recently active repos when the box is empty.`;
  }

  return (
    <div>
      <Input
        value={query}
        placeholder={`Search ${PROVIDER_LABEL[group.provider]} repos, or paste a full path…`}
        aria-label={`Search ${PROVIDER_LABEL[group.provider]} repos`}
        onChange={(e) => setQuery(e.target.value)}
      />
      <ErrorText>{error}</ErrorText>
      {note && <p className="mt-2 text-xs text-muted">{note}</p>}
      {canAddTyped && (
        <div className="mt-2 flex items-center justify-between gap-3 rounded-card border border-hairline px-3 py-2">
          <span className="min-w-0 truncate font-mono text-xs text-content">{typedPath}</span>
          <Button variant="secondary" onClick={() => onAdd(typedPath)}>
            Add this path
          </Button>
        </div>
      )}
      {offered.length > 0 && (
        <ul className="mt-2 max-h-64 overflow-y-auto rounded-card border border-hairline">
          {offered.map((p) => (
            <li
              key={p.path}
              className="flex items-center justify-between gap-3 border-b border-hairline px-3 py-2 last:border-0"
            >
              <span className="min-w-0">
                <span className="block truncate text-sm text-content">{p.name}</span>
                <span className="block truncate font-mono text-xs text-muted">{p.path}</span>
              </span>
              <Button variant="secondary" onClick={() => onAdd(p.path)}>
                Add
              </Button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
