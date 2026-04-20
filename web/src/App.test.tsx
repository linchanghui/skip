import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import App from "./App";

const area = {
  id: "changi",
  name: "Singapore Changi Airport (demo)",
  center: { lat: 1.3596, lng: 103.9891 },
  default_zoom: 14,
  bbox: { south: 1.342, west: 103.975, north: 1.375, east: 104.005 },
};

const stores = {
  stores: [
    {
      id: "sb-jewel",
      name: "Starbucks (Jewel)",
      area_id: "changi",
      terminal: "Jewel",
      floor: "L1",
      category: "coffee",
      location: { lat: 1.36137, lng: 103.98915 },
      external_ref: "seed",
      latest_status: {
        busy_level: "moderate",
        queue_length: 3,
        wait_minutes_est: 10,
        as_of: "2026-04-20T10:00:00Z",
        source: "operator",
      },
    },
  ],
};

const detail = {
  ...stores.stores[0],
  status_history: [
    {
      id: 1,
      store_id: "sb-jewel",
      busy_level: "moderate",
      queue_length: 3,
      wait_minutes_est: 10,
      source: "operator",
      reporter_id: null,
      reported_at: "2026-04-20T10:00:00Z",
      note: null,
    },
  ],
};

describe("App", () => {
  beforeEach(() => {
    vi.stubEnv("VITE_GOOGLE_MAPS_API_KEY", "");
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = typeof input === "string" ? input : input.toString();
        if (url.includes("/v1/areas/changi")) {
          return new Response(JSON.stringify(area), { status: 200 });
        }
        if (url.includes("/v1/stores?") && url.includes("area_id=changi")) {
          return new Response(JSON.stringify(stores), { status: 200 });
        }
        if (url.includes("/v1/stores/sb-jewel")) {
          return new Response(JSON.stringify(detail), { status: 200 });
        }
        return new Response("not found", { status: 404 });
      }),
    );
  });

  it("在没有 Maps Key 时显示占位说明并渲染门店名", async () => {
    render(<App />);
    expect(
      await screen.findByText(/VITE_GOOGLE_MAPS_API_KEY/i),
    ).toBeInTheDocument();
    expect(await screen.findByText("Starbucks (Jewel)")).toBeInTheDocument();
  });
});
