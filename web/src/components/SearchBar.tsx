import { useEffect, useRef } from 'react';

const isMac =
  typeof navigator !== 'undefined' && /Mac|iPhone|iPad|iPod/.test(navigator.platform || navigator.userAgent);

// SearchBar is the shared header search: a leading glyph, a trailing shortcut
// hint, and a global Cmd/Ctrl+K that focuses it from anywhere. `shortcut` is off
// for the per-page filters, so one Cmd+K keeps meaning the global search.
export default function SearchBar({
  value,
  onChange,
  onSubmit,
  placeholder = 'Search services…',
  shortcut = true,
  className = 'w-full max-w-md',
}: {
  value: string;
  onChange: (v: string) => void;
  onSubmit?: () => void;
  placeholder?: string;
  shortcut?: boolean;
  className?: string;
}) {
  const ref = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!shortcut) {
      return;
    }
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        ref.current?.focus();
        ref.current?.select();
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [shortcut]);

  let padding = 'pr-3';
  if (shortcut) {
    padding = 'pr-14';
  }

  return (
    <div className={`relative ${className}`}>
      <svg
        className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted"
        viewBox="0 0 16 16"
        fill="none"
        aria-hidden
      >
        <circle cx="7" cy="7" r="4.5" stroke="currentColor" strokeWidth="1.5" />
        <path d="m11 11 3 3" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      </svg>
      <input
        ref={ref}
        type="search"
        value={value}
        placeholder={placeholder}
        aria-label={placeholder}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && onSubmit) {
            onSubmit();
          }
          if (e.key === 'Escape') {
            ref.current?.blur();
          }
        }}
        className={`w-full rounded-button border border-hairline-strong bg-canvas py-2 pl-9 ${padding} text-sm text-content placeholder:text-muted focus:outline-none focus-visible:ring-2 focus-visible:ring-link [&::-webkit-search-cancel-button]:hidden`}
      />
      {shortcut && (
        <kbd className="pointer-events-none absolute right-2.5 top-1/2 -translate-y-1/2 rounded-button border border-hairline bg-surface-2 px-1.5 py-0.5 text-[10px] font-medium text-muted">
          {isMac ? '⌘K' : 'Ctrl K'}
        </kbd>
      )}
    </div>
  );
}
