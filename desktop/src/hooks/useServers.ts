import { useEffect, useMemo } from "react";
import { useNavigate, useParams } from "react-router-dom";
import {
  selectActiveDetail,
  sortChannels,
  useServersStore,
} from "@/stores/servers.store";
import type { Channel, ChannelType, Invite, Member, Role, Server, ServerDetail } from "@/types";

export interface UseServersResult {
  servers: Server[];
  detail: ServerDetail | null;
  channels: Channel[];
  members: Member[];
  roles: Role[];
  invites: Invite[];
  activeServerId: string | null;
  activeChannelId: string | null;
  activeChannel: Channel | null;
  loading: boolean;
  error: string | null;
  selectServer: (serverId: string) => void;
  selectChannel: (channelId: string) => void;
  refresh: () => Promise<void>;
}

export function useServers(): UseServersResult {
  const navigate = useNavigate();
  const params = useParams<{ channelId?: string }>();

  const servers = useServersStore((state) => state.servers);
  const activeServerId = useServersStore((state) => state.activeServerId);
  const activeChannelId = useServersStore((state) => state.activeChannelId);
  const loading = useServersStore((state) => state.loading);
  const error = useServersStore((state) => state.error);
  const loadServers = useServersStore((state) => state.loadServers);
  const loadServer = useServersStore((state) => state.loadServer);
  const selectServerAction = useServersStore((state) => state.selectServer);
  const selectChannelAction = useServersStore((state) => state.selectChannel);

  // Only the detail object is subscribed directly: it is a stable reference in
  // the store. The collections below are derived from it with useMemo, because
  // a selector returning a fresh array on every call would loop React forever.
  const detail = useServersStore(selectActiveDetail);

  const channels = useMemo(() => sortChannels(detail), [detail]);
  const members = useMemo(() => detail?.members ?? [], [detail]);
  const roles = useMemo(() => detail?.roles ?? [], [detail]);
  const invites = useMemo(() => detail?.invites ?? [], [detail]);
  const activeChannel = useMemo(
    () => channels.find((channel) => channel.id === activeChannelId) ?? null,
    [channels, activeChannelId],
  );

  useEffect(() => {
    void loadServers();
  }, [loadServers]);

  // The URL is the source of truth for the open channel.
  useEffect(() => {
    const channelId = params.channelId;
    if (channelId && channelId !== activeChannelId) {
      selectChannelAction(channelId);
    }
  }, [params.channelId, activeChannelId, selectChannelAction]);

  /** Selecting a server opens its first text channel so the URL always fits. */
  const selectServer = (serverId: string) => {
    selectServerAction(serverId);
    const detail = useServersStore.getState().details[serverId];
    const firstText = detail?.channels.find((channel) => channel.type === "text");
    navigate(firstText ? `/channels/${firstText.id}` : "/channels");
  };

  const selectChannel = (channelId: string) => {
    selectChannelAction(channelId);
    navigate(`/channels/${channelId}`);
  };

  const refresh = useMemo(
    () => async () => {
      await loadServers();
      const current = useServersStore.getState().activeServerId;
      if (current) await loadServer(current);
    },
    [loadServers, loadServer],
  );

  return {
    servers,
    detail,
    channels,
    members,
    roles,
    invites,
    activeServerId,
    activeChannelId,
    activeChannel,
    loading,
    error,
    selectServer,
    selectChannel,
    refresh,
  };
}

/** Channels visible in the sidebar, minus the category containers. */
export function useVisibleChannels(channels: Channel[]): Channel[] {
  return useMemo(() => channels.filter((channel) => channel.type !== "category"), [channels]);
}

export function useCreateChannel(): (
  serverId: string,
  input: { name: string; type?: ChannelType; topic?: string },
) => Promise<Channel | null> {
  const createChannel = useServersStore((state) => state.createChannel);
  return useMemo(
    () => (serverId, input) => createChannel(serverId, input),
    [createChannel],
  );
}