import type { Store } from "../types";

type Props = {
  stores: Store[];
  selectedId: string | null;
  onSelect: (id: string) => void;
};

function busyLabel(level: string | undefined): string {
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
      return "未知";
  }
}

export function StoreList({ stores, selectedId, onSelect }: Props): JSX.Element {
  return (
    <ul className="store-list" aria-label="门店列表">
      {stores.map((s) => (
        <li key={s.id}>
          <button
            type="button"
            className={
              s.id === selectedId ? "store-list__btn is-active" : "store-list__btn"
            }
            onClick={() => onSelect(s.id)}
          >
            <span className="store-list__name">{s.name}</span>
            {s.latest_status ? (
              <span className="store-list__meta">
                {busyLabel(s.latest_status.busy_level)}
                {s.latest_status.wait_minutes_est != null
                  ? ` · 约 ${s.latest_status.wait_minutes_est} 分钟`
                  : ""}
              </span>
            ) : (
              <span className="store-list__meta">暂无状态</span>
            )}
          </button>
        </li>
      ))}
    </ul>
  );
}
