import { useQuery } from "@tanstack/react-query";
import { fetchLiveKitToken } from "@/lib/api/livekit";
import type { LiveKitRoomType, LiveKitTokenResponse } from "@/types";

export interface UseLiveKitTokenResult {
  token: LiveKitTokenResponse | null;
  isLoading: boolean;
  error: string | null;
  enabled: boolean;
}

/**
 * Requests a short-lived LiveKit token. Tokens expire after an hour, so the
 * query is disabled until a room is actually needed and refetched on demand.
 */
export function useLiveKitToken(
  roomName: string | null,
  roomType: LiveKitRoomType,
  canPublish: boolean,
): UseLiveKitTokenResult {
  const enabled = Boolean(roomName);

  const query = useQuery({
    queryKey: ["livekit-token", roomName, roomType, canPublish],
    enabled,
    queryFn: () =>
      fetchLiveKitToken({
        roomName: roomName as string,
        roomType,
        canPublish,
        canSubscribe: true,
      }),
    staleTime: 5 * 60 * 1000,
    retry: false,
  });

  return {
    token: query.data ?? null,
    isLoading: query.isLoading,
    error: query.error?.message ?? null,
    enabled,
  };
}