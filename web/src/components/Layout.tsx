import { useState } from 'react';
import { NavLink, Outlet, useMatch, useNavigate } from 'react-router-dom';
import { api, type Collection, type Host, type Tool } from '../api';
import { useAuth } from '../auth';
import { Button } from './ui';
import Avatar from './Avatar';
import SearchBar from './SearchBar';
import NavIcon, { type IconName } from './NavIcon';
import Wordmark from './Wordmark';
import ThemeToggle from './ThemeToggle';
import { useResource } from '../lib/cache';
import { SOURCE_URL } from '../version';

// How many records a section lists before it needs the toggle. Five keeps three
// sections and the admin group on one screen at the shortest viewport we design
// for; the rest is one click away.
const NAV_PREVIEW = 5;

const primaryNav: { to: string; label: string; icon: IconName; end: boolean }[] = [
  { to: '/dashboard', label: 'Dashboard', icon: 'dashboard', end: true },
  { to: '/services', label: 'Services', icon: 'services', end: false },
  { to: '/collections', label: 'Collections', icon: 'collections', end: false },
  { to: '/hosts', label: 'Hosts', icon: 'hosts', end: false },
  { to: '/requests', label: 'Requests', icon: 'requests', end: false },
];

const adminNav: { to: string; label: string; icon: IconName }[] = [
  { to: '/admin/users', label: 'Users', icon: 'users' },
  { to: '/admin/groups', label: 'User groups', icon: 'groups' },
  { to: '/admin/webhooks', label: 'Webhooks', icon: 'webhooks' },
  { to: '/admin/alerts', label: 'Alerts', icon: 'alerts' },
  { to: '/admin/audit', label: 'Audit', icon: 'audit' },
  { to: '/admin/server', label: 'Server', icon: 'server' },
  { to: '/admin/settings', label: 'Settings', icon: 'settings' },
];

export default function Layout() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [search, setSearch] = useState('');
  // The same cache keys the pages read, so the sidebar's lists cost no extra
  // request and update with whatever the open page just refreshed.
  const tools = useResource<Tool[]>('/api/tools', () => api.get<Tool[]>('/api/tools')).data ?? [];
  const collections = useResource<Collection[]>('/api/collections', () => api.get<Collection[]>('/api/collections')).data ?? [];
  const hosts = useResource<Host[]>('/api/hosts', () => api.get<Host[]>('/api/hosts')).data ?? [];

  const records: Record<string, { to: string; label: string }[]> = {
    '/services': tools.map((t) => ({ to: `/services/${t.id}`, label: t.name })),
    '/collections': collections.map((c) => ({ to: `/collections/${c.id}`, label: c.name })),
    '/hosts': hosts.map((h) => ({ to: `/hosts/${h.id}`, label: h.name })),
  };

  const submitSearch = () => navigate(`/services?q=${encodeURIComponent(search.trim())}`);

  return (
    <div className="flex h-screen">
      <aside className="flex w-60 shrink-0 flex-col border-r border-hairline bg-surface-1">
        <div className="px-4 py-4">
          <Wordmark />
        </div>

        <div className="px-3">
          <SearchBar value={search} onChange={setSearch} onSubmit={submitSearch} />
        </div>

        <nav className="mt-4 flex-1 space-y-0.5 overflow-y-auto px-3">
          {primaryNav.map((item) => (
            <NavSection key={item.to} item={item} records={records[item.to] ?? []} />
          ))}
          {user?.role === 'admin' && (
            <>
              <p className="px-2 pb-1 pt-5 text-eyebrow font-semibold uppercase text-muted">Admin</p>
              {adminNav.map((item) => (
                <NavItem key={item.to} {...item} end={false} />
              ))}
            </>
          )}
        </nav>

        <div className="border-t border-hairline p-3">
          <NavLink
            to="/profile"
            className={({ isActive }) =>
              `flex items-center gap-2 rounded-button px-2 py-1.5 transition-colors hover:bg-surface-2 ${
                isActive ? 'bg-surface-2' : ''
              }`
            }
          >
            <Avatar url={user?.avatar_url} name={user?.display_name} email={user?.email ?? ''} size={28} />
            <span className="min-w-0 flex-1 truncate text-xs text-content">{user?.display_name || user?.email}</span>
          </NavLink>
          <div className="mt-2 flex gap-2">
            <Button variant="secondary" className="flex-1" onClick={() => logout()}>
              Sign out
            </Button>
            <ThemeToggle />
          </div>
          <a
            href={SOURCE_URL}
            target="_blank"
            rel="noreferrer"
            className="mt-2 block text-center text-[10px] text-muted transition-colors hover:text-content"
          >
            Source · AGPL-3.0
          </a>
        </div>
      </aside>

      <main className="min-w-0 flex-1 overflow-auto bg-canvas">
        <div className="mx-auto max-w-7xl px-8 py-6">
          <Outlet />
        </div>
      </main>
    </div>
  );
}

interface NavRecord {
  to: string;
  label: string;
}

// A section is its own nav link plus, while that section is open, the first few
// of its records and a toggle for the rest. The other sections stay a single
// line: the sidebar is navigation, not a second list page.
function NavSection({
  item,
  records,
}: {
  item: { to: string; label: string; icon: IconName; end: boolean };
  records: NavRecord[];
}) {
  const [all, setAll] = useState(false);
  const open = useMatch({ path: item.to, end: false }) !== null;
  let shown: NavRecord[] = [];
  if (open) {
    shown = records;
    if (!all) {
      shown = records.slice(0, NAV_PREVIEW);
    }
  }
  return (
    <div>
      <NavItem {...item} />
      {shown.length > 0 && (
        <div className="mb-1 ml-4 border-l border-hairline pl-1">
          {shown.map((r) => (
            <NavLink
              key={r.to}
              to={r.to}
              className={({ isActive }) =>
                `block truncate rounded-button px-2 py-1 text-xs transition-colors ${
                  isActive ? 'bg-surface-2 font-semibold text-content' : 'text-muted hover:bg-surface-2 hover:text-content'
                }`
              }
            >
              {r.label}
            </NavLink>
          ))}
          {records.length > NAV_PREVIEW && (
            <button
              type="button"
              onClick={() => setAll((v) => !v)}
              className="block w-full px-2 py-1 text-left text-xs text-muted underline transition-colors hover:text-content"
            >
              {all ? 'Show fewer' : `Show all ${records.length}`}
            </button>
          )}
        </div>
      )}
    </div>
  );
}

function NavItem({ to, label, icon, end }: { to: string; label: string; icon: IconName; end: boolean }) {
  return (
    <NavLink
      to={to}
      end={end}
      className={({ isActive }) =>
        `flex items-center gap-2.5 rounded-button border-l-2 px-2 py-1.5 text-sm transition-colors ${
          isActive
            ? 'border-accent bg-surface-2 font-semibold text-content'
            : 'border-transparent text-content hover:bg-surface-2'
        }`
      }
    >
      <NavIcon name={icon} />
      {label}
    </NavLink>
  );
}
