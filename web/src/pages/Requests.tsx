import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api, type AccessRequest } from '../api';
import { Button, Card, Pill } from '../components/ui';
import PageHeader from '../components/PageHeader';
import EmptyState from '../components/EmptyState';

export default function Requests() {
  const [box, setBox] = useState<'inbox' | 'mine'>('inbox');
  const [reqs, setReqs] = useState<AccessRequest[]>([]);

  const load = useCallback(() => {
    api.get<AccessRequest[]>(`/api/access-requests?box=${box}`).then((r) => setReqs(r ?? []));
  }, [box]);
  useEffect(() => {
    load();
  }, [load]);

  const decide = async (id: string, action: 'approve' | 'deny') => {
    await api.post(`/api/access-requests/${id}/${action}`, {});
    load();
  };

  return (
    <div>
      <PageHeader title="Access requests" subtitle="Approve or track requests to reveal host credentials." />
      <div className="mt-6 flex gap-2">
        <Tab active={box === 'inbox'} onClick={() => setBox('inbox')}>
          To review
        </Tab>
        <Tab active={box === 'mine'} onClick={() => setBox('mine')}>
          Mine
        </Tab>
      </div>

      <div className="mt-6 space-y-2">
        {reqs.length === 0 && (
          <EmptyState
            title={box === 'inbox' ? 'Nothing to review' : 'No requests yet'}
            description={
              box === 'inbox'
                ? 'Requests to reveal host credentials will show up here.'
                : 'Credential access you request will show up here.'
            }
          />
        )}
        {reqs.map((r) => (
          <Card key={r.id} className="flex items-center justify-between px-4 py-3">
            <div className="min-w-0">
              <Link to={`/hosts/${r.host_id}`} className="text-sm text-content hover:text-accent">
                {r.host_name || r.host_id}
              </Link>
              {r.note && <p className="truncate text-xs text-muted">“{r.note}”</p>}
              <p className="text-xs text-muted">{new Date(r.created_at).toLocaleString()}</p>
            </div>
            {box === 'inbox' ? (
              <div className="flex gap-2">
                <Button onClick={() => decide(r.id, 'approve')}>Approve</Button>
                <Button variant="secondary" onClick={() => decide(r.id, 'deny')}>
                  Deny
                </Button>
              </div>
            ) : (
              <Pill tone={r.status === 'approved' ? 'up' : r.status === 'denied' ? 'down' : 'muted'}>
                {r.status}
              </Pill>
            )}
          </Card>
        ))}
      </div>
    </div>
  );
}

function Tab({ active, onClick, children }: { active: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button
      onClick={onClick}
      className={`rounded-button px-3 py-1.5 text-sm transition-colors ${
        active ? 'bg-surface-2 text-content' : 'text-muted hover:text-content'
      }`}
    >
      {children}
    </button>
  );
}
