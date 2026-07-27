import { Link } from 'react-router-dom';
import type { ReactNode } from 'react';

// Consistent back affordance for detail pages.
export default function BackLink({ to, children }: { to: string; children: ReactNode }) {
  return (
    <Link to={to} className="text-sm text-link transition-colors hover:text-link-hover">
      ← {children}
    </Link>
  );
}
