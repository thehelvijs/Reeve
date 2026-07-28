import { useEffect, useRef } from 'react';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import { useTheme, type Theme } from '../lib/theme';
import { pinSVG } from '../lib/pin';

export interface MapPoint {
  id?: string;
  lat: number;
  lon: number;
  label?: string;
  // #rrggbb for this point's pin; anything else draws the brand accent.
  color?: string;
}

// CARTO ships a raster per theme, so the basemap is swapped, not recoloured; the
// dark one is brightened in index.css to match the canvas it sits on.
function tileURL(theme: Theme): string {
  if (theme === 'light') {
    return 'https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png';
  }
  return 'https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png';
}

// LeafletMap renders a slippy map (scroll to zoom) tiled for the active theme. It
// fits the view to the markers so a single server shows its city rather than the
// whole globe. onPick fires with a clicked coordinate (location picker);
// onPointClick fires when a marker is clicked. Tiles come from CARTO (the one
// external dependency).
//
// recenter=false leaves the view alone when the markers change, which is what a
// picker wants for a pin the operator just dropped: the pin moves, the map does
// not.
export default function LeafletMap({
  points = [],
  onPick,
  onPointClick,
  height = 460,
  recenter = true,
}: {
  points?: MapPoint[];
  onPick?: (lat: number, lon: number) => void;
  onPointClick?: (id: string) => void;
  height?: number | string;
  recenter?: boolean;
}) {
  const elRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<L.Map | null>(null);
  const layerRef = useRef<L.LayerGroup | null>(null);
  const tilesRef = useRef<L.TileLayer | null>(null);
  const theme = useTheme();
  // The map is built once, so the setup effect reads the theme through a ref
  // rather than listing it as a dependency and tearing the map down on a switch.
  const themeRef = useRef(theme);
  themeRef.current = theme;
  const pickRef = useRef(onPick);
  pickRef.current = onPick;
  const pointClickRef = useRef(onPointClick);
  pointClickRef.current = onPointClick;
  // Read through a ref, so turning recentering back on does not by itself move a
  // view the operator has since panned.
  const recenterRef = useRef(recenter);
  recenterRef.current = recenter;

  useEffect(() => {
    if (!elRef.current || mapRef.current) {
      return;
    }
    const map = L.map(elRef.current, {
      scrollWheelZoom: true,
      worldCopyJump: true,
      minZoom: 2,
    }).setView([20, 0], 2);
    tilesRef.current = L.tileLayer(tileURL(themeRef.current), {
      subdomains: 'abcd',
      maxZoom: 19,
      detectRetina: true,
      attribution: '&copy; OpenStreetMap &copy; CARTO',
    }).addTo(map);
    layerRef.current = L.layerGroup().addTo(map);
    map.on('click', (e: L.LeafletMouseEvent) => {
      if (pickRef.current) {
        pickRef.current(Number(e.latlng.lat.toFixed(4)), Number(e.latlng.lng.toFixed(4)));
      }
    });
    mapRef.current = map;
    setTimeout(() => map.invalidateSize(), 0);
    const ro = new ResizeObserver(() => map.invalidateSize());
    ro.observe(elRef.current);
    return () => {
      ro.disconnect();
      map.remove();
      mapRef.current = null;
    };
  }, []);

  useEffect(() => {
    tilesRef.current?.setUrl(tileURL(theme));
  }, [theme]);

  useEffect(() => {
    const map = mapRef.current;
    const layer = layerRef.current;
    if (!map || !layer) {
      return;
    }
    layer.clearLayers();
    const coords: L.LatLngTuple[] = [];
    for (const p of points) {
      const icon = L.divIcon({
        className: 'reeve-pin-wrap',
        html: pinSVG(p.color),
        iconSize: [24, 32],
        // The tip is the coordinate, so the pin sits above the point it marks.
        iconAnchor: [12, 32],
      });
      const m = L.marker([p.lat, p.lon], { icon }).addTo(layer);
      if (p.label) {
        m.bindTooltip(p.label, { permanent: true, direction: 'right', offset: [14, -12], className: 'reeve-tip' });
      }
      if (p.id && pointClickRef.current) {
        m.on('click', () => pointClickRef.current?.(p.id as string));
      }
      coords.push([p.lat, p.lon]);
    }
    if (!recenterRef.current) {
      return;
    }
    if (coords.length === 1) {
      map.setView(coords[0], 11);
    } else if (coords.length > 1) {
      map.fitBounds(L.latLngBounds(coords).pad(0.3), { maxZoom: 12 });
    }
  }, [points]);

  return <div ref={elRef} style={{ height }} className="w-full overflow-hidden rounded-card border border-hairline" />;
}
