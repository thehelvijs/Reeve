import { describe, expect, it } from 'vitest';
import { cityMatches, locationKey, mergePlaces, type Place } from './geocode';

const riga: Place = { label: 'Riga, Latvia', lat: 56.95, lon: 24.11 };
const street: Place = { label: 'Brivibas iela 32, Riga, Latvia', lat: 56.96, lon: 24.12 };

describe('locationKey', () => {
  // Autosave fires on a changed key, so whitespace must not count as an edit.
  it('ignores surrounding whitespace', () => {
    expect(locationKey(' Riga ', 1, 2)).toBe(locationKey('Riga', 1, 2));
  });

  it('separates a cleared pin from a zero pin', () => {
    expect(locationKey('Riga', null, null)).not.toBe(locationKey('Riga', 0, 0));
  });

  it('changes when only the pin moves', () => {
    expect(locationKey('Riga', 1, 2)).not.toBe(locationKey('Riga', 1, 3));
  });

  it('changes when only the pin colour changes', () => {
    expect(locationKey('Riga', 1, 2, '#ff8800')).not.toBe(locationKey('Riga', 1, 2));
    expect(locationKey('Riga', 1, 2, '')).toBe(locationKey('Riga', 1, 2));
  });
});

describe('cityMatches', () => {
  it('stays quiet under two characters', () => {
    expect(cityMatches('r')).toEqual([]);
    expect(cityMatches(' ')).toEqual([]);
  });

  it('matches a city prefix regardless of case and carries its pin', () => {
    const hits = cityMatches('rig');
    expect(hits.length).toBeGreaterThan(0);
    expect(hits[0].label.startsWith('Riga')).toBe(true);
    expect(Number.isFinite(hits[0].lat) && Number.isFinite(hits[0].lon)).toBe(true);
  });

  it('offers at most three, leaving room for addresses', () => {
    expect(cityMatches('a').length).toBeLessThanOrEqual(3);
    expect(cityMatches('san').length).toBeLessThanOrEqual(3);
  });
});

describe('mergePlaces', () => {
  it('keeps the local hit first and drops the geocoder duplicate', () => {
    const merged = mergePlaces([riga], [{ ...riga, lat: 0, lon: 0 }, street]);
    expect(merged.map((p) => p.label)).toEqual([riga.label, street.label]);
    expect(merged[0].lat).toBe(riga.lat);
  });

  it('is case-insensitive about duplicates', () => {
    const merged = mergePlaces([riga], [{ ...riga, label: 'riga, latvia' }]);
    expect(merged).toHaveLength(1);
  });

  it('caps the list at eight', () => {
    const many = Array.from({ length: 20 }, (_, i) => ({ label: `p${i}`, lat: i, lon: i }));
    expect(mergePlaces([], many)).toHaveLength(8);
  });
});
