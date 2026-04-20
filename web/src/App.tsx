import { useCallback, useEffect, useState } from "react";
import { fetchAreaChangi, fetchStoreDetail, fetchStores } from "./api";
import type { Area, Store, StoreDetail } from "./types";
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
          setListError(e instanceof Error ? e.message : "加载失败");
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
          setDetailError(e instanceof Error ? e.message : "加载失败");
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

  const onSelectStore = useCallback((id: string) => {
    setSelectedId(id);
  }, []);

  return (
    <div className="app">
      <header className="app__header">
        <h1>Skip · 樟宜 Demo</h1>
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
