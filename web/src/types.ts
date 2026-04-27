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

export interface QueueSignalValue {
  queue_length?: number | null;
  wait_minutes_est?: number | null;
  busy_level: BusyLevel;
  confidence_flag: "normal" | "low";
  reporter_type: "runner" | "user" | "operator";
  reporter_id?: string | null;
  expires_at: string;
}

export interface QueueReport extends QueueSignalValue {
  id: number;
  store_id: string;
  reported_at: string;
  created_at: string;
}

export interface QueueSignal {
  store_id: string;
  status_expired: boolean;
  last_updated_at: string;
  last_updated_x_mins_ago: number;
  signal?: QueueSignalValue | null;
}

export type TaskStatus =
  | "created"
  | "matching"
  | "accepted"
  | "arrived"
  | "queuing"
  | "completed"
  | "failed"
  | "cancelled";

export interface Task {
  id: string;
  user_id: string;
  store_id: string;
  task_type: "queue_for_me";
  status: TaskStatus;
  accepted_runner_id?: string | null;
  requested_at: string;
  sla_accept_by: string;
  sla_arrive_by?: string | null;
}

export interface TaskListResponse {
  tasks: Task[];
}

export interface RunnerAvailability {
  runner_id: string;
  is_online: boolean;
  location?: LatLng | null;
  active_task_id?: string | null;
  last_ping_at: string;
  updated_at: string;
}

export interface CreateQueueReportInput {
  reporter_type: "runner" | "user" | "operator";
  reporter_id?: string;
  queue_length?: number;
  wait_minutes_est?: number;
  busy_level: BusyLevel;
  ttl_minutes?: number;
  evidence_url?: string;
}
