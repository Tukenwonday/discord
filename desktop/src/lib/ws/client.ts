import type {
  ClientEvent,
  ClientEventMap,
  ClientHandler,
  ConnectionState,
  ServerEvent,
  StatusListener,
  Unsubscribe,
  WsEnvelope,
} from "./events";
import { useSettingsStore } from "@/stores/settings.store";

const HEARTBEAT_INTERVAL = 25_000;
const BASE_BACKOFF = 1_000;
const MAX_BACKOFF = 30_000;
/** A connection silent for this long is treated as dead and recycled. */
const STALE_AFTER = 90_000;

/** Internal, erased handler shape; the public `on` restores the generics. */
type AnyHandler = (data: unknown, envelope: WsEnvelope<string, unknown>) => void;

export class WsClient {
  private socket: WebSocket | null = null;
  private token: string | null = null;
  private url: string | null = null;
  private handlers = new Map<ServerEvent, Set<AnyHandler>>();
  private statusListeners = new Set<StatusListener>();
  private heartbeatTimer: number | null = null;
  private reconnectTimer: number | null = null;
  private staleTimer: number | null = null;
  private attempt = 0;
  private closedByUser = false;
  private state: ConnectionState = "idle";

  getState(): ConnectionState {
    return this.state;
  }

  private setState(next: ConnectionState): void {
    if (this.state === next) return;
    this.state = next;
    for (const listener of this.statusListeners) listener(next);
  }

  onStatus(listener: StatusListener): Unsubscribe {
    this.statusListeners.add(listener);
    listener(this.state);
    return () => {
      this.statusListeners.delete(listener);
    };
  }

  /**
   * Connects to `url` with `?token=`. Re-connecting with an identical token and
   * url is a no-op, so effects may call this freely on re-render.
   */
  connect(token: string, url: string): void {
    if (this.socket && this.token === token && this.url === url) return;

    this.teardown();
    this.token = token;
    this.url = url;
    this.closedByUser = false;
    this.attempt = 0;
    this.open();
  }

  private open(): void {
    if (!this.token || !this.url) return;

    const separator = this.url.includes("?") ? "&" : "?";
    const target = `${this.url}${separator}token=${encodeURIComponent(this.token)}`;

    let socket: WebSocket;
    try {
      socket = new WebSocket(target);
    } catch {
      this.scheduleReconnect();
      return;
    }

    this.socket = socket;
    this.setState(this.attempt === 0 ? "connecting" : "reconnecting");

    socket.onopen = () => {
      // A socket replaced by a newer connect() must not revive this one.
      if (this.socket !== socket) return;
      this.attempt = 0;
      this.setState("open");
      this.startHeartbeat(socket);
    };

    socket.onmessage = (event: MessageEvent<unknown>) => {
      if (this.socket !== socket) return;
      if (this.staleTimer !== null) window.clearTimeout(this.staleTimer);
      this.staleTimer = window.setTimeout(() => {
        if (this.socket === socket) socket.close();
      }, STALE_AFTER);
      this.dispatch(event.data);
    };

    socket.onerror = () => {
      if (this.socket === socket && this.staleTimer !== null) {
        window.clearTimeout(this.staleTimer);
        this.staleTimer = null;
      }
    };

    socket.onclose = () => {
      if (this.socket !== socket) return;
      this.clearTimers();
      this.socket = null;
      if (this.closedByUser) {
        this.setState("closed");
        return;
      }
      this.scheduleReconnect();
    };
  }

  private scheduleReconnect(): void {
    if (this.closedByUser || !this.token) {
      this.setState("closed");
      return;
    }
    this.setState("reconnecting");
    // Exponential backoff with jitter, so a server restart does not draw a
    // synchronised stampede from every connected client.
    const ceiling = Math.min(BASE_BACKOFF * 2 ** this.attempt, MAX_BACKOFF);
    const delay = Math.min(ceiling, MAX_BACKOFF) * (0.7 + Math.random() * 0.6);
    this.attempt += 1;

    if (this.reconnectTimer !== null) window.clearTimeout(this.reconnectTimer);
    this.reconnectTimer = window.setTimeout(() => {
      this.reconnectTimer = null;
      this.open();
    }, delay);
  }

  private startHeartbeat(socket: WebSocket): void {
    if (this.heartbeatTimer !== null) window.clearInterval(this.heartbeatTimer);
    this.heartbeatTimer = window.setInterval(() => {
      if (socket.readyState !== WebSocket.OPEN) return;
      this.send("heartbeat", {});
    }, HEARTBEAT_INTERVAL);
  }
private clearTimers(): void {
    if (this.heartbeatTimer !== null) {
      window.clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
    if (this.staleTimer !== null) {
      window.clearTimeout(this.staleTimer);
      this.staleTimer = null;
    }
    if (this.reconnectTimer !== null) {
      window.clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  private teardown(): void {
    this.clearTimers();
    const socket = this.socket;
    this.socket = null;
    if (socket) {
      socket.onopen = null;
      socket.onmessage = null;
      socket.onerror = null;
      socket.onclose = null;
      if (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING) {
        socket.close();
      }
    }
  }

  private dispatch(raw: unknown): void {
    if (typeof raw !== "string") return;

    let parsed: WsEnvelope<string, unknown>;
    try {
      parsed = JSON.parse(raw) as WsEnvelope<string, unknown>;
    } catch {
      return;
    }

    if (!parsed || typeof parsed.event !== "string") return;

    const set = this.handlers.get(parsed.event as ServerEvent);
    if (!set) return;

    for (const handler of Array.from(set)) {
      try {
        handler(parsed.data, parsed);
      } catch (error) {
        // One broken listener must never take down the dispatch loop.
        console.error("[ws] handler failed", error);
      }
    }
  }

  on<K extends ServerEvent>(event: K, handler: ClientHandler<K>): Unsubscribe {
    const erased = handler as AnyHandler;
    let set = this.handlers.get(event);
    if (!set) {
      set = new Set<AnyHandler>();
      this.handlers.set(event, set);
    }
    set.add(erased);
    return () => {
      set?.delete(erased);
    };
  }

  send<K extends ClientEvent>(event: K, data: ClientEventMap[K], nonce?: string): void {
    const socket = this.socket;
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    const envelope: WsEnvelope<K, ClientEventMap[K]> = nonce
      ? { event, data, nonce }
      : { event, data };
    try {
      socket.send(JSON.stringify(envelope));
    } catch (error) {
      console.error("[ws] send failed", error);
    }
  }

  close(): void {
    this.closedByUser = true;
    this.token = null;
    this.teardown();
    this.setState("closed");
  }
}

/** Process-wide socket; AppPage connects it once an access token exists. */
export const socket = new WsClient();

/** Reads the configured WS endpoint, falling back to the build-time env var. */
export function resolveWsUrl(): string {
  return useSettingsStore.getState().wsUrl || import.meta.env.VITE_WS_URL || "";
}