import { useEffect, useMemo, useRef, useState } from 'react';
import { api, ApiError } from '../api';
import { ErrorText, Form, Input } from './ui';
import LeafletMap from './LeafletMap';
import { cityMatches, locationKey, lookupAddress, mergePlaces, type Place } from '../lib/geocode';

const SAVE_DELAY = 800;
const SEARCH_DELAY = 250;

// MapPicker edits a host's physical location: a search field that suggests
// cities from the bundled list and addresses from the server's geocoder proxy,
// plus a click-to-drop pin on the map. There is no save button — an edit that
// settles is written on its own.
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
  // The map follows a place the operator chose by name, and holds still for a pin
  // they dropped by hand.
  const [follow, setFollow] = useState(true);
  const [hits, setHits] = useState<Place[]>([]);
  const [searching, setSearching] = useState(false);
  const [busy, setBusy] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState('');
  const savedKey = useRef(locationKey(location, latitude ?? null, longitude ?? null));

  const suggestions = useMemo(() => mergePlaces(cityMatches(loc), hits), [loc, hits]);

  const save = async () => {
    const key = locationKey(loc, lat, lon);
    setError('');
    setBusy(true);
    try {
      await api.patch(`/api/admin/hosts/${hostId}`, {
        physical_location: loc.trim(),
        latitude: lat,
        longitude: lon,
      });
      savedKey.current = key;
      setSaved(true);
      onSaved();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'could not save');
    } finally {
      setBusy(false);
    }
  };

  const dirty = locationKey(loc, lat, lon) !== savedKey.current;
  const saveRef = useRef(save);
  saveRef.current = save;

  // Autosave: an edit that stops changing for SAVE_DELAY is written. The typed
  // name is saved as typed, so a place the geocoder does not know still sticks.
  useEffect(() => {
    if (!dirty) {
      return;
    }
    setSaved(false);
    const t = setTimeout(() => saveRef.current(), SAVE_DELAY);
    return () => clearTimeout(t);
  }, [loc, lat, lon, dirty]);

  // Address typeahead, debounced so a held key is one request, not ten.
  useEffect(() => {
    const q = loc.trim();
    if (q.length < 3) {
      setHits([]);
      setSearching(false);
      return;
    }
    setSearching(true);
    let live = true;
    const t = setTimeout(async () => {
      const found = await lookupAddress(q);
      if (live) {
        setHits(found);
        setSearching(false);
      }
    }, SEARCH_DELAY);
    return () => {
      live = false;
      clearTimeout(t);
    };
  }, [loc]);

  const pick = (p: Place) => {
    setLoc(p.label);
    setLat(p.lat);
    setLon(p.lon);
    setFollow(true);
    setOpen(false);
  };

  // Memoised, or every unrelated re-render would count as a moved pin and the
  // map would jump back to its fitted view mid-edit.
  const points = useMemo(() => {
    if (lat == null || lon == null) {
      return [];
    }
    return [{ lat, lon }];
  }, [lat, lon]);

  let status = '';
  if (busy) {
    status = 'Saving…';
  } else if (dirty) {
    status = 'Unsaved…';
  } else if (saved) {
    status = 'Saved.';
  }

  return (
    <Form onSubmit={save} className="space-y-3">
      <div className="relative max-w-sm">
        <Input
          placeholder="Search a city or address, e.g. Brivibas iela 32, Riga"
          value={loc}
          onChange={(e) => {
            setLoc(e.target.value);
            setOpen(true);
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
            {suggestions.map((p) => (
              <button
                key={`${p.label}-${p.lat}-${p.lon}`}
                type="button"
                onClick={() => pick(p)}
                className="block w-full px-3 py-1.5 text-left text-sm text-muted transition-colors hover:bg-surface-2 hover:text-content"
              >
                {p.label}
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
        recenter={follow}
        bright
        onPick={(la, lo) => {
          setFollow(false);
          setLat(la);
          setLon(lo);
        }}
      />

      <div className="flex h-4 items-center gap-3 text-xs text-muted">
        <span>{status}</span>
        {searching && <span>Searching addresses…</span>}
      </div>
      <ErrorText>{error}</ErrorText>
    </Form>
  );
}
