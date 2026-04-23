import type { Store } from "../types";

type Props = {
  stores: Store[];
  selectedId: string | null;
  onSelect: (id: string) => void;
};

function busyLabel(level: string | undefined): string {
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
      return "Unknown";
  }
}

export function StoreList({ stores, selectedId, onSelect }: Props): JSX.Element {
  return (
    <ul className="store-list" aria-label="Store list">
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
                  ? ` · ~${s.latest_status.wait_minutes_est} min`
                  : ""}
              </span>
            ) : (
              <span className="store-list__meta">No status yet</span>
            )}
          </button>
        </li>
      ))}
    </ul>
  );
}
