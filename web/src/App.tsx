import { useCallback, useEffect, useMemo, useState } from "react";
import {
  acceptTask,
  createTask,
  fetchAreaChangi,
  fetchQueueSignal,
  fetchStoreDetail,
  fetchStores,
} from "./api";
import type { Area, QueueSignal, Store, StoreDetail, Task } from "./types";
import { MapPanel } from "./components/MapPanel";
import { StoreDetail as StoreDetailView } from "./components/StoreDetail";
import { StoreList } from "./components/StoreList";
import "./App.css";

export default function App(): JSX.Element {
  const [area, setArea] = useState<Area | null>(null);
  const [stores, setStores] = useState<Store[]>([]);
  const [listError, setListError] = useState<string | null>(null);

  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<StoreDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [queueSignal, setQueueSignal] = useState<QueueSignal | null>(null);

  const [taskUserId, setTaskUserID] = useState("user-web");
  const [taskResult, setTaskResult] = useState<Task | null>(null);
  const [taskError, setTaskError] = useState<string | null>(null);
  const [taskSubmitting, setTaskSubmitting] = useState(false);

  const [runnerId, setRunnerID] = useState("runner-alex");
  const [runnerTaskId, setRunnerTaskID] = useState("");
  const [runnerResult, setRunnerResult] = useState<Task | null>(null);
  const [runnerError, setRunnerError] = useState<string | null>(null);
  const [runnerSubmitting, setRunnerSubmitting] = useState(false);

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

  const onSelectStore = useCallback((id: string) => {
    setSelectedId(id);
  }, []);

  const selectedStoreId = useMemo(() => selectedId ?? "", [selectedId]);

  const submitTask = useCallback(async () => {
    if (!selectedStoreId) {
      setTaskError("Please select a store first.");
      return;
    }
    setTaskSubmitting(true);
    setTaskError(null);
    try {
      const created = await createTask({
        user_id: taskUserId.trim(),
        store_id: selectedStoreId,
        task_type: "queue_for_me",
      });
      setTaskResult(created);
      setRunnerTaskID(created.id);
    } catch (e) {
      setTaskError(e instanceof Error ? e.message : "Failed to create task.");
    } finally {
      setTaskSubmitting(false);
    }
  }, [selectedStoreId, taskUserId]);

  const submitAccept = useCallback(async () => {
    setRunnerSubmitting(true);
    setRunnerError(null);
    try {
      const accepted = await acceptTask({
        task_id: runnerTaskId.trim(),
        runner_id: runnerId.trim(),
      });
      setRunnerResult(accepted);
    } catch (e) {
      setRunnerError(e instanceof Error ? e.message : "Failed to accept task.");
    } finally {
      setRunnerSubmitting(false);
    }
  }, [runnerId, runnerTaskId]);

  return (
    <div className="app">
      <header className="app__header">
        <h1>Skip · Changi Demo</h1>
        {area ? <p className="app__tagline">{area.name}</p> : null}
      </header>

      {listError ? (
        <div className="app__banner app__banner--error">{listError}</div>
      ) : null}

      <main className="app__main">
        <aside className="app__aside">
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
          <section className="action-panel">
            <h3>Create Task (MVP)</h3>
            <label>
              User ID
              <input
                value={taskUserId}
                onChange={(e) => setTaskUserID(e.target.value)}
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
          <section className="action-panel">
            <h3>Runner Accept (Internal)</h3>
            <label>
              Runner ID
              <input
                value={runnerId}
                onChange={(e) => setRunnerID(e.target.value)}
              />
            </label>
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
              {runnerSubmitting ? "Processing..." : "Mark Accepted"}
            </button>
            {runnerError ? (
              <p className="action-panel__error">{runnerError}</p>
            ) : null}
            {runnerResult ? (
              <p>
                Task <code>{runnerResult.id}</code> status is {runnerResult.status}
              </p>
            ) : null}
          </section>
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
