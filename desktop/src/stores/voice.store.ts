import { create } from "zustand";
import { Room, type LocalTrackPublication, type RemoteParticipant } from "livekit-client";
import type { VoiceRoomState } from "@/types";
import { syncTrayState } from "@/lib/tauri/desktop";
import { fetchLiveKitToken } from "@/lib/api/livekit";

export interface VoiceState {
  channelId: string | null;
  roomName: string | null;
  roomType: "channel" | "dm" | null;
  state: VoiceRoomState;
  error: string | null;
  isMuted: boolean;
  isDeafened: boolean;
  isScreenSharing: boolean;
  isCameraOn: boolean;
  participants: RemoteParticipant[];
  /** The live room instance; kept outside React state on purpose. */
  room: Room | null;
  /** Fetches a LiveKit token and joins the room for this conversation. */
  connect: (
    channelId: string,
    roomName: string,
    roomType: "channel" | "dm",
    canPublish: boolean,
  ) => Promise<Room | null>;
  setRoom: (room: Room) => void;
  setState: (state: VoiceRoomState) => void;
  setError: (error: string | null) => void;
  setParticipants: (participants: RemoteParticipant[]) => void;
  toggleMic: () => Promise<void>;
  toggleCam: () => Promise<void>;
  toggleScreenShare: () => Promise<void>;
  toggleDeafen: (value: boolean) => void;
  disconnect: () => Promise<void>;
}

export const useVoiceStore = create<VoiceState>((set, get) => ({
  channelId: null,
  roomName: null,
  roomType: null,
  state: "disconnected",
  error: null,
  isMuted: false,
  isDeafened: false,
  isScreenSharing: false,
  isCameraOn: false,
  participants: [],
  room: null,

  /** Fetches a LiveKit token and joins; the caller supplies the room name. */
  connect: async (channelId, roomName, roomType, canPublish) => {
    set({ channelId, roomName, roomType, state: "connecting", error: null });
    try {
      const credentials = await fetchLiveKitToken({
        roomName,
        roomType,
        canPublish,
        canSubscribe: true,
      });

      const room = new Room({
        adaptiveStream: true,
        dynacast: true,
      });

      await room.connect(credentials.url, credentials.token);
      if (canPublish) {
        await room.localParticipant.setMicrophoneEnabled(true);
      }

      set({ room, state: "connected", isMuted: !canPublish, error: null });
      await syncTrayState({ muted: !canPublish });
      return room;
    } catch (error) {
      console.error("[voice] connect failed", error);
      set({
        state: "error",
        room: null,
        error: error instanceof Error ? error.message : "Could not join the voice channel",
      });
      return null;
    }
  },

  setRoom: (room) => set({ room, state: "connected" }),

  setState: (state) => set({ state }),

  setError: (error) => set({ error, state: error ? "error" : get().state }),

  setParticipants: (participants) => set({ participants }),

  toggleMic: async () => {
    const room = get().room;
    if (!room) return;
    try {
      const next = !room.localParticipant.isMicrophoneEnabled;
      await room.localParticipant.setMicrophoneEnabled(next);
      set({ isMuted: !next });
      await syncTrayState({ muted: !next });
    } catch (error) {
      console.error("[voice] toggleMic failed", error);
    }
  },

  toggleCam: async () => {
    const room = get().room;
    if (!room) return;
    try {
      const next = !room.localParticipant.isCameraEnabled;
      await room.localParticipant.setCameraEnabled(next);
      set({ isCameraOn: next });
    } catch (error) {
      console.error("[voice] toggleCam failed", error);
    }
  },

  toggleScreenShare: async () => {
    const room = get().room;
    if (!room) return;
    try {
      if (get().isScreenSharing) {
        for (const publication of room.localParticipant
          .getTrackPublications()
          .values()) {
          const source = publication.source;
          if (source === "screen_share" || source === "screen_share_audio") {
            await room.localParticipant.unpublishTrack(publication.track, true);
          }
        }
        set({ isScreenSharing: false });
        return;
      }

      await room.localParticipant.setScreenShareEnabled(true);
      set({ isScreenSharing: true });
    } catch (error) {
      console.error("[voice] toggleScreenShare failed", error);
    }
  },

  toggleDeafen: (value) => {
    set({ isDeafened: value });
    // Deafening implies muting, matching the tray's combined toggle.
    if (value) {
      void get().room?.localParticipant.setMicrophoneEnabled(false).then(() => {
        set({ isMuted: true });
        void syncTrayState({ muted: true, deafened: true });
      });
      return;
    }
    void syncTrayState({ deafened: false });
  },

  disconnect: async () => {
    const room = get().room;
    set({
      channelId: null,
      roomName: null,
      roomType: null,
      state: "disconnected",
      isMuted: false,
      isDeafened: false,
      isScreenSharing: false,
      isCameraOn: false,
      participants: [],
      room: null,
    });
    if (room) {
      try {
        await room.disconnect();
      } catch (error) {
        console.error("[voice] disconnect failed", error);
      }
    }
    await syncTrayState({ muted: false, deafened: false });
  },
}));

/** Screen-share publications currently published by the local participant. */
export function screenSharePublications(room: Room | null): LocalTrackPublication[] {
  if (!room) return [];
  return Array.from(room.localParticipant.getTrackPublications().values()).filter(
    (publication) =>
      publication.source === "screen_share" || publication.source === "screen_share_audio",
  );
}

export function selectIsConnected(state: VoiceState): boolean {
  return state.state === "connected" && state.room !== null;
}