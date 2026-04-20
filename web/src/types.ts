export type BusyLevel = "quiet" | "moderate" | "busy" | "closed";

export type StatusSource = "merchant" | "operator" | "integration" | "crowd";

export interface LatLng {
  lat: number;
  lng: number;
}

export interface BBox {
  south: number;
  west: number;
  north: number;
  east: number;
}

export interface Area {
  id: string;
  name: string;
  center: LatLng;
  default_zoom: number;
  bbox: BBox;
}

export interface LatestStatus {
  busy_level: BusyLevel;
  queue_length?: number | null;
  wait_minutes_est?: number | null;
  as_of: string;
  source: StatusSource;
}

export interface Store {
  id: string;
  name: string;
  area_id: string;
  terminal?: string | null;
  floor?: string | null;
  category: string;
  location: LatLng;
  external_ref?: string | null;
  latest_status?: LatestStatus | null;
}

export interface StatusReport {
  id: number;
  store_id: string;
  busy_level: BusyLevel;
  queue_length?: number | null;
  wait_minutes_est?: number | null;
  source: StatusSource;
  reporter_id?: string | null;
  reported_at: string;
  note?: string | null;
}

export interface StoreDetail extends Store {
  status_history: StatusReport[];
}
