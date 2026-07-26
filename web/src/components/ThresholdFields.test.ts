import { describe, expect, it } from 'vitest';
import { METRIC_KEYS, thresholdToDisplay, thresholdToWire } from './ThresholdFields';

const MIB = 1048576;

describe('threshold unit conversion', () => {
  // The net threshold is the only one where the UI unit differs from the wire
  // unit, so a mix-up here silently arms alerting a million times too low.
  it('converts net between MiB/s on screen and bytes/s on the wire', () => {
    expect(thresholdToDisplay('net', 5 * MIB)).toBe(5);
    expect(thresholdToWire('net', 5)).toBe(5 * MIB);
  });

  it('leaves every other metric untouched', () => {
    for (const key of METRIC_KEYS.filter((k) => k !== 'net')) {
      expect(thresholdToDisplay(key, 90)).toBe(90);
      expect(thresholdToWire(key, 90)).toBe(90);
    }
  });

  it('round-trips every metric back to the value it started at', () => {
    for (const key of METRIC_KEYS) {
      expect(thresholdToDisplay(key, thresholdToWire(key, 12.5))).toBe(12.5);
    }
  });

  it('covers every metric the form renders', () => {
    expect(METRIC_KEYS).toEqual(['cpu', 'mem', 'disk', 'temp', 'load', 'net']);
  });
});
