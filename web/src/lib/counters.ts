// perSecond turns a counter into the rate an operator actually reads. The agent
// reports network and disk traffic as totals since boot, so charting them raw
// draws a near-flat line whose whole height is history the host cannot undo.
//
// The first point has no predecessor and a counter that went backwards (a
// reboot, a nic reset) yields no point rather than a negative spike. Both are
// null, which Chart draws as a gap. Over a rolled-up range each value is the
// bucket average, so the delta between two buckets is the average rate across
// them, the same reading at a coarser grain.
export function perSecond(values: number[], xs: number[]): (number | null)[] {
  return values.map((v, i) => {
    if (i === 0) {
      return null;
    }
    const seconds = xs[i] - xs[i - 1];
    const delta = v - values[i - 1];
    if (seconds <= 0 || delta < 0) {
      return null;
    }
    return delta / seconds;
  });
}
