// A single labelled resource bar: grey track, blue fill, red past 90%.
export default function MetricBar({ label, pct, detail }: { label: string; pct: number; detail?: string }) {
  const clamped = Math.max(0, Math.min(100, pct));
  const danger = clamped >= 90;
  return (
    <div>
      <div className="flex items-center justify-between text-xs">
        <span className="text-muted">{label}</span>
        <span className="text-content">{detail ?? `${clamped.toFixed(0)}%`}</span>
      </div>
      <div className="mt-1 h-1.5 w-full overflow-hidden rounded-pill bg-surface-2">
        <div
          className={`h-full rounded-pill ${danger ? 'bg-down-solid' : 'bg-link'}`}
          style={{ width: `${clamped}%` }}
        />
      </div>
    </div>
  );
}
