import type { QueueSignal, StoreDetail } from "../types";

type Props = {
  detail: StoreDetail | null;
  queueSignal: QueueSignal | null;
  loading: boolean;
  error: string | null;
};

function busyLabel(level: string): string {
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
      return level;
  }
}

export function StoreDetail({
  detail,
  queueSignal,
  loading,
  error,
}: Props): JSX.Element {
  if (error) {
    return <div className="store-detail store-detail--error">{error}</div>;
  }
  if (loading) {
    return <div className="store-detail">Loading details...</div>;
  }
  if (!detail) {
    return <div className="store-detail">Please select a store.</div>;
  }

  return (
    <section className="store-detail" aria-live="polite">
      <h2 className="store-detail__title">{detail.name}</h2>
      <p className="store-detail__sub">
        {[detail.terminal, detail.floor].filter(Boolean).join(" · ") ||
          "Changi Demo"}
      </p>
      {detail.latest_status ? (
        <dl className="store-detail__dl">
          <div>
            <dt>Busyness</dt>
            <dd>{busyLabel(detail.latest_status.busy_level)}</dd>
          </div>
          {detail.latest_status.queue_length != null ? (
            <div>
              <dt>Queue length</dt>
              <dd>{detail.latest_status.queue_length}</dd>
            </div>
          ) : null}
          {detail.latest_status.wait_minutes_est != null ? (
            <div>
              <dt>Estimated wait</dt>
              <dd>~{detail.latest_status.wait_minutes_est} min</dd>
            </div>
          ) : null}
          <div>
            <dt>Updated at</dt>
            <dd>{new Date(detail.latest_status.as_of).toLocaleString()}</dd>
          </div>
        </dl>
      ) : (
        <p>No latest status.</p>
      )}

      {queueSignal ? (
        <section className="store-detail__signal">
          <h3 className="store-detail__history-title">Live Signal (MVP)</h3>
          <p>
            Last update: {new Date(queueSignal.last_updated_at).toLocaleString()}
            {" ("}
            {queueSignal.last_updated_x_mins_ago} min ago)
          </p>
          {queueSignal.status_expired || !queueSignal.signal ? (
            <p className="store-detail__expired">
              This signal is stale. Please rely on task execution for real-time results.
            </p>
          ) : (
            <p>
              Current: {busyLabel(queueSignal.signal.busy_level)}
              {queueSignal.signal.wait_minutes_est != null
                ? ` · ~${queueSignal.signal.wait_minutes_est} min`
                : ""}
            </p>
          )}
        </section>
      ) : null}

      {detail.status_history?.length ? (
        <>
          <h3 className="store-detail__history-title">Recent Reports</h3>
          <ol className="store-detail__history">
            {detail.status_history.slice(0, 8).map((h) => (
              <li key={h.id}>
                <time dateTime={h.reported_at}>
                  {new Date(h.reported_at).toLocaleString()}
                </time>
                {" · "}
                {busyLabel(h.busy_level)}
                {h.queue_length != null ? ` · Queue ${h.queue_length}` : ""}
              </li>
            ))}
          </ol>
        </>
      ) : null}
    </section>
  );
}
