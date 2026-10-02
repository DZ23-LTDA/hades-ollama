import { useEffect, useRef, useState } from "react";
import { agentFetch } from "@/lib/agenticClient";
import { shouldStopSseReconnect } from "@/lib/sse";

export interface MissionEvent {
  id: string;
  type: string;
  step_id?: string;
  created_at: string;
  payload?: Record<string, unknown>;
}

export interface BrowserFrame {
  url?: string;
  title?: string;
  screenshot?: string;
  timestamp: string;
}

export function useMissionEvents(missionId?: string | null) {
  const [events, setEvents] = useState<MissionEvent[]>([]);
  const [browserFrame, setBrowserFrame] = useState<BrowserFrame | null>(null);
  const [isLive, setIsLive] = useState(false);
  const [connectionError, setConnectionError] = useState<string | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    if (!missionId) {
      setEvents([]);
      setBrowserFrame(null);
      setIsLive(false);
      return;
    }

    let isMounted = true;

    // Load initial events from REST
    agentFetch<{ events: MissionEvent[] }>(`/api/agent/v1/missions/${encodeURIComponent(missionId)}/events`)
      .then((res) => {
        if (!isMounted) return;
        const initial = res.events ?? [];
        setEvents(initial);
        // Find latest browser frame if present
        for (let i = initial.length - 1; i >= 0; i--) {
          const evt = initial[i];
          if (evt.type === "browser.frame" && evt.payload && typeof evt.payload.screenshot === "string") {
            setBrowserFrame({
              url: typeof evt.payload.url === "string" ? evt.payload.url : undefined,
              title: typeof evt.payload.title === "string" ? evt.payload.title : undefined,
              screenshot: evt.payload.screenshot,
              timestamp: evt.created_at,
            });
            break;
          }
        }
      })
      .catch((err) => {
        if (isMounted) {
          setConnectionError(err instanceof Error ? err.message : "Falha ao carregar eventos");
        }
      });

    // Establish live SSE connection
    try {
      const streamUrl = `/api/agent/v1/missions/${encodeURIComponent(missionId)}/events/stream`;
      const es = new EventSource(streamUrl);
      eventSourceRef.current = es;
      let consecutiveErrors = 0;

      es.onopen = () => {
        consecutiveErrors = 0;
        if (isMounted) {
          setIsLive(true);
          setConnectionError(null);
        }
      };

      es.addEventListener("mission", (msg) => {
        if (!isMounted) return;
        try {
          const newEvents = JSON.parse(msg.data) as MissionEvent[];
          if (Array.isArray(newEvents) && newEvents.length > 0) {
            setEvents((prev) => {
              const existingIds = new Set(prev.map((e) => e.id));
              const additions = newEvents.filter((e) => !existingIds.has(e.id));
              return additions.length > 0 ? [...prev, ...additions] : prev;
            });

            // Check for browser.frame events
            for (const evt of newEvents) {
              if (evt.type === "browser.frame" && evt.payload && typeof evt.payload.screenshot === "string") {
                setBrowserFrame({
                  url: typeof evt.payload.url === "string" ? evt.payload.url : undefined,
                  title: typeof evt.payload.title === "string" ? evt.payload.title : undefined,
                  screenshot: evt.payload.screenshot,
                  timestamp: evt.created_at,
                });
              }
            }
          }
        } catch (e) {
          console.warn("Failed to parse mission SSE event payload:", e);
        }
      });

      es.onerror = () => {
        if (isMounted) {
          setIsLive(false);
        }
        consecutiveErrors += 1;
        // Stop the native auto-reconnect once it keeps failing, so a down/404
        // stream endpoint does not become a reconnect storm.
        if (shouldStopSseReconnect(consecutiveErrors)) {
          es.close();
          if (eventSourceRef.current === es) eventSourceRef.current = null;
          if (isMounted) {
            setConnectionError(
              "Conexão com o stream perdida. Recarregue para tentar novamente.",
            );
          }
        }
      };
    } catch (e) {
      if (isMounted) {
        setConnectionError(e instanceof Error ? e.message : "Erro ao abrir stream SSE");
      }
    }

    return () => {
      isMounted = false;
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
    };
  }, [missionId]);

  return { events, browserFrame, isLive, connectionError };
}
