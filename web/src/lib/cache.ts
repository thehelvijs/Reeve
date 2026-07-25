import { useCallback, useEffect, useRef, useState } from 'react';

// A tiny stale-while-revalidate cache keyed by request path. Values live at
// module scope so they survive route changes: revisiting a page shows the last
// data instantly while a background refresh updates it. No external library.

type Entry = { data: unknown };

const cache = new Map<string, Entry>();
const subscribers = new Map<string, Set<() => void>>();
const inflight = new Map<string, Promise<void>>();

function notify(key: string) {
  const subs = subscribers.get(key);
  if (subs) {
    subs.forEach((fn) => fn());
  }
}

export function getCached<T>(key: string): T | undefined {
  return cache.get(key)?.data as T | undefined;
}

// prefetch fetches and stores a value, deduping concurrent calls for the same
// key. A failed fetch leaves any prior cached value untouched.
export function prefetch<T>(key: string, fetcher: () => Promise<T>): Promise<void> {
  const existing = inflight.get(key);
  if (existing) {
    return existing;
  }
  const run = fetcher()
    .then((data) => {
      cache.set(key, { data });
      notify(key);
    })
    .catch(() => undefined)
    .finally(() => {
      inflight.delete(key);
    });
  inflight.set(key, run);
  return run;
}

// useResource returns cached data immediately (loading=false) when present, and
// only reports loading on a cold key with nothing cached yet. It revalidates on
// mount and, when pollMs is set, on an interval.
export function useResource<T>(key: string, fetcher: () => Promise<T>, pollMs?: number) {
  const [data, setData] = useState<T | undefined>(() => getCached<T>(key));
  const [loading, setLoading] = useState<boolean>(() => getCached<T>(key) === undefined);
  const fetcherRef = useRef(fetcher);
  fetcherRef.current = fetcher;

  const refresh = useCallback(() => prefetch(key, fetcherRef.current), [key]);

  useEffect(() => {
    const sync = () => {
      setData(getCached<T>(key));
      setLoading(getCached<T>(key) === undefined);
    };
    let subs = subscribers.get(key);
    if (!subs) {
      subs = new Set();
      subscribers.set(key, subs);
    }
    subs.add(sync);
    sync();
    refresh();
    let id: number | undefined;
    if (pollMs) {
      id = window.setInterval(refresh, pollMs);
    }
    return () => {
      subs!.delete(sync);
      if (id) {
        window.clearInterval(id);
      }
    };
  }, [key, pollMs, refresh]);

  return { data, loading, refresh };
}
