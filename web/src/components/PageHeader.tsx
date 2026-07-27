import type { ReactNode } from 'react';

// Every top-level page opens with this: title, optional subtitle, and one
// right-aligned primary action, closed by a rule that separates the heading
// from the page's content. Keeps headers identical across views.
export default function PageHeader({
  title,
  subtitle,
  action,
}: {
  title: string;
  subtitle?: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex items-start justify-between gap-4 border-b border-hairline pb-4">
      <div className="min-w-0">
        <h1 className="text-xl font-semibold text-content">{title}</h1>
        {subtitle && <p className="mt-0.5 text-sm text-muted">{subtitle}</p>}
      </div>
      {action && <div className="shrink-0">{action}</div>}
    </div>
  );
}
