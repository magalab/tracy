import { useCallback, useEffect, useMemo, useState } from "react";
import {
  createAnnotation,
  deleteAnnotation,
  getDashboardMetrics,
  getTrace,
  listTraces,
} from "../api/client";
import type { Annotation, DashboardMetrics, Page, Span } from "../types";

export function useTraceExplorer(token: string, statusFilter: string, kindFilter: string) {
  const [page, setPage] = useState<Page>({ items: [] });
  const [metrics, setMetrics] = useState<DashboardMetrics | null>(null);
  const [selected, setSelected] = useState<Span[]>([]);
  const [annotations, setAnnotations] = useState<Annotation[]>([]);
  const [selectedID, setSelectedID] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [cursor, setCursor] = useState<string | undefined>();

  const loadPage = useCallback(
    async (nextCursor?: string) => {
      if (!token) return;
      setLoading(true);
      setError("");
      try {
        const next = await listTraces(token, {
          cursor: nextCursor,
          status: statusFilter,
          kind: kindFilter,
        });
        setPage({ ...next, items: next.items ?? [] });
        setCursor(next.next_cursor);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        setLoading(false);
      }
    },
    [kindFilter, statusFilter, token],
  );

  const loadMetrics = useCallback(async () => {
    if (!token) return;
    try {
      setMetrics(await getDashboardMetrics(token));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [token]);

  useEffect(() => {
    setSelected([]);
    setAnnotations([]);
    setSelectedID("");
    if (token) {
      void loadPage();
      void loadMetrics();
    }
  }, [loadMetrics, loadPage, token]);

  const openTrace = useCallback(
    async (id: string) => {
      setSelectedID(id);
      setError("");
      try {
        const result = await getTrace(id, token);
        setSelected(result.spans);
        setAnnotations(result.annotations);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    },
    [token],
  );

  const addAnnotation = useCallback(
    async (input: { key: string; score: number; label: string; comment: string }) => {
      if (!selectedID || !input.key.trim()) return;
      try {
        const annotation = await createAnnotation(selectedID, token, {
          ...input,
          key: input.key.trim(),
        });
        setAnnotations((items) => [...items, annotation]);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    },
    [selectedID, token],
  );

  const removeAnnotation = useCallback(
    async (id: string) => {
      try {
        await deleteAnnotation(id, token);
        setAnnotations((items) => items.filter((item) => item.id !== id));
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    },
    [token],
  );

  return useMemo(
    () => ({
      page,
      metrics,
      selected,
      annotations,
      selectedID,
      loading,
      error,
      cursor,
      loadPage,
      loadMetrics,
      openTrace,
      addAnnotation,
      removeAnnotation,
    }),
    [
      addAnnotation,
      annotations,
      cursor,
      error,
      loadMetrics,
      loadPage,
      loading,
      metrics,
      openTrace,
      page,
      removeAnnotation,
      selected,
      selectedID,
    ],
  );
}
