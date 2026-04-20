import type { Area, Store, StoreDetail } from "./types";

/** 与 Vite base（VITE_BASE_PATH）对齐，使生产环境请求落在 /skip/v1/... 等同源路径。 */
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
