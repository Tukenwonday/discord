import type * as React from "react";

import { GridLayout, ParticipantTile, useTracks } from "@livekit/components-react";
import { Track } from "livekit-client";
import "@livekit/components-styles";
import { CallControls } from "@/components/voice/CallControls";
import { useVoiceStore } from "@/stores/voice.store";

/**
 * Live video grid. Reconnection is handled by livekit-client itself, so this
 * component only reacts to track and participant changes.
 */
export function VideoCallGrid(): React.JSX.Element {
  const room = useVoiceStore((state) => state.room);
  const state = useVoiceStore((state) => state.state);

  const tracks = useTracks(
    [
      { source: Track.Source.Camera, withPlaceholder: true },
      { source: Track.Source.ScreenShare, withPlaceholder: false },
    ],
    { room },
  );

  if (!room || state !== "connected") {
    return (
      <div className="flex h-40 items-center justify-center bg-black/40 text-xs text-muted-foreground">
        {state === "connecting" ? "Connecting to the call…" : "No active call"}
      </div>
    );
  }

  return (
    <div className="flex h-64 shrink-0 flex-col border-b border-black/30">
      <div className="min-h-0 flex-1 p-2">
        <GridLayout
          tracks={tracks}
          className="h-full w-full"
          tileAspectRatio={16 / 9}
          tilePadding={4}
        >
          <ParticipantTile />
        </GridLayout>
      </div>

      <CallControls />
    </div>
  );
}
