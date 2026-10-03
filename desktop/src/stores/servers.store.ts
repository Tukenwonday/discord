import { create } from "zustand";
import type {
  Channel,
  ChannelType,
  Invite,
  Member,
  Role,
  Server,
  ServerDetail,
} from "@/types";
import * as serversApi from "@/lib/api/servers";
import * as channelsApi from "@/lib/api/channels";

export interface ServersState {
  servers: Server[];
  activeServerId: string | null;
  activeChannelId: string | null;
  details: Record<string, ServerDetail>;
  loading: boolean;
  error: string | null;
  loadServers: () => Promise<void>;
  loadServer: (id: string) => Promise<ServerDetail | null>;
  createServer: (input: { name: string; description?: string }) => Promise<Server | null>;
  createChannel: (
    serverId: string,
    input: { name: string; type?: ChannelType; topic?: string; parentId?: string | null },
  ) => Promise<Channel | null>;
  selectServer: (serverId: string) => void;
  selectChannel: (channelId: string) => void;
  removeServer: (serverId: string) => Promise<void>;
  setError: (error: string | null) => void;
}

/** Categories sort first, then everything else by its explicit position. */
export function compareChannels(a: Channel, b: Channel): number {
  const aCategory = a.type === "category";
  const bCategory = b.type === "category";
  if (aCategory !== bCategory) return aCategory ? -1 : 1;
  return a.position - b.position;
}

/** Ordered copy of a server's channels; categories lead. */
export function sortChannels(detail: ServerDetail | null): Channel[] {
  if (!detail) return [];
  return [...detail.channels].sort(compareChannels);
}

export const useServersStore = create<ServersState>((set, get) => ({
  servers: [],
  activeServerId: null,
  activeChannelId: null,
  details: {},
  loading: false,
  error: null,

  setError: (error) => set({ error }),

  loadServers: async () => {
    set({ loading: true, error: null });
    try {
      const page = await serversApi.listServers();
      set({ servers: page.items, loading: false });

      // Land on a real channel as soon as the first server is available.
      if (!get().activeServerId && page.items.length > 0) {
        await get().loadServer(page.items[0].id);
      }
    } catch (error) {
      set({
        loading: false,
        error: error instanceof Error ? error.message : "Could not load servers",
      });
    }
  },

  loadServer: async (id) => {
    try {
      const detail = await serversApi.getServer(id);
      set((state) => ({ details: { ...state.details, [id]: detail } }));

      const current = get().activeChannelId;
      const stillValid = current
        ? detail.channels.some((channel) => channel.id === current)
        : false;

      if (!stillValid) {
        const firstText = detail.channels.find((channel) => channel.type === "text");
        set({
          activeServerId: id,
          activeChannelId: firstText?.id ?? detail.channels[0]?.id ?? null,
        });
      } else {
        set({ activeServerId: id });
      }

      return detail;
    } catch (error) {
      set({ error: error instanceof Error ? error.message : "Could not load server" });
      return null;
    }
  },

  createServer: async (input) => {
    try {
      const detail = await serversApi.createServer(input);
      set((state) => ({
        servers: [...state.servers, detail.server],
        details: { ...state.details, [detail.server.id]: detail },
        activeServerId: detail.server.id,
        activeChannelId: detail.channels.find((c) => c.type === "text")?.id ?? null,
      }));
      return detail.server;
    } catch (error) {
      set({ error: error instanceof Error ? error.message : "Could not create server" });
      return null;
    }
  },

  createChannel: async (serverId, input) => {
    try {
      const channel = await channelsApi.createChannel(serverId, input);
      set((state) => {
        const detail = state.details[serverId];
        if (!detail) return state;
        return {
          details: {
            ...state.details,
            [serverId]: {
              ...detail,
              channels: [...detail.channels, channel].sort(compareChannels),
            },
          },
        };
      });
      return channel;
    } catch (error) {
      set({ error: error instanceof Error ? error.message : "Could not create channel" });
      return null;
    }
  },

  selectServer: (serverId) => {
    set({ activeServerId: serverId });
    void get().loadServer(serverId);
  },

  selectChannel: (channelId) => set({ activeChannelId: channelId }),

  removeServer: async (serverId) => {
    try {
      await serversApi.deleteServer(serverId);
      set((state) => {
        const details = { ...state.details };
        delete details[serverId];
        const servers = state.servers.filter((server) => server.id !== serverId);
        const wasActive = state.activeServerId === serverId;
        return {
          details,
          servers,
          activeServerId: wasActive ? (servers[0]?.id ?? null) : state.activeServerId,
          activeChannelId: wasActive ? null : state.activeChannelId,
        };
      });
    } catch (error) {
      set({ error: error instanceof Error ? error.message : "Could not delete server" });
    }
/* ------------------------------------------------------------------ */
/* Derived selectors                                                   */
/* ------------------------------------------------------------------ */

export function selectActiveDetail(state: ServersState): ServerDetail | null {
  return state.activeServerId ? state.details[state.activeServerId] ?? null : null;
}

export function selectChannels(state: ServersState): Channel[] {
  return sortChannels(selectActiveDetail(state));
}

export function selectMembers(state: ServersState): Member[] {
  return selectActiveDetail(state)?.members ?? [];
}

export function selectRoles(state: ServersState): Role[] {
  return selectActiveDetail(state)?.roles ?? [];
}

export function selectInvites(state: ServersState): Invite[] {
  return selectActiveDetail(state)?.invites ?? [];
}

export function selectActiveChannel(state: ServersState): Channel | null {
  if (!state.activeChannelId) return null;
  return selectChannels(state).find((channel) => channel.id === state.activeChannelId) ?? null;
}
  },
}));