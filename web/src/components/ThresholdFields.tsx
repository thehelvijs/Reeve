import { type ThresholdMetric } from '../api';
import { Field, Input } from './ui';

export interface ThresholdRow {
  enabled: boolean;
  threshold: number | '';
}

export type ThresholdRows = Partial<Record<ThresholdMetric, ThresholdRow>>;

const METRICS: { key: ThresholdMetric; label: string }[] = [
  { key: 'cpu', label: 'CPU %' },
  { key: 'mem', label: 'Memory %' },
  { key: 'disk', label: 'Disk %' },
  { key: 'temp', label: 'Temperature °C' },
  { key: 'load', label: 'Load (1m)' },
  { key: 'net', label: 'Network MiB/s' },
];

export const METRIC_KEYS: ThresholdMetric[] = METRICS.map((m) => m.key);

export const EMPTY_ROW: ThresholdRow = { enabled: false, threshold: '' };

// net thresholds live in MiB/s in the UI but bytes/s on the wire.
const NET_UNIT_BYTES = 1048576;

export function thresholdToDisplay(key: ThresholdMetric, wire: number): number {
  if (key === 'net') {
    return wire / NET_UNIT_BYTES;
  }
  return wire;
}

export function thresholdToWire(key: ThresholdMetric, display: number): number {
  if (key === 'net') {
    return display * NET_UNIT_BYTES;
  }
  return display;
}

export default function ThresholdFields({
  value,
  onChange,
}: {
  value: ThresholdRows;
  onChange: (metric: ThresholdMetric, row: ThresholdRow) => void;
}) {
  return (
    <div className="grid grid-cols-2 gap-4">
      {METRICS.map((m) => {
        const row = value[m.key] ?? EMPTY_ROW;
        return (
          <div key={m.key} className="flex items-end gap-3">
            <label className="flex items-center gap-2 text-sm text-content">
              <input
                type="checkbox"
                checked={row.enabled}
                onChange={(e) => onChange(m.key, { ...row, enabled: e.target.checked })}
                className="h-4 w-4 rounded border-hairline accent-accent"
              />
              {m.label}
            </label>
            <div className="flex-1">
              <Field label="Threshold">
                <Input
                  type="number"
                  value={row.threshold}
                  onChange={(e) =>
                    onChange(m.key, {
                      ...row,
                      threshold: e.target.value === '' ? '' : Number(e.target.value),
                    })
                  }
                />
              </Field>
            </div>
          </div>
        );
      })}
    </div>
  );
}
