import { useEffect, useState } from 'react';
import { NavLink, Outlet, useNavigate } from 'react-router-dom';
import { api, type Host, type Tool } from '../api';
import { useAuth } from '../auth';
import { Button } from './ui';
import Avatar from './Avatar';
import SearchBar from './SearchBar';
import NavIcon, { type IconName } from './NavIcon';
import Wordmark from './Wordmark';
import { prefetch } from '../lib/cache';
import { SOURCE_URL } from '../version';

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

  useEffect(() => {
    prefetch('/api/tools', () => api.get<Tool[]>('/api/tools'));
    prefetch('/api/hosts', () => api.get<Host[]>('/api/hosts'));
  }, []);

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
            <NavItem key={item.to} {...item} />
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
          <Button variant="secondary" className="mt-2 w-full" onClick={() => logout()}>
            Sign out
          </Button>
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
