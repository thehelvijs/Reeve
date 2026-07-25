import type { ReactNode } from 'react';

// A quiet, on-brand empty state: a dashed hairline panel that reads as "nothing
// here yet" and points at the next action, rather than a bare line of text.
export default function EmptyState({
  title,
  description,
  action,
}: {
  title: string;
  description?: string;
  action?: ReactNode;
}) {
  return (
    <div className="rounded-card border border-dashed border-hairline px-6 py-12 text-center">
      <p className="text-sm font-medium text-content">{title}</p>
      {description && <p className="mx-auto mt-1 max-w-sm text-sm text-muted">{description}</p>}
      {action && <div className="mt-4 flex justify-center">{action}</div>}
    </div>
  );
}
