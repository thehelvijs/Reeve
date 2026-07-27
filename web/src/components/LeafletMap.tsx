import { useEffect, useRef } from 'react';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';

export interface MapPoint {
  id?: string;
  lat: number;
  lon: number;
  label?: string;
}

// LeafletMap renders a dark-tiled slippy map (scroll to zoom). It fits the view
// to the markers so a single server shows its city rather than the whole globe.
// onPick fires with a clicked coordinate (location picker); onPointClick fires
// when a marker is clicked. Tiles come from CARTO (the one external dependency).
export default function LeafletMap({
  points = [],
  onPick,
  onPointClick,
  height = 460,
}: {
  points?: MapPoint[];
  onPick?: (lat: number, lon: number) => void;
  onPointClick?: (id: string) => void;
  height?: number | string;
}) {
  const elRef = useRef<HTMLDivElement>(null);
  const mapRef = useRef<L.Map | null>(null);
  const layerRef = useRef<L.LayerGroup | null>(null);
  const pickRef = useRef(onPick);
  pickRef.current = onPick;
  const pointClickRef = useRef(onPointClick);
  pointClickRef.current = onPointClick;

  useEffect(() => {
    if (!elRef.current || mapRef.current) {
      return;
    }
    const map = L.map(elRef.current, {
      scrollWheelZoom: true,
      worldCopyJump: true,
      minZoom: 2,
    }).setView([20, 0], 2);
    L.tileLayer('https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png', {
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
    const map = mapRef.current;
    const layer = layerRef.current;
    if (!map || !layer) {
      return;
    }
    layer.clearLayers();
    const coords: L.LatLngTuple[] = [];
    for (const p of points) {
      const icon = L.divIcon({ className: 'reeve-pin-wrap', html: '<span class="reeve-pin"></span>', iconSize: [16, 16], iconAnchor: [8, 8] });
      const m = L.marker([p.lat, p.lon], { icon }).addTo(layer);
      if (p.label) {
        m.bindTooltip(p.label, { permanent: true, direction: 'right', offset: [8, 0], className: 'reeve-tip' });
      }
      if (p.id && pointClickRef.current) {
        m.on('click', () => pointClickRef.current?.(p.id as string));
      }
      coords.push([p.lat, p.lon]);
    }
    if (coords.length === 1) {
      map.setView(coords[0], 11);
    } else if (coords.length > 1) {
      map.fitBounds(L.latLngBounds(coords).pad(0.3), { maxZoom: 12 });
    }
  }, [points]);

  return <div ref={elRef} style={{ height }} className="w-full overflow-hidden rounded-card border border-hairline" />;
}
