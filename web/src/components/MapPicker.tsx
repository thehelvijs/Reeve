import { useMemo, useState } from 'react';
import { api, ApiError } from '../api';
import { Button, ErrorText, Form, Input } from './ui';
import LeafletMap from './LeafletMap';
import citiesData from '../assets/cities.json';

interface City {
  name: string;
  country: string;
  lat: number;
  lon: number;
}
const cities = citiesData as City[];

// MapPicker edits a host's physical location: a text field with local
// place-name suggestions, plus a click-to-drop pin on the bundled world map.
// Fully local — no geocoding service is contacted.
export default function MapPicker({
  hostId,
  location,
  latitude,
  longitude,
  onSaved,
}: {
  hostId: string;
  location: string;
  latitude?: number;
  longitude?: number;
  onSaved: () => void;
}) {
  const [loc, setLoc] = useState(location);
  const [lat, setLat] = useState<number | null>(latitude ?? null);
  const [lon, setLon] = useState<number | null>(longitude ?? null);
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState('');

  const suggestions = useMemo(() => {
    const q = loc.trim().toLowerCase();
    if (q.length < 2) {
      return [];
    }
    return cities.filter((c) => c.name.toLowerCase().startsWith(q)).slice(0, 8);
  }, [loc]);

  const pick = (c: City) => {
    setLoc(`${c.name}, ${c.country}`);
    setLat(c.lat);
    setLon(c.lon);
    setOpen(false);
    setSaved(false);
  };

  const save = async () => {
    setError('');
    setBusy(true);
    try {
      await api.patch(`/api/v1/admin/hosts/${hostId}`, {
        physical_location: loc.trim(),
        latitude: lat,
        longitude: lon,
      });
      setSaved(true);
      onSaved();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'could not save');
    } finally {
      setBusy(false);
    }
  };

  const points = lat != null && lon != null ? [{ lat, lon, accent: true }] : [];

  return (
    <Form onSubmit={save} className="space-y-3">
      <div className="relative max-w-sm">
        <Input
          placeholder="Search a city, e.g. Riga"
          value={loc}
          onChange={(e) => {
            setLoc(e.target.value);
            setOpen(true);
            setSaved(false);
          }}
          onFocus={() => setOpen(true)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && open && suggestions.length > 0) {
              e.preventDefault();
              pick(suggestions[0]);
            }
          }}
        />
        {open && suggestions.length > 0 && (
          <div className="absolute z-20 mt-1 w-full overflow-hidden rounded-card border border-hairline bg-surface-3">
            {suggestions.map((c) => (
              <button
                key={`${c.name}-${c.lat}-${c.lon}`}
                type="button"
                onClick={() => pick(c)}
                className="block w-full px-3 py-1.5 text-left text-sm text-muted transition-colors hover:bg-surface-2 hover:text-content"
              >
                {c.name}, {c.country}
              </button>
            ))}
          </div>
        )}
      </div>

      <p className="text-xs text-muted">
        Or click the map to drop a pin.
        {lat != null && lon != null && (
          <>
            {' '}
            <span className="text-content">
              {lat.toFixed(3)}, {lon.toFixed(3)}
            </span>{' '}
            <button
              type="button"
              className="text-muted underline hover:text-content"
              onClick={() => {
                setLat(null);
                setLon(null);
                setSaved(false);
              }}
            >
              clear
            </button>
          </>
        )}
      </p>

      <LeafletMap
        points={points}
        height={360}
        onPick={(la, lo) => {
          setLat(la);
          setLon(lo);
          setSaved(false);
        }}
      />

      <div className="flex items-center gap-3">
        <Button type="submit" disabled={busy}>
          {busy ? 'Saving…' : 'Save location'}
        </Button>
        {saved && <span className="text-xs text-muted">Saved.</span>}
      </div>
      <ErrorText>{error}</ErrorText>
    </Form>
  );
}
