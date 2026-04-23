import { useEffect, useRef, useState } from "react";
import { Loader } from "@googlemaps/js-api-loader";
import type { Area, Store } from "../types";

type Props = {
  area: Area | null;
  stores: Store[];
  selectedId: string | null;
  onSelectStore: (id: string) => void;
};

export function MapPanel({
  area,
  stores,
  selectedId,
  onSelectStore,
}: Props): JSX.Element {
  const mapEl = useRef<HTMLDivElement | null>(null);
  const mapRef = useRef<google.maps.Map | null>(null);
  const markersRef = useRef<google.maps.Marker[]>([]);
  const [mapReady, setMapReady] = useState(false);
  const apiKey = (import.meta.env.VITE_GOOGLE_MAPS_API_KEY ?? "").trim();

  useEffect(() => {
    if (!apiKey || !area || !mapEl.current) {
      setMapReady(false);
      mapRef.current = null;
      return;
    }

    const el = mapEl.current;
    let cancelled = false;

    const loader = new Loader({
      apiKey,
      version: "weekly",
    });

    loader
      .load()
      .then(() => {
        if (cancelled || !el) {
          return;
        }
        markersRef.current.forEach((m) => m.setMap(null));
        markersRef.current = [];

        const map = new google.maps.Map(el, {
          center: {
            lat: area.center.lat,
            lng: area.center.lng,
          },
          zoom: area.default_zoom,
          mapTypeControl: false,
          streetViewControl: false,
          fullscreenControl: true,
        });
        mapRef.current = map;

        for (const s of stores) {
          const marker = new google.maps.Marker({
            map,
            position: { lat: s.location.lat, lng: s.location.lng },
            title: s.name,
          });
          marker.addListener("click", () => onSelectStore(s.id));
          markersRef.current.push(marker);
        }
        if (!cancelled) {
          setMapReady(true);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setMapReady(false);
        }
      });

    return () => {
      cancelled = true;
      markersRef.current.forEach((m) => m.setMap(null));
      markersRef.current = [];
      mapRef.current = null;
      setMapReady(false);
    };
  }, [apiKey, area, stores, onSelectStore]);

  useEffect(() => {
    if (!apiKey || !mapReady || !mapRef.current || !selectedId) {
      return;
    }
    const s = stores.find((x) => x.id === selectedId);
    if (!s) {
      return;
    }
    mapRef.current.panTo({ lat: s.location.lat, lng: s.location.lng });
  }, [apiKey, mapReady, selectedId, stores]);

  if (!apiKey) {
    return (
      <div className="map-panel map-panel--placeholder">
        <p className="map-panel__title">Map (Maps JavaScript API)</p>
        <p>
          <code>VITE_GOOGLE_MAPS_API_KEY</code> is not configured. The list and
          details still work. Set the key and restart <code>npm run dev</code>{" "}
          to render Google Maps markers.
        </p>
        <p className="map-panel__hint">
          See the design doc section "5.1 Google Cloud & API Key" for setup.
        </p>
      </div>
    );
  }

  if (!area) {
    return <div className="map-panel map-panel--placeholder">Loading area data...</div>;
  }

  return (
    <div
      ref={mapEl}
      className="map-panel map-panel--live"
      role="application"
      aria-label="Google Map"
    />
  );
}
