import type { ReactNode } from 'react';
import SearchBar from './SearchBar';
import Wordmark from './Wordmark';

// AppHeader is the single header used across the app and the public portal, so
// height and chrome stay identical. `nav` and `right` are page-specific slots;
// the centered search is shared.
export default function AppHeader({
  search,
  onSearch,
  onSubmitSearch,
  nav,
  right,
}: {
  search: string;
  onSearch: (v: string) => void;
  onSubmitSearch?: () => void;
  nav?: ReactNode;
  right?: ReactNode;
}) {
  return (
    <header className="relative flex shrink-0 items-center gap-4 border-b border-hairline bg-surface-1 px-8 py-3">
      <Wordmark />
      {nav}
      <div className="pointer-events-none absolute inset-x-0 hidden justify-center lg:flex">
        <div className="pointer-events-auto w-full max-w-sm">
          <SearchBar value={search} onChange={onSearch} onSubmit={onSubmitSearch} />
        </div>
      </div>
      <div className="ml-auto flex shrink-0 items-center gap-3">{right}</div>
    </header>
  );
}
