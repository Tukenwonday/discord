import { useCallback } from "react";
import { useVoiceStore } from "@/stores/voice.store";
import { canPublishForChannel } from "@/lib/api/livekit";
import type { Channel } from "@/types";

export interface UseVoiceResult {
  channelId: string | null;
  roomName: string | null;
  state: ReturnType<typeof useVoiceStore.getState>["state"];
  isMuted: boolean;
  isDeafened: boolean;
  isScreenSharing: boolean;
  isCameraOn: boolean;
  participants: ReturnType<typeof useVoiceStore.getState>["participants"];
  error: string | null;
  join: (channel: Channel) => Promise<void>;
  leave: () => Promise<void>;
  toggleMic: () => Promise<void>;
  toggleCam: () => Promise<void>;
  toggleScreenShare: () => Promise<void>;
  toggleDeafen: (value: boolean) => void;
}

export function useVoice(): UseVoiceResult {
  const channelId = useVoiceStore((state) => state.channelId);
  const roomName = useVoiceStore((state) => state.roomName);
  const state = useVoiceStore((state) => state.state);
  const isMuted = useVoiceStore((state) => state.isMuted);
  const isDeafened = useVoiceStore((state) => state.isDeafened);
  const isScreenSharing = useVoiceStore((state) => state.isScreenSharing);
  const isCameraOn = useVoiceStore((state) => state.isCameraOn);
  const participants = useVoiceStore((state) => state.participants);
  const error = useVoiceStore((state) => state.error);
  const connect = useVoiceStore((state) => state.connect);
  const disconnect = useVoiceStore((state) => state.disconnect);
  const toggleMicAction = useVoiceStore((state) => state.toggleMic);
  const toggleCamAction = useVoiceStore((state) => state.toggleCam);
  const toggleScreenShareAction = useVoiceStore((state) => state.toggleScreenShare);
  const toggleDeafenAction = useVoiceStore((state) => state.toggleDeafen);

  const join = useCallback(
    async (channel: Channel) => {
      const roomName = `channel:${channel.id}`;
      const room = await connect(channel.id, roomName, "channel", canPublishForChannel(channel.type));
      if (!room) return;

      // Keep the participant list in step with LiveKit's own event stream.
      const sync = () => {
        useVoiceStore
          .getState()
          .setParticipants(Array.from(room.remoteParticipants.values()));
      };

      room.on("participantConnected", sync);
      room.on("participantDisconnected", sync);
      room.on("trackSubscribed", sync);
      room.on("trackUnsubscribed", sync);
      room.on("connectionStateChanged", (next) => {
        if (next === "connected") sync();
      });
      sync();
    },
    [connect],
  );

  const leave = useCallback(async () => {
    await disconnect();
  }, [disconnect]);

  return {
    channelId,
    roomName,
    state,
    isMuted,
    isDeafened,
    isScreenSharing,
    isCameraOn,
    participants,
    error,
    join,
    leave,
    toggleMic: toggleMicAction,
    toggleCam: toggleCamAction,
    toggleScreenShare: toggleScreenShareAction,
    toggleDeafen: toggleDeafenAction,
  };
}