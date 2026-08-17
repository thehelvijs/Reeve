import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api, type PipelineGroup, type PipelineOverview, type PipelineProject } from '../api';
import { useAuth } from '../auth';
import { ErrorText, Pill, Section, Table, Td, Th, Tr } from '../components/ui';
import PageHeader from '../components/PageHeader';
import EmptyState from '../components/EmptyState';
import { pipelineTone } from '../lib/statusTone';
import { matchesQuery } from '../lib/search';

// GitLab is polled through the server, so a page left open follows the fleet
// without anyone reaching for reload.
const REFRESH_MS = 30_000;

export default function Pipelines() {
  const { user } = useAuth();
  const [data, setData] = useState<PipelineOverview | null>(null);
  const [error, setError] = useState('');
  const [search, setSearch] = useState('');

  const load = useCallback(() => {
    api
      .get<PipelineOverview>('/api/gitlab/pipelines')
      .then((d) => {
        setData(d);
        setError('');
      })
      .catch((e) => setError(e instanceof Error ? e.message : 'could not load pipelines'));
  }, []);

  useEffect(() => {
    load();
    const id = window.setInterval(load, REFRESH_MS);
    return () => window.clearInterval(id);
  }, [load]);

  const groups = data?.groups ?? [];
  const failing = groups.reduce(
    (n, g) => n + g.projects.filter((p) => p.status === 'failed').length,
    0,
  );

  let subtitle = 'Latest pipeline for every project in the configured GitLab groups.';
  if (failing > 0) {
    subtitle = `${failing} project${failing === 1 ? '' : 's'} failing.`;
  }

  return (
    <div>
      <PageHeader
        title="Pipelines"
        subtitle={subtitle}
        search={{ value: search, onChange: setSearch, placeholder: 'Search projects…' }}
      />
      <div className="mt-3">
        <ErrorText>{error}</ErrorText>
      </div>

      {data && !data.configured && (
        <div className="mt-6">
          <EmptyState
            title="No GitLab connection"
            description={
              user?.role === 'admin'
                ? 'Add the server URL, an access token and the groups to watch in Settings.'
                : 'An admin has not connected a GitLab server yet.'
            }
            action={
              user?.role === 'admin' ? (
                <Link to="/admin/settings" className="text-sm text-link underline underline-offset-2 hover:text-link-hover">
                  Open settings
                </Link>
              ) : undefined
            }
          />
        </div>
      )}

      <div className="mt-6 space-y-8">
        {groups.map((g) => (
          <GroupSection key={g.path} group={g} search={search} />
        ))}
      </div>
    </div>
  );
}

function GroupSection({ group, search }: { group: PipelineGroup; search: string }) {
  const shown = group.projects.filter((p) => matchesQuery(search, p.name, p.path, p.status, p.ref));

  let description: string | undefined;
  if (group.truncated) {
    description = `Showing the first ${group.projects.length} projects; the group holds more.`;
  }

  return (
    <Section
      title={group.path}
      count={group.projects.length}
      description={description}
      action={
        group.url && (
          <a
            href={group.url}
            target="_blank"
            rel="noreferrer"
            className="text-xs text-link underline underline-offset-2 hover:text-link-hover"
          >
            Open in GitLab
          </a>
        )
      }
    >
      {group.error && <ErrorText>{group.error}</ErrorText>}
      {!group.error && shown.length === 0 && (
        <EmptyState title="No projects" description="Nothing under this group matches." />
      )}
      {shown.length > 0 && (
        <Table
          head={
            <>
              <Th>Project</Th>
              <Th>Pipeline</Th>
              <Th>Branch</Th>
              <Th>Last run</Th>
            </>
          }
        >
          {shown.map((p) => (
            <ProjectRow key={p.path} project={p} />
          ))}
        </Table>
      )}
    </Section>
  );
}

function ProjectRow({ project }: { project: PipelineProject }) {
  let status = <Pill tone="muted">never run</Pill>;
  if (project.status) {
    status = <Pill tone={pipelineTone(project.status)}>{project.status.replace(/_/g, ' ')}</Pill>;
  }

  let ran = '—';
  if (project.updated_at) {
    ran = new Date(project.updated_at).toLocaleString();
  }

  return (
    <Tr>
      <Td>
        <a
          href={project.url}
          target="_blank"
          rel="noreferrer"
          className="text-link underline underline-offset-2 hover:text-link-hover"
        >
          {project.name}
        </a>
        <span className="block text-xs text-muted">{project.path}</span>
      </Td>
      <Td>
        {project.pipeline_url ? (
          <a href={project.pipeline_url} target="_blank" rel="noreferrer">
            {status}
          </a>
        ) : (
          status
        )}
      </Td>
      <Td className="font-mono text-xs text-muted">{project.ref || '—'}</Td>
      <Td className="whitespace-nowrap text-xs text-muted">{ran}</Td>
    </Tr>
  );
}
