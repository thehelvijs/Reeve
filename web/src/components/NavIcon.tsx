import type { ReactElement } from 'react';

export type IconName =
  | 'dashboard'
  | 'services'
  | 'collections'
  | 'hosts'
  | 'requests'
  | 'pipelines'
  | 'users'
  | 'groups'
  | 'webhooks'
  | 'alerts'
  | 'audit'
  | 'server'
  | 'settings';

// Compact 16px line icons for the sidebar nav.
const paths: Record<IconName, ReactElement> = {
  dashboard: (
    <>
      <rect x="2.5" y="2.5" width="4.5" height="4.5" rx="1" />
      <rect x="9" y="2.5" width="4.5" height="4.5" rx="1" />
      <rect x="2.5" y="9" width="4.5" height="4.5" rx="1" />
      <rect x="9" y="9" width="4.5" height="4.5" rx="1" />
    </>
  ),
  services: (
    <>
      <path d="M8 2 14 5l-6 3-6-3 6-3Z" />
      <path d="M2 8l6 3 6-3" />
      <path d="M2 11l6 3 6-3" />
    </>
  ),
  collections: (
    <>
      <path d="M8 2 14 5l-6 3-6-3 6-3Z" />
      <rect x="3.5" y="9.5" width="9" height="4" rx="1" />
    </>
  ),
  hosts: (
    <>
      <rect x="2.5" y="3" width="11" height="4" rx="1" />
      <rect x="2.5" y="9" width="11" height="4" rx="1" />
      <path d="M5 5h.01M5 11h.01" />
    </>
  ),
  requests: (
    <>
      <path d="M2.5 8h3l1 2h3l1-2h3" />
      <path d="M2.5 8 4 3.5h8L13.5 8v4a1 1 0 0 1-1 1h-9a1 1 0 0 1-1-1V8Z" />
    </>
  ),
  pipelines: (
    <>
      <circle cx="4" cy="4" r="1.75" />
      <circle cx="4" cy="12" r="1.75" />
      <circle cx="12" cy="8" r="1.75" />
      <path d="M5.75 4h2.75a1.5 1.5 0 0 1 1.5 1.5v.9M5.75 12H8.5a1.5 1.5 0 0 0 1.5-1.5v-.9" />
    </>
  ),
  users: (
    <>
      <circle cx="6" cy="5.5" r="2.2" />
      <path d="M2.5 13c0-2 1.6-3.3 3.5-3.3S9.5 11 9.5 13" />
      <path d="M10.5 4.2a2.2 2.2 0 0 1 0 4.1M11 13c0-1.6-.6-2.7-1.6-3.3" />
    </>
  ),
  groups: (
    <>
      <path d="M2.5 5.5 4 3.5h3l1.2 1.5h5.3v7.5a1 1 0 0 1-1 1h-10a1 1 0 0 1-1-1v-7Z" />
    </>
  ),
  webhooks: (
    <>
      <path d="M6 9 4.3 10.7a2.4 2.4 0 1 0 3.4 3.4L9 12.7" />
      <path d="M10 7l1.7-1.7a2.4 2.4 0 1 0-3.4-3.4L7 3.3" />
      <path d="M6.2 9.8 9.8 6.2" />
    </>
  ),
  alerts: (
    <>
      <path d="M4 7a4 4 0 0 1 8 0c0 3 1 4 1 4H3s1-1 1-4Z" />
      <path d="M6.5 13a1.5 1.5 0 0 0 3 0" />
    </>
  ),
  audit: (
    <>
      <rect x="3" y="2.5" width="10" height="11" rx="1" />
      <path d="M5.5 6h5M5.5 8.5h5M5.5 11h3" />
    </>
  ),
  settings: (
    <>
      <circle cx="8" cy="8" r="2" />
      <path d="M8 1.8v1.6M8 12.6v1.6M2.2 8h1.6M12.2 8h1.6M3.9 3.9l1.1 1.1M11 11l1.1 1.1M12.1 3.9 11 5M5 11l-1.1 1.1" />
    </>
  ),
  server: (
    <>
      <rect x="3" y="3" width="10" height="10" rx="2" />
      <rect x="5.5" y="5.5" width="5" height="5" rx="1" />
      <path d="M1.5 6h1.5M1.5 10h1.5M13 6h1.5M13 10h1.5M6 1.5V3M10 1.5V3M6 13v1.5M10 13v1.5" />
    </>
  ),
};

export default function NavIcon({ name }: { name: IconName }) {
  return (
    <svg
      className="h-4 w-4 shrink-0 [[aria-current=page]_&]:stroke-2"
      viewBox="0 0 16 16"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
    >
      {paths[name]}
    </svg>
  );
}
