import type { StoreDetail } from "../types";

type Props = {
  detail: StoreDetail | null;
  loading: boolean;
  error: string | null;
};

function busyLabel(level: string): string {
  switch (level) {
    case "quiet":
      return "空闲";
    case "moderate":
      return "适中";
    case "busy":
      return "繁忙";
    case "closed":
      return "休息";
    default:
      return level;
  }
}

export function StoreDetail({ detail, loading, error }: Props): JSX.Element {
  if (error) {
    return <div className="store-detail store-detail--error">{error}</div>;
  }
  if (loading) {
    return <div className="store-detail">加载详情…</div>;
  }
  if (!detail) {
    return <div className="store-detail">请选择一家门店。</div>;
  }

  return (
    <section className="store-detail" aria-live="polite">
      <h2 className="store-detail__title">{detail.name}</h2>
      <p className="store-detail__sub">
        {[detail.terminal, detail.floor].filter(Boolean).join(" · ") || "樟宜 Demo"}
      </p>
      {detail.latest_status ? (
        <dl className="store-detail__dl">
          <div>
            <dt>繁忙度</dt>
            <dd>{busyLabel(detail.latest_status.busy_level)}</dd>
          </div>
          {detail.latest_status.queue_length != null ? (
            <div>
              <dt>排队长度</dt>
              <dd>{detail.latest_status.queue_length}</dd>
            </div>
          ) : null}
          {detail.latest_status.wait_minutes_est != null ? (
            <div>
              <dt>预计等待</dt>
              <dd>约 {detail.latest_status.wait_minutes_est} 分钟</dd>
            </div>
          ) : null}
          <div>
            <dt>更新时间</dt>
            <dd>{new Date(detail.latest_status.as_of).toLocaleString()}</dd>
          </div>
        </dl>
      ) : (
        <p>暂无最新状态。</p>
      )}

      {detail.status_history?.length ? (
        <>
          <h3 className="store-detail__history-title">最近上报</h3>
          <ol className="store-detail__history">
            {detail.status_history.slice(0, 8).map((h) => (
              <li key={h.id}>
                <time dateTime={h.reported_at}>
                  {new Date(h.reported_at).toLocaleString()}
                </time>
                {" · "}
                {busyLabel(h.busy_level)}
                {h.queue_length != null ? ` · 排队 ${h.queue_length}` : ""}
              </li>
            ))}
          </ol>
        </>
      ) : null}
    </section>
  );
}
