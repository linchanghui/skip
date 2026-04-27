import type {
  Area,
  CreateQueueReportInput,
  QueueReport,
  QueueSignal,
  RunnerAvailability,
  Store,
  StoreDetail,
  Task,
  TaskListResponse,
} from "./types";

/** Align API root with Vite base (VITE_BASE_PATH) so production requests stay same-origin (e.g. /skip/v1/...). */
function apiBase(): string {
  const explicit = import.meta.env.VITE_API_BASE?.trim() ?? "";
  if (explicit) {
    return explicit.replace(/\/$/, "");
  }
  const b = import.meta.env.BASE_URL;
  if (b === "/") {
    return "";
  }
  return b.replace(/\/$/, "");
}

async function getJSON<T>(path: string): Promise<T> {
  const root = apiBase();
  const p = path.startsWith("/") ? path : `/${path}`;
  const url = root ? `${root}${p}` : p;
  const res = await fetch(url);
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`HTTP ${res.status}: ${text}`);
  }
  return (await res.json()) as T;
}

async function postJSON<T>(path: string, body: unknown): Promise<T> {
  const root = apiBase();
  const p = path.startsWith("/") ? path : `/${path}`;
  const url = root ? `${root}${p}` : p;
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`HTTP ${res.status}: ${text}`);
  }
  return (await res.json()) as T;
}

export function fetchAreaChangi(): Promise<Area> {
  return getJSON<Area>("/v1/areas/changi");
}

export function fetchStores(areaId: string): Promise<{ stores: Store[] }> {
  const q = new URLSearchParams({ area_id: areaId });
  return getJSON<{ stores: Store[] }>(`/v1/stores?${q.toString()}`);
}

export function fetchStoreDetail(
  id: string,
  historyLimit = 20,
): Promise<StoreDetail> {
  const q = new URLSearchParams({ history_limit: String(historyLimit) });
  return getJSON<StoreDetail>(`/v1/stores/${encodeURIComponent(id)}?${q}`);
}

export function fetchQueueSignal(id: string): Promise<QueueSignal> {
  return getJSON<QueueSignal>(`/v1/stores/${encodeURIComponent(id)}/queue-signal`);
}

export function createTask(input: {
  user_id: string;
  store_id: string;
  task_type: "queue_for_me";
  note?: string;
}): Promise<Task> {
  return postJSON<Task>("/v1/tasks", input);
}

export function acceptTask(input: {
  task_id: string;
  runner_id: string;
  note?: string;
}): Promise<Task> {
  return postJSON<Task>(`/v1/tasks/${encodeURIComponent(input.task_id)}/accept`, {
    runner_id: input.runner_id,
    note: input.note ?? "",
  });
}

export function listTasks(input: {
  status?: string[];
  runner_id?: string;
  limit?: number;
}): Promise<TaskListResponse> {
  const q = new URLSearchParams();
  if (input.status && input.status.length > 0) {
    q.set("status", input.status.join(","));
  }
  if (input.runner_id?.trim()) {
    q.set("runner_id", input.runner_id.trim());
  }
  if (input.limit && input.limit > 0) {
    q.set("limit", String(input.limit));
  }
  const suffix = q.toString();
  return getJSON<TaskListResponse>(suffix ? `/v1/tasks?${suffix}` : "/v1/tasks");
}

export function setRunnerAvailability(input: {
  runner_id: string;
  is_online: boolean;
}): Promise<RunnerAvailability> {
  return postJSON<RunnerAvailability>(
    `/v1/runners/${encodeURIComponent(input.runner_id)}/availability`,
    { is_online: input.is_online },
  );
}

export function createQueueReport(input: {
  store_id: string;
  payload: CreateQueueReportInput;
}): Promise<QueueReport> {
  return postJSON<QueueReport>(
    `/v1/stores/${encodeURIComponent(input.store_id)}/queue-reports`,
    input.payload,
  );
}
