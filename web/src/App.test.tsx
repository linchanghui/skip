import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
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
  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  beforeEach(() => {
    vi.stubEnv("VITE_GOOGLE_MAPS_API_KEY", "");
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = typeof input === "string" ? input : input.toString();

        if (url.endsWith("/v1/areas/changi")) {
          return new Response(JSON.stringify(area), { status: 200 });
        }
        if (url.includes("/v1/stores?") && url.includes("area_id=changi")) {
          return new Response(JSON.stringify(stores), { status: 200 });
        }
        if (url.includes("/v1/stores/sb-jewel/queue-signal")) {
          return new Response(JSON.stringify(queueSignalExpired), { status: 200 });
        }
        if (url.includes("/v1/stores/sb-jewel?") && url.includes("history_limit")) {
          return new Response(JSON.stringify(detail), { status: 200 });
        }
        if (url.includes("/v1/tasks?") && url.includes("status=matching")) {
          return new Response(
            JSON.stringify({
              tasks: [
                {
                  id: "task-002",
                  user_id: "user-002",
                  store_id: "sb-jewel",
                  task_type: "queue_for_me",
                  status: "matching",
                  requested_at: "2026-04-20T10:03:00Z",
                  sla_accept_by: "2026-04-20T10:08:00Z",
                },
              ],
            }),
            { status: 200 },
          );
        }
        if (url.includes("/v1/tasks?") && url.includes("runner_id=runner-alex")) {
          return new Response(JSON.stringify({ tasks: [] }), { status: 200 });
        }
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
        if (url.includes("/v1/tasks/task-002/accept") && init?.method === "POST") {
          return new Response(
            JSON.stringify({
              id: "task-002",
              user_id: "user-002",
              store_id: "sb-jewel",
              task_type: "queue_for_me",
              status: "accepted",
              accepted_runner_id: "runner-alex",
              requested_at: "2026-04-20T10:03:00Z",
              sla_accept_by: "2026-04-20T10:08:00Z",
            }),
            { status: 200 },
          );
        }
        if (
          url.includes("/v1/runners/runner-alex/availability") &&
          init?.method === "POST"
        ) {
          return new Response(
            JSON.stringify({
              runner_id: "runner-alex",
              is_online: true,
              active_task_id: null,
              last_ping_at: "2026-04-20T10:02:00Z",
              updated_at: "2026-04-20T10:02:00Z",
            }),
            { status: 200 },
          );
        }
        if (url.includes("/v1/stores/sb-jewel/queue-reports") && init?.method === "POST") {
          return new Response(
            JSON.stringify({
              id: 101,
              store_id: "sb-jewel",
              reporter_type: "runner",
              reporter_id: "runner-alex",
              queue_length: 4,
              wait_minutes_est: 9,
              busy_level: "moderate",
              confidence_flag: "normal",
              reported_at: "2026-04-20T10:06:00Z",
              expires_at: "2026-04-20T10:36:00Z",
              created_at: "2026-04-20T10:06:00Z",
            }),
            { status: 201 },
          );
        }

        return new Response("not found", { status: 404 });
      }),
    );
  });

  it("shows role selection and store list", async () => {
    render(<App />);
    expect(await screen.findByText("Choose Your Role")).toBeInTheDocument();
    expect(await screen.findByText("Starbucks (Jewel)")).toBeInTheDocument();
  });

  it("creates task from requester hub", async () => {
    render(<App />);
    fireEvent.click(await screen.findByRole("button", { name: "Post Tasks" }));
    fireEvent.click(await screen.findByRole("button", { name: "Create Task" }));
    expect(await screen.findByText(/Created:/)).toBeInTheDocument();
    expect(await screen.findByText(/Status matching/)).toBeInTheDocument();
  });

  it("accepts task and submits queue report in runner console", async () => {
    render(<App />);
    fireEvent.click(await screen.findByRole("button", { name: "Accept Tasks" }));

    const taskBtn = await screen.findByRole("button", {
      name: /task-002 · sb-jewel/i,
    });
    fireEvent.click(taskBtn);

    fireEvent.click(await screen.findByRole("button", { name: "Accept Task" }));
    expect(await screen.findByText(/status is accepted/)).toBeInTheDocument();

    fireEvent.click(await screen.findByRole("button", { name: "Submit Queue Report" }));
    expect(await screen.findByText("Queue report submitted.")).toBeInTheDocument();
  });
});
