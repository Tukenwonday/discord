import { post } from "./client";
import type { ChannelType, LiveKitTokenInput, LiveKitTokenResponse } from "@/types";

export function fetchLiveKitToken(input: LiveKitTokenInput): Promise<LiveKitTokenResponse> {
  return post<LiveKitTokenResponse>("/api/livekit/token", input);
}

/** Voice and video channels may publish; text channels are listen-only. */
export function canPublishForChannel(type: ChannelType): boolean {
  return type === "voice" || type === "video";
}