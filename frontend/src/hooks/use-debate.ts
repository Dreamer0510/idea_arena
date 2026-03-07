import { useState, useCallback, useRef, useEffect } from "react";
import apiClient from "@/lib/api-client";

export interface DebateEvent {
  type: string;
  idea_id: number;
  round?: number;
  agent?: string;
  content?: string;
  data?: Record<string, unknown>;
}

export function useDebate(ideaId: number) {
  const [events, setEvents] = useState<DebateEvent[]>([]);
  const [isRunning, setIsRunning] = useState(false);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  const startDebate = useCallback(async () => {
    if (!ideaId) return;
    setError(null);
    setEvents([]);

    try {
      await apiClient.post(`/api/v1/debate/${ideaId}/start`);
      setIsRunning(true);
    } catch (err: unknown) {
      const message =
        err instanceof Error
          ? err.message
          : "启动辩论失败";
      // Try to extract axios error message
      if (typeof err === "object" && err !== null && "response" in err) {
        const axiosErr = err as { response?: { data?: { error?: string } } };
        if (axiosErr.response?.data?.error) {
          setError(axiosErr.response.data.error);
          return;
        }
      }
      setError(message);
    }
  }, [ideaId]);

  const intervene = useCallback(async (extraRounds: number, prompt: string) => {
    if (!ideaId) return;
    setError(null);
    setEvents([]);

    try {
      await apiClient.post(`/api/v1/debate/${ideaId}/intervene`, {
        extra_rounds: extraRounds,
        prompt,
      });
      setIsRunning(true);
    } catch (err: unknown) {
      if (typeof err === "object" && err !== null && "response" in err) {
        const axiosErr = err as { response?: { data?: { error?: string } } };
        if (axiosErr.response?.data?.error) {
          setError(axiosErr.response.data.error);
          return;
        }
      }
      setError(err instanceof Error ? err.message : "人工介入失败");
    }
  }, [ideaId]);

  const connectSSE = useCallback(() => {
    if (!ideaId || eventSourceRef.current) return;

    const baseURL =
      process.env.NEXT_PUBLIC_API_URL || "http://localhost:8000";
    const url = `${baseURL}/api/v1/debate/${ideaId}/stream`;

    const es = new EventSource(url);
    eventSourceRef.current = es;

    es.onopen = () => {
      setIsConnected(true);
    };

    es.onmessage = (e) => {
      try {
        const event: DebateEvent = JSON.parse(e.data);
        setEvents((prev) => [...prev, event]);

        if (event.type === "connected") {
          setIsConnected(true);
        }

        if (
          event.type === "done" ||
          event.type === "error" ||
          event.type === "stream_end"
        ) {
          setIsRunning(false);
          es.close();
          eventSourceRef.current = null;
          setIsConnected(false);
        }
      } catch {
        // ignore parse errors
      }
    };

    es.onerror = () => {
      setIsConnected(false);
      es.close();
      eventSourceRef.current = null;
    };
  }, [ideaId]);

  const disconnect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
      setIsConnected(false);
    }
  }, []);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      disconnect();
    };
  }, [disconnect]);

  // Derived state
  const latestEvent = events.length > 0 ? events[events.length - 1] : null;
  const currentRound =
    events.reduce((max, e) => Math.max(max, e.round || 0), 0);
  const scores = events
    .filter((e) => e.type === "eval" && e.data)
    .map((e) => e.data as Record<string, number>);
  const latestScores = scores.length > 0 ? scores[scores.length - 1] : null;

  return {
    events,
    isRunning,
    isConnected,
    error,
    latestEvent,
    currentRound,
    latestScores,
    startDebate,
    intervene,
    connectSSE,
    disconnect,
  };
}
