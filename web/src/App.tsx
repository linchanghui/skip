import { useCallback, useEffect, useMemo, useState } from "react";
import {
  acceptTask,
  createQueueReport,
  createTask,
  fetchAreaChangi,
  fetchQueueSignal,
  fetchStoreDetail,
  fetchStores,
  listTasks,
  setRunnerAvailability,
} from "./api";
import type {
  Area,
  BusyLevel,
  QueueSignal,
  Store,
  StoreDetail,
  Task,
} from "./types";
import { MapPanel } from "./components/MapPanel";
import { StoreDetail as StoreDetailView } from "./components/StoreDetail";
import { StoreList } from "./components/StoreList";
import "./App.css";

type RoleView = "unselected" | "requester" | "runner";

type CategoryDef = {
  key: string;
  label: string;
  templates: Array<{ key: string; label: string }>;
};

const CATEGORY_DEFS: CategoryDef[] = [
  {
    key: "daily_micro_tasks",
    label: "Daily Micro-Tasks",
    templates: [
      { key: "queue_food_coffee", label: "Queue for Food/Coffee" },
      { key: "parcel_pickup", label: "Parcel Pickup" },
      { key: "small_errand", label: "Small Errand" },
    ],
  },
  {
    key: "high_stakes_time",
    label: "High-Stakes Time",
    templates: [
      { key: "visa_gov_queue", label: "Visa/Government Queue" },
      { key: "limited_drop_purchase", label: "Limited Drop Purchase" },
      { key: "slot_holding", label: "Slot Holding" },
    ],
  },
  {
    key: "physical_presence",
    label: "Physical Presence",
    templates: [
      { key: "attend_for_me", label: "Attend for Me" },
      { key: "hold_place", label: "Hold Place for Me" },
      { key: "verify_for_me", label: "Verify for Me" },
    ],
  },
];

export default function App(): JSX.Element {
  const [role, setRole] = useState<RoleView>("unselected");

  const [area, setArea] = useState<Area | null>(null);
  const [stores, setStores] = useState<Store[]>([]);
  const [listError, setListError] = useState<string | null>(null);

  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<StoreDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [queueSignal, setQueueSignal] = useState<QueueSignal | null>(null);

  const [taskUserId, setTaskUserID] = useState("user-web");
  const [taskCategory, setTaskCategory] = useState(CATEGORY_DEFS[0].key);
  const [taskTemplate, setTaskTemplate] = useState(CATEGORY_DEFS[0].templates[0].key);
  const [taskNote, setTaskNote] = useState("");
  const [taskResult, setTaskResult] = useState<Task | null>(null);
  const [taskError, setTaskError] = useState<string | null>(null);
  const [taskSubmitting, setTaskSubmitting] = useState(false);

  const [runnerId, setRunnerID] = useState("runner-alex");
  const [runnerOnline, setRunnerOnline] = useState(false);
  const [runnerTaskId, setRunnerTaskID] = useState("");
  const [runnerResult, setRunnerResult] = useState<Task | null>(null);
  const [runnerError, setRunnerError] = useState<string | null>(null);
  const [runnerSubmitting, setRunnerSubmitting] = useState(false);
  const [availableTasks, setAvailableTasks] = useState<Task[]>([]);
  const [myTasks, setMyTasks] = useState<Task[]>([]);
  const [runnerTasksLoading, setRunnerTasksLoading] = useState(false);

  const [reportBusy, setReportBusy] = useState<BusyLevel>("moderate");
  const [reportQueue, setReportQueue] = useState("3");
  const [reportWait, setReportWait] = useState("10");
  const [reportTTL, setReportTTL] = useState("30");
  const [reportSubmitting, setReportSubmitting] = useState(false);
  const [reportError, setReportError] = useState<string | null>(null);
  const [reportSuccess, setReportSuccess] = useState<string | null>(null);

  const refreshStores = useCallback(async () => {
    const { stores: st } = await fetchStores("changi");
    setStores(st);
    if (st.length) {
      setSelectedId((prev) => prev ?? st[0].id);
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [a, { stores: st }] = await Promise.all([
          fetchAreaChangi(),
          fetchStores("changi"),
        ]);
        if (cancelled) {
          return;
        }
        setArea(a);
        setStores(st);
        setListError(null);
        if (st.length) {
          setSelectedId((prev) => prev ?? st[0].id);
        }
      } catch (e) {
        if (!cancelled) {
          setListError(e instanceof Error ? e.message : "Failed to load data.");
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    const category = CATEGORY_DEFS.find((x) => x.key === taskCategory);
    if (!category) {
      return;
    }
    if (!category.templates.some((x) => x.key === taskTemplate)) {
      setTaskTemplate(category.templates[0].key);
    }
  }, [taskCategory, taskTemplate]);

  useEffect(() => {
    if (!selectedId) {
      setDetail(null);
      return;
    }
    let cancelled = false;
    setDetailLoading(true);
    setDetailError(null);
    (async () => {
      try {
        const d = await fetchStoreDetail(selectedId, 20);
        if (!cancelled) {
          setDetail(d);
        }
      } catch (e) {
        if (!cancelled) {
          setDetail(null);
          setDetailError(e instanceof Error ? e.message : "Failed to load store details.");
        }
      } finally {
        if (!cancelled) {
          setDetailLoading(false);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [selectedId]);

  useEffect(() => {
    if (!selectedId) {
      setQueueSignal(null);
      return;
    }
    let cancelled = false;
    (async () => {
      try {
        const sig = await fetchQueueSignal(selectedId);
        if (!cancelled) {
          setQueueSignal(sig);
        }
      } catch {
        if (!cancelled) {
          setQueueSignal(null);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [selectedId]);

  const refreshRunnerTasks = useCallback(async () => {
    setRunnerTasksLoading(true);
    try {
      const [available, mine] = await Promise.all([
        listTasks({ status: ["matching"], limit: 20 }),
        listTasks({
          runner_id: runnerId.trim(),
          status: ["accepted", "arrived", "queuing"],
          limit: 20,
        }),
      ]);
      setAvailableTasks(available.tasks);
      setMyTasks(mine.tasks);
    } finally {
      setRunnerTasksLoading(false);
    }
  }, [runnerId]);

  useEffect(() => {
    if (role !== "runner") {
      return;
    }
    let cancelled = false;
    (async () => {
      try {
        await refreshRunnerTasks();
      } catch (e) {
        if (!cancelled) {
          setRunnerError(e instanceof Error ? e.message : "Failed to load runner tasks.");
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [refreshRunnerTasks, role]);

  const onSelectStore = useCallback((id: string) => {
    setSelectedId(id);
  }, []);

  const selectedStoreId = useMemo(() => selectedId ?? "", [selectedId]);
  const selectedCategory = useMemo(
    () => CATEGORY_DEFS.find((x) => x.key === taskCategory) ?? CATEGORY_DEFS[0],
    [taskCategory],
  );

  const submitTask = useCallback(async () => {
    if (!selectedStoreId) {
      setTaskError("Please select a store first.");
      return;
    }
    setTaskSubmitting(true);
    setTaskError(null);
    try {
      const note = JSON.stringify({
        category: taskCategory,
        template: taskTemplate,
        user_note: taskNote.trim(),
      });
      const created = await createTask({
        user_id: taskUserId.trim(),
        store_id: selectedStoreId,
        task_type: "queue_for_me",
        note,
      });
      setTaskResult(created);
      setRunnerTaskID(created.id);
    } catch (e) {
      setTaskError(e instanceof Error ? e.message : "Failed to create task.");
    } finally {
      setTaskSubmitting(false);
    }
  }, [selectedStoreId, taskCategory, taskNote, taskTemplate, taskUserId]);

  const submitAccept = useCallback(async () => {
    const taskID = runnerTaskId.trim();
    if (!taskID) {
      setRunnerError("Please choose a task first.");
      return;
    }
    setRunnerSubmitting(true);
    setRunnerError(null);
    try {
      const accepted = await acceptTask({
        task_id: taskID,
        runner_id: runnerId.trim(),
      });
      setRunnerResult(accepted);
      await refreshRunnerTasks();
    } catch (e) {
      setRunnerError(e instanceof Error ? e.message : "Failed to accept task.");
    } finally {
      setRunnerSubmitting(false);
    }
  }, [refreshRunnerTasks, runnerId, runnerTaskId]);

  const submitReport = useCallback(async () => {
    if (!selectedStoreId) {
      setReportError("Please select a store first.");
      return;
    }
    setReportSubmitting(true);
    setReportError(null);
    setReportSuccess(null);
    try {
      const queueValue = Number.parseInt(reportQueue, 10);
      const waitValue = Number.parseInt(reportWait, 10);
      const ttlValue = Number.parseInt(reportTTL, 10);
      await createQueueReport({
        store_id: selectedStoreId,
        payload: {
          reporter_type: "runner",
          reporter_id: runnerId.trim(),
          busy_level: reportBusy,
          queue_length: Number.isNaN(queueValue) ? undefined : queueValue,
          wait_minutes_est: Number.isNaN(waitValue) ? undefined : waitValue,
          ttl_minutes: Number.isNaN(ttlValue) ? undefined : ttlValue,
        },
      });
      const [signal, latestDetail] = await Promise.all([
        fetchQueueSignal(selectedStoreId),
        fetchStoreDetail(selectedStoreId, 20),
        refreshStores(),
      ]);
      setQueueSignal(signal);
      setDetail(latestDetail);
      setReportSuccess("Queue report submitted.");
    } catch (e) {
      setReportError(
        e instanceof Error ? e.message : "Failed to submit queue report.",
      );
    } finally {
      setReportSubmitting(false);
    }
  }, [
    refreshStores,
    reportBusy,
    reportQueue,
    reportTTL,
    reportWait,
    runnerId,
    selectedStoreId,
  ]);

  const toggleRunnerOnline = useCallback(async () => {
    setRunnerError(null);
    try {
      const next = !runnerOnline;
      await setRunnerAvailability({
        runner_id: runnerId.trim(),
        is_online: next,
      });
      setRunnerOnline(next);
    } catch (e) {
      setRunnerError(
        e instanceof Error ? e.message : "Failed to update runner availability.",
      );
    }
  }, [runnerId, runnerOnline]);

  return (
    <div className="app">
      <header className="app__header">
        <h1>SKIPP · Time Arbitrage Marketplace</h1>
        {area ? <p className="app__tagline">{area.name}</p> : null}
      </header>

      {listError ? (
        <div className="app__banner app__banner--error">{listError}</div>
      ) : null}

      <main className="app__main">
        <aside className="app__aside">
          <section className="role-panel">
            <h2>Choose Your Role</h2>
            <p className="role-panel__helper">You can switch roles anytime.</p>
            <div className="role-panel__actions">
              <button
                type="button"
                className={role === "requester" ? "is-active" : ""}
                onClick={() => setRole("requester")}
              >
                Post Tasks
              </button>
              <button
                type="button"
                className={role === "runner" ? "is-active" : ""}
                onClick={() => setRole("runner")}
              >
                Accept Tasks
              </button>
            </div>
          </section>

          {role === "unselected" ? (
            <section className="action-panel">
              <p>Select a role to start.</p>
            </section>
          ) : null}

          {role === "requester" ? (
            <section className="action-panel">
              <h3>Task Creation Hub</h3>
              <label>
                User ID
                <input
                  value={taskUserId}
                  onChange={(e) => setTaskUserID(e.target.value)}
                />
              </label>
              <label>
                Category
                <select
                  value={taskCategory}
                  onChange={(e) => setTaskCategory(e.target.value)}
                >
                  {CATEGORY_DEFS.map((item) => (
                    <option key={item.key} value={item.key}>
                      {item.label}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                Template
                <select
                  value={taskTemplate}
                  onChange={(e) => setTaskTemplate(e.target.value)}
                >
                  {selectedCategory.templates.map((item) => (
                    <option key={item.key} value={item.key}>
                      {item.label}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                Notes
                <input
                  value={taskNote}
                  onChange={(e) => setTaskNote(e.target.value)}
                  placeholder="Need before 6 PM"
                />
              </label>
              <button type="button" onClick={submitTask} disabled={taskSubmitting}>
                {taskSubmitting ? "Submitting..." : "Create Task"}
              </button>
              {taskError ? <p className="action-panel__error">{taskError}</p> : null}
              {taskResult ? (
                <p>
                  Created: <code>{taskResult.id}</code> · Status {taskResult.status}
                </p>
              ) : null}
            </section>
          ) : null}

          {role === "runner" ? (
            <>
              <section className="action-panel">
                <h3>Runner Identity</h3>
                <label>
                  Runner ID
                  <input
                    value={runnerId}
                    onChange={(e) => setRunnerID(e.target.value)}
                  />
                </label>
                <button type="button" onClick={toggleRunnerOnline}>
                  {runnerOnline ? "Go Offline" : "Go Online"}
                </button>
                {runnerError ? (
                  <p className="action-panel__error">{runnerError}</p>
                ) : null}
              </section>

              <section className="action-panel">
                <h3>Available Tasks</h3>
                {runnerTasksLoading ? <p>Loading tasks...</p> : null}
                {availableTasks.length === 0 ? <p>No available tasks.</p> : null}
                <ul className="task-list">
                  {availableTasks.map((task) => (
                    <li key={task.id}>
                      <button
                        type="button"
                        onClick={() => {
                          setRunnerTaskID(task.id);
                          setSelectedId(task.store_id);
                        }}
                      >
                        {task.id} · {task.store_id}
                      </button>
                    </li>
                  ))}
                </ul>
                <label>
                  Task ID
                  <input
                    value={runnerTaskId}
                    onChange={(e) => setRunnerTaskID(e.target.value)}
                  />
                </label>
                <button
                  type="button"
                  onClick={submitAccept}
                  disabled={runnerSubmitting || !runnerTaskId.trim()}
                >
                  {runnerSubmitting ? "Processing..." : "Accept Task"}
                </button>
                {runnerResult ? (
                  <p>
                    Task <code>{runnerResult.id}</code> status is {runnerResult.status}
                  </p>
                ) : null}
              </section>

              <section className="action-panel">
                <h3>My Active Tasks</h3>
                {myTasks.length === 0 ? <p>No active tasks.</p> : null}
                <ul className="task-list">
                  {myTasks.map((task) => (
                    <li key={task.id}>
                      {task.id} · {task.status} · {task.store_id}
                    </li>
                  ))}
                </ul>
              </section>

              <section className="action-panel">
                <h3>Queue Report</h3>
                <label>
                  Busy Level
                  <select
                    value={reportBusy}
                    onChange={(e) => setReportBusy(e.target.value as BusyLevel)}
                  >
                    <option value="quiet">Quiet</option>
                    <option value="moderate">Moderate</option>
                    <option value="busy">Busy</option>
                    <option value="closed">Closed</option>
                  </select>
                </label>
                <label>
                  Queue Length
                  <input
                    value={reportQueue}
                    onChange={(e) => setReportQueue(e.target.value)}
                  />
                </label>
                <label>
                  Wait (minutes)
                  <input
                    value={reportWait}
                    onChange={(e) => setReportWait(e.target.value)}
                  />
                </label>
                <label>
                  TTL (minutes)
                  <input
                    value={reportTTL}
                    onChange={(e) => setReportTTL(e.target.value)}
                  />
                </label>
                <button
                  type="button"
                  onClick={submitReport}
                  disabled={reportSubmitting}
                >
                  {reportSubmitting ? "Submitting..." : "Submit Queue Report"}
                </button>
                {reportError ? <p className="action-panel__error">{reportError}</p> : null}
                {reportSuccess ? <p>{reportSuccess}</p> : null}
              </section>
            </>
          ) : null}

          <StoreList
            stores={stores}
            selectedId={selectedId}
            onSelect={onSelectStore}
          />
          <StoreDetailView
            detail={detail}
            queueSignal={queueSignal}
            loading={detailLoading}
            error={detailError}
          />
        </aside>
        <MapPanel
          area={area}
          stores={stores}
          selectedId={selectedId}
          onSelectStore={onSelectStore}
        />
      </main>
    </div>
  );
}
