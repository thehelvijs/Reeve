// Placeholder shapes shown only on a cold first load, before the cache warms.

export function Skeleton({ className = '' }: { className?: string }) {
  return <div className={`animate-pulse rounded-button bg-surface-2 ${className}`} />;
}

// A stack of card-shaped rows matching the list pages' row height.
export function ListSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div className="space-y-2" aria-hidden>
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="rounded-card border border-hairline bg-surface-1 px-4 py-3">
          <Skeleton className="h-4 w-40" />
          <Skeleton className="mt-2 h-3 w-64" />
        </div>
      ))}
    </div>
  );
}
