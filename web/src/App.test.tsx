import { fireEvent, render, screen } from "@testing-library/react";
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

const queueSignalExpired = {
  store_id: "sb-jewel",
  status_expired: true,
  last_updated_at: "2026-04-20T09:00:00Z",
  last_updated_x_mins_ago: 80,
  signal: null,
};

describe("App", () => {
  beforeEach(() => {
    vi.stubEnv("VITE_GOOGLE_MAPS_API_KEY", "");
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = typeof input === "string" ? input : input.toString();
        if (url.endsWith("/v1/tasks") && init?.method === "POST") {
          return new Response(
            JSON.stringify({
              id: "task-new-1",
              user_id: "user-web",
              store_id: "sb-jewel",
              task_type: "queue_for_me",
              status: "matching",
              requested_at: "2026-04-20T10:02:00Z",
              sla_accept_by: "2026-04-20T10:07:00Z",
            }),
            { status: 201 },
          );
        }
        if (url.includes("/v1/tasks/task-new-1/accept") && init?.method === "POST") {
          return new Response(
            JSON.stringify({
              id: "task-new-1",
              user_id: "user-web",
              store_id: "sb-jewel",
              task_type: "queue_for_me",
              status: "accepted",
              accepted_runner_id: "runner-alex",
              requested_at: "2026-04-20T10:02:00Z",
              sla_accept_by: "2026-04-20T10:07:00Z",
            }),
            { status: 200 },
          );
        }
        if (url.includes("/v1/areas/changi")) {
          return new Response(JSON.stringify(area), { status: 200 });
        }
        if (url.includes("/v1/stores?") && url.includes("area_id=changi")) {
          return new Response(JSON.stringify(stores), { status: 200 });
        }
        if (url.includes("/v1/stores/sb-jewel/queue-signal")) {
          return new Response(JSON.stringify(queueSignalExpired), { status: 200 });
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

  it("显示过期 queue signal 提示", async () => {
    render(<App />);
    expect(
      await screen.findByText(/数据较旧，以下单实时执行为准/),
    ).toBeInTheDocument();
  });

  it("可创建任务并显示结果", async () => {
    render(<App />);
    const createBtns = await screen.findAllByRole("button", { name: "创建任务" });
    const createBtn = createBtns[0];
    fireEvent.click(createBtn);
    expect(await screen.findByText(/已创建：/)).toBeInTheDocument();
    expect(await screen.findByText(/状态 matching/)).toBeInTheDocument();
  });

  it("可进行 runner 接单并显示 accepted", async () => {
    render(<App />);
    const createBtns = await screen.findAllByRole("button", { name: "创建任务" });
    const createBtn = createBtns[0];
    fireEvent.click(createBtn);
    const acceptBtns = await screen.findAllByRole("button", { name: "标记接单" });
    const acceptBtn = acceptBtns[0];
    fireEvent.click(acceptBtn);
    expect(await screen.findByText(/当前状态 accepted/)).toBeInTheDocument();
  });
});
