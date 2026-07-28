import { useEffect, useRef } from 'react';
import uPlot from 'uplot';
import 'uplot/dist/uPlot.min.css';
import { cssVar, useTheme } from '../lib/theme';

export interface Series {
  label: string;
  color: string;
  data: (number | null)[];
}

// autoAxisSize widens the y gutter to fit its widest label. uPlot reserves a
// fixed 50px, which silently clips the left of anything longer: "231KB/s" drew
// as ")1KB/s", and a big enough disk would lose a digit the same way.
//
// uPlot calls this repeatedly until the size settles, so it has to stop
// converging or it would loop.
const autoAxisSize: uPlot.Axis['size'] = (self, values, axisIdx, cycleNum) => {
  const ax = self.axes[axisIdx] as uPlot.Axis & { _size?: number; font?: [string, number] };
  if (cycleNum > 2) {
    return ax._size ?? 50;
  }
  const ticks = typeof ax.ticks?.size === 'number' ? ax.ticks.size : 10;
  const gap = typeof ax.gap === 'number' ? ax.gap : 5;
  const longest = (values ?? []).reduce((acc, v) => (v.length > acc.length ? v : acc), '');
  if (longest === '') {
    return ticks + gap;
  }
  if (ax.font) {
    self.ctx.font = ax.font[0];
  }
  return Math.ceil(ticks + gap + self.ctx.measureText(longest).width / devicePixelRatio);
};

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
  // uPlot paints to a canvas from color strings, so it cannot follow a variable
  // on its own: the theme is a dependency that forces a redraw.
  const theme = useTheme();

  useEffect(() => {
    const el = ref.current;
    if (!el) {
      return;
    }
    const axis = cssVar('--axis');
    const grid = cssVar('--grid');
    const tick = cssVar('--tick');
    const width = el.clientWidth || 600;
    const opts: uPlot.Options = {
      width,
      height,
      cursor: { y: false, points: { show: true } },
      // uPlot's legend names each line and reads out the value under the cursor,
      // which is worth a row of chrome only when there is more than one line. On
      // a lone series it sat under every chart showing "Time: -- CPU %: —", and
      // the card header carries that number now.
      legend: { show: series.length > 1 },
      scales: { x: { time: true } },
      // Axis labels wear the muted text token; the grid stays recessive.
      axes: [
        {
          stroke: axis,
          grid: { stroke: grid, width: 1 },
          ticks: { stroke: tick },
        },
        {
          stroke: axis,
          grid: { stroke: grid, width: 1 },
          ticks: { stroke: tick },
          values: fmt ? (_u, splits) => splits.map((v) => fmt(v)) : undefined,
          size: autoAxisSize,
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
  }, [xs, series, fmt, height, theme]);

  return <div ref={ref} className="w-full" />;
}
