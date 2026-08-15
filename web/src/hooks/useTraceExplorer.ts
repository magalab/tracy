import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { getDashboardMetrics, getTrace, listTraces } from "../api/client";
import type { DashboardMetrics, Page, Span, TraceDetails } from "../types";

export function useTraceExplorer(
  token: string,
  workspaceID: string,
  statusFilter: string,
  kindFilter: string,
  startTime: string,
  endTime: string,
) {
  const [page, setPage] = useState<Page>({ items: [] });
  const [metrics, setMetrics] = useState<DashboardMetrics | null>(null);
  const [selected, setSelected] = useState<Span[]>([]);
  const [selectedDetails, setSelectedDetails] = useState<Pick<
    TraceDetails,
    "start_time" | "end_time" | "span_count"
  > | null>(null);
  const [selectedID, setSelectedID] = useState("");
  const [selectedNextCursor, setSelectedNextCursor] = useState<string | undefined>();
  const [traceLoading, setTraceLoading] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [cursor, setCursor] = useState<string | undefined>();
  const pageRequestID = useRef(0);
  const metricsRequestID = useRef(0);
  const traceRequestID = useRef(0);

  const loadPage = useCallback(
    async (nextCursor?: string) => {
      if (!token) return;
      const requestID = ++pageRequestID.current;
      setLoading(true);
      setError("");
      try {
        const next = await listTraces(token, workspaceID, {
          cursor: nextCursor,
          status: statusFilter,
          kind: kindFilter,
          startTime,
          endTime,
        });
        if (requestID !== pageRequestID.current) return;
        setPage({ ...next, items: next.items ?? [] });
        setCursor(next.next_cursor);
      } catch (err) {
        if (requestID !== pageRequestID.current) return;
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        if (requestID === pageRequestID.current) setLoading(false);
      }
    },
    [endTime, kindFilter, startTime, statusFilter, token, workspaceID],
  );

  const loadMetrics = useCallback(async () => {
    if (!token) return;
    const requestID = ++metricsRequestID.current;
    try {
      const next = await getDashboardMetrics(token, workspaceID);
      if (requestID === metricsRequestID.current) setMetrics(next);
    } catch (err) {
      if (requestID === metricsRequestID.current) {
        setError(err instanceof Error ? err.message : String(err));
      }
    }
  }, [token, workspaceID]);

  useEffect(() => {
    setSelected([]);
    setSelectedDetails(null);
    setSelectedID("");
    if (token && workspaceID) {
      void loadPage();
      void loadMetrics();
    }
  }, [loadMetrics, loadPage, token, workspaceID]);

  const openTrace = useCallback(
    async (id: string) => {
      const requestID = ++traceRequestID.current;
      setSelectedID(id);
      setSelected([]);
      setSelectedDetails(null);
      setSelectedNextCursor(undefined);
      setTraceLoading(true);
      setError("");
      try {
        const result = await getTrace(id, token, workspaceID, { limit: 100 });
        if (requestID === traceRequestID.current) {
          setSelected(result.spans);
          setSelectedDetails({
            start_time: result.start_time,
            end_time: result.end_time,
            span_count: result.span_count,
          });
          setSelectedNextCursor(result.next_cursor);
        }
      } catch (err) {
        if (requestID === traceRequestID.current) {
          setSelectedID("");
          setSelectedDetails(null);
          setError(err instanceof Error ? err.message : String(err));
        }
      } finally {
        if (requestID === traceRequestID.current) setTraceLoading(false);
      }
    },
    [token, workspaceID],
  );

  const loadMoreTrace = useCallback(async () => {
    if (!selectedID || !selectedNextCursor || traceLoading) return;
    const requestID = ++traceRequestID.current;
    setTraceLoading(true);
    try {
      const result = await getTrace(selectedID, token, workspaceID, {
        cursor: selectedNextCursor,
        limit: 100,
      });
      if (requestID === traceRequestID.current) {
        setSelected((items) => [...items, ...result.spans]);
        setSelectedNextCursor(result.next_cursor);
      }
    } catch (err) {
      if (requestID === traceRequestID.current)
        setError(err instanceof Error ? err.message : String(err));
    } finally {
      if (requestID === traceRequestID.current) setTraceLoading(false);
    }
  }, [selectedID, selectedNextCursor, token, traceLoading, workspaceID]);

  const clearSelection = useCallback(() => {
    traceRequestID.current += 1;
    setSelectedID("");
    setSelected([]);
    setSelectedDetails(null);
    setSelectedNextCursor(undefined);
  }, []);

  return useMemo(
    () => ({
      page,
      metrics,
      selected,
      selectedDetails,
      selectedID,
      selectedNextCursor,
      traceLoading,
      loading,
      error,
      cursor,
      loadPage,
      loadMetrics,
      openTrace,
      loadMoreTrace,
      clearSelection,
    }),
    [
      clearSelection,
      cursor,
      error,
      loadMetrics,
      loadMoreTrace,
      loadPage,
      loading,
      metrics,
      openTrace,
      page,
      selected,
      selectedDetails,
      selectedID,
      selectedNextCursor,
      traceLoading,
    ],
  );
}
