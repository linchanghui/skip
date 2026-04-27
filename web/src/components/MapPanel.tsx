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
  const infoWindowRef = useRef<google.maps.InfoWindow | null>(null);
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
        if (infoWindowRef.current) {
          infoWindowRef.current.close();
        }
        infoWindowRef.current = new google.maps.InfoWindow();

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
          const stale = isStaleStatus(s);
          const queueLabel = queueLabelText(s);
          const busyColor = markerColor(s.latest_status?.busy_level);
          const marker = new google.maps.Marker({
            map,
            position: { lat: s.location.lat, lng: s.location.lng },
            title: s.name,
            icon: {
              path: google.maps.SymbolPath.CIRCLE,
              scale: 11,
              fillColor: busyColor,
              fillOpacity: 0.95,
              strokeColor: "#ffffff",
              strokeWeight: 2,
            },
            label: {
              text: queueLabel,
              color: "#ffffff",
              fontSize: "11px",
              fontWeight: "700",
            },
          });
          marker.addListener("click", () => {
            onSelectStore(s.id);
            if (!infoWindowRef.current) {
              return;
            }
            infoWindowRef.current.setContent(
              buildInfoWindowContent(s, stale),
            );
            infoWindowRef.current.open({
              map,
              anchor: marker,
            });
          });
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
      if (infoWindowRef.current) {
        infoWindowRef.current.close();
      }
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

function markerColor(level: string | undefined): string {
  switch (level) {
    case "quiet":
      return "#16a34a";
    case "moderate":
      return "#d97706";
    case "busy":
      return "#dc2626";
    case "closed":
      return "#64748b";
    default:
      return "#2563eb";
  }
}

function queueLabelText(store: Store): string {
  if (store.latest_status?.queue_length == null) {
    return "--";
  }
  const n = store.latest_status.queue_length;
  if (n > 99) {
    return "99+";
  }
  return String(n);
}

function isStaleStatus(store: Store): boolean {
  const asOf = store.latest_status?.as_of;
  if (!asOf) {
    return true;
  }
  const t = new Date(asOf).getTime();
  if (!Number.isFinite(t)) {
    return true;
  }
  return Date.now() - t > 30 * 60 * 1000;
}

function busyLabel(level: string | undefined): string {
  switch (level) {
    case "quiet":
      return "Quiet";
    case "moderate":
      return "Moderate";
    case "busy":
      return "Busy";
    case "closed":
      return "Closed";
    default:
      return "Unknown";
  }
}

function escapeHtml(input: string): string {
  return input
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function buildInfoWindowContent(store: Store, stale: boolean): string {
  const name = escapeHtml(store.name);
  const queue =
    store.latest_status?.queue_length != null
      ? `${store.latest_status.queue_length}`
      : "N/A";
  const wait =
    store.latest_status?.wait_minutes_est != null
      ? `${store.latest_status.wait_minutes_est} min`
      : "N/A";
  const busy = busyLabel(store.latest_status?.busy_level);
  const staleText = stale ? " (stale)" : "";
  return `
    <div style="min-width:200px;line-height:1.4">
      <div style="font-weight:700;margin-bottom:4px">${name}</div>
      <div>Queue: <strong>${queue}</strong> · Wait: <strong>${wait}</strong></div>
      <div>Status: ${busy}${staleText}</div>
    </div>
  `;
}
