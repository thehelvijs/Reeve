import { useEffect, useRef } from 'react';
import uPlot from 'uplot';
import 'uplot/dist/uPlot.min.css';

export interface Series {
  label: string;
  color: string;
  data: (number | null)[];
}

// Chart renders a uPlot time-series sized to its container. It recreates the
// plot when data or width changes; uPlot itself is imperative.
export default function Chart({
  xs,
  series,
  fmt,
  height = 160,
}: {
  xs: number[];
  series: Series[];
  fmt?: (v: number) => string;
  height?: number;
}) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) {
      return;
    }
    const width = el.clientWidth || 600;
    const opts: uPlot.Options = {
      width,
      height,
      cursor: { y: false, points: { show: true } },
      legend: { show: true },
      scales: { x: { time: true } },
      // Axis labels wear the muted text token; the grid stays recessive.
      axes: [
        {
          stroke: '#626168',
          grid: { stroke: 'rgba(31,30,36,0.08)', width: 1 },
          ticks: { stroke: 'rgba(31,30,36,0.16)' },
        },
        {
          stroke: '#626168',
          grid: { stroke: 'rgba(31,30,36,0.08)', width: 1 },
          ticks: { stroke: 'rgba(31,30,36,0.16)' },
          values: fmt ? (_u, splits) => splits.map((v) => fmt(v)) : undefined,
        },
      ],
      series: [
        {},
        ...series.map((s) => ({
          label: s.label,
          stroke: s.color,
          width: 2,
          points: { show: false },
          value: (_u: uPlot, v: number | null) => {
            if (v == null) {
              return '—';
            }
            if (fmt) {
              return fmt(v);
            }
            return String(v);
          },
        })),
      ],
    };
    const data: uPlot.AlignedData = [xs, ...series.map((s) => s.data)];
    const plot = new uPlot(opts, data, el);

    const onResize = () => plot.setSize({ width: el.clientWidth || width, height });
    window.addEventListener('resize', onResize);
    return () => {
      window.removeEventListener('resize', onResize);
      plot.destroy();
    };
  }, [xs, series, fmt, height]);

  return <div ref={ref} className="w-full" />;
}
