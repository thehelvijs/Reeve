import type { ReactNode } from 'react';
import SearchBar from './SearchBar';

// Every page opens with this: a title with the record's icon and current state
// beside it, an optional subtitle, an optional search that filters the page's own
// list, and one right-aligned primary action, closed by a rule that separates the
// heading from the page's content. Navigation is the sidebar's job; a detail page
// carries no trail back to the list it came from.
//
// A detail page takes the same header as a list page rather than growing its own.
// The two had drifted: a detail page ran a larger, tighter title with no rule
// under it, which reads as a different product one click in.
export default function PageHeader({
  title,
  subtitle,
  icon,
  badges,
  meta,
  search,
  action,
}: {
  title: string;
  subtitle?: string;
  icon?: ReactNode;
  badges?: ReactNode;
  meta?: ReactNode;
  search?: { value: string; onChange: (v: string) => void; placeholder: string };
  action?: ReactNode;
}) {
  return (
    <div className="border-b border-hairline pb-4">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex min-w-0 items-start gap-3">
          {icon}
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="text-xl font-semibold text-content">{title}</h1>
              {badges}
            </div>
            {subtitle && <p className="mt-0.5 text-sm text-muted">{subtitle}</p>}
            {meta}
          </div>
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
    </div>
  );
}
