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

// hoverReadout is the value under the cursor, for the charts uPlot's legend does
// not cover: a lone series carries no legend row, and even with one, reading a
// number off a table below the canvas is not what a reader points at.
//
// The element lives inside uPlot's own over-rect, so cursor.left/top are already
// the coordinates to place it at.
function hoverReadout(fmt?: (v: number) => string): uPlot.Plugin {
  const tip = document.createElement('div');
  tip.className = 'reeve-chart-tip';
  return {
    hooks: {
      init: (u: uPlot) => u.over.appendChild(tip),
      setCursor: (u: uPlot) => {
        const { idx, left, top } = u.cursor;
        if (idx == null || left == null || top == null || left < 0 || top < 0) {
          tip.style.display = 'none';
          return;
        }
        const at = u.data[0][idx];
        const lines = [new Date(at * 1000).toLocaleString()];
        for (let i = 1; i < u.series.length; i++) {
          const v = u.data[i]?.[idx];
          let shown = '—';
          if (v != null) {
            if (fmt) {
              shown = fmt(v);
            } else {
              shown = String(v);
            }
          }
          lines.push(`${u.series[i].label}: ${shown}`);
        }
        tip.textContent = lines.join('\n');
        tip.style.display = 'block';
        tip.style.left = `${left}px`;
        tip.style.top = `${top}px`;
        // Flip across the cursor rather than run off the right edge.
        if (left > u.over.clientWidth / 2) {
          tip.style.transform = 'translate(calc(-100% - 10px), -50%)';
        } else {
          tip.style.transform = 'translate(10px, -50%)';
        }
      },
    },
  };
}

// Chart renders a uPlot time-series sized to its container. uPlot is imperative,
// so the plot is built once and fed with setData afterwards.
//
// It used to be rebuilt on every render, which every poll causes: destroy()
// empties the container, the page loses the height of every chart on it at once,
// the browser clamps the scroll position to what is left, and the reader is
// thrown back to the top of the page 15 seconds after they scrolled down.
//
// A rebuild is still right when the *shape* changes — a series appearing, a
// theme swapping the palette — because those are baked into the options.
export default function Chart({
  xs,
  series,
  fmt,
  height = 160,
  windowSecs,
}: {
  xs: number[];
  series: Series[];
  fmt?: (v: number) => string;
  height?: number;
  // The selected range, in seconds. The axis spans it even when the data does
  // not, so switching range visibly changes the timeline; a host holding more
  // history than the window widens it rather than losing points off the left.
  windowSecs?: number;
}) {
  const ref = useRef<HTMLDivElement>(null);
  // uPlot paints to a canvas from color strings, so it cannot follow a variable
  // on its own: the theme is a dependency that forces a redraw.
  const theme = useTheme();
  const plotRef = useRef<uPlot | null>(null);
  // Read by the effects, so neither has to list the arrays every render creates
  // fresh, which is exactly the churn that used to rebuild the plot.
  const live = useRef({ xs, series });
  live.current = { xs, series };
  // The plot's shape: what the options are built from, as opposed to the numbers
  // fed in afterwards.
  const shape = series.map((s) => `${s.label}|${s.color}`).join('');

  useEffect(() => {
    const el = ref.current;
    if (!el) {
      return;
    }
    const { xs: x0, series: s0 } = live.current;
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
      legend: { show: s0.length > 1 },
      plugins: [hoverReadout(fmt)],
      scales: {
        x: {
          time: true,
          range: (_u, dataMin, dataMax) => {
            const now = Date.now() / 1000;
            let lo = dataMin;
            let hi = dataMax;
            if (!Number.isFinite(lo) || !Number.isFinite(hi)) {
              lo = now - (windowSecs ?? 3600);
              hi = now;
            }
            if (windowSecs) {
              return [Math.min(lo, now - windowSecs), Math.max(hi, now)];
            }
            return [lo, hi];
          },
        },
      },
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
        ...s0.map((s) => ({
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
    const plot = new uPlot(opts, [x0, ...s0.map((s) => s.data)], el);
    plotRef.current = plot;

    const onResize = () => plot.setSize({ width: el.clientWidth || width, height });
    window.addEventListener('resize', onResize);
    return () => {
      window.removeEventListener('resize', onResize);
      plotRef.current = null;
      plot.destroy();
    };
  }, [shape, fmt, height, theme, windowSecs]);

  // Every render after that is new numbers on the same plot: the canvas redraws
  // and the page keeps its height, so nothing moves under the reader.
  useEffect(() => {
    plotRef.current?.setData([xs, ...series.map((s) => s.data)]);
  }, [xs, series]);

  return <div ref={ref} className="w-full" />;
}
