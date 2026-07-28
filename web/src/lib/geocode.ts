import { api } from '../api';
import citiesData from '../assets/cities.json';

export interface Place {
  label: string;
  lat: number;
  lon: number;
}

interface City {
  name: string;
  country: string;
  lat: number;
  lon: number;
}

const cities = citiesData as City[];

// locationKey is what "already saved" means to the picker: the trimmed place
// name and the pin. Autosave compares it against the last write.
export function locationKey(loc: string, lat: number | null, lon: number | null): string {
  return `${loc.trim()}|${lat ?? ''}|${lon ?? ''}`;
}

// cityMatches answers from the bundled city list, so the first keystrokes get a
// suggestion with no request and the field still works with no internet.
export function cityMatches(query: string): Place[] {
  const q = query.trim().toLowerCase();
  if (q.length < 2) {
    return [];
  }
  return cities
    .filter((c) => c.name.toLowerCase().startsWith(q))
    .slice(0, 3)
    .map((c) => ({ label: `${c.name}, ${c.country}`, lat: c.lat, lon: c.lon }));
}

// mergePlaces puts the offline cities first and the geocoded addresses after,
// dropping a label either side already offered.
export function mergePlaces(local: Place[], remote: Place[]): Place[] {
  const seen = new Set<string>();
  const out: Place[] = [];
  for (const p of [...local, ...remote]) {
    const key = p.label.toLowerCase();
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    out.push(p);
  }
  return out.slice(0, 8);
}

// lookupAddress asks the server's geocoder proxy for street-level matches. A
// failure is not an error the operator needs: the city list already answered.
export async function lookupAddress(query: string): Promise<Place[]> {
  if (query.trim().length < 3) {
    return [];
  }
  try {
    return await api.get<Place[]>(`/api/admin/geocode?q=${encodeURIComponent(query.trim())}`);
  } catch {
    return [];
  }
}
