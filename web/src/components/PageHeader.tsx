import type { ReactNode } from 'react';
import SearchBar from './SearchBar';

// Every top-level page opens with this: title, optional subtitle, an optional
// search that filters the page's own list, and one right-aligned primary action,
// closed by a rule that separates the heading from the page's content. Keeps
// headers identical across views.
export default function PageHeader({
  title,
  subtitle,
  search,
  action,
}: {
  title: string;
  subtitle?: string;
  search?: { value: string; onChange: (v: string) => void; placeholder: string };
  action?: ReactNode;
}) {
  return (
    <div className="flex flex-wrap items-start justify-between gap-4 border-b border-hairline pb-4">
      <div className="min-w-0">
        <h1 className="text-xl font-semibold text-content">{title}</h1>
        {subtitle && <p className="mt-0.5 text-sm text-muted">{subtitle}</p>}
      </div>
      {(search || action) && (
        <div className="flex shrink-0 items-center gap-2">
          {search && (
            <SearchBar
              value={search.value}
              onChange={search.onChange}
              placeholder={search.placeholder}
              shortcut={false}
              className="w-56"
            />
          )}
          {action}
        </div>
      )}
    </div>
  );
}
