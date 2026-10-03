import type * as React from "react";

import { Mic, MicOff, Headphones, LogOut, ScreenShare, ScreenShareOff, Video, VideoOff } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useVoice } from "@/hooks/useVoice";

export { VoicePanel } from "@/components/voice/VoicePanel";

/** Mic, camera, screen-share, deafen and hang-up for a live call. */
export function CallControls(): React.JSX.Element | null {
  const voice = useVoice();
  if (voice.state !== "connected") return null;

  return (
    <div className="flex items-center justify-center gap-2 border-t border-black/30 bg-cordis-sidebar px-4 py-3">
      <ControlButton
        active={!voice.isMuted}
        label={voice.isMuted ? "Unmute" : "Mute"}
        onClick={() => void voice.toggleMic()}
      >
        {voice.isMuted ? <MicOff className="h-4 w-4" /> : <Mic className="h-4 w-4" />}
      </ControlButton>

      <ControlButton
        active={voice.isCameraOn}
        label={voice.isCameraOn ? "Turn camera off" : "Turn camera on"}
        onClick={() => void voice.toggleCam()}
      >
        {voice.isCameraOn ? <Video className="h-4 w-4" /> : <VideoOff className="h-4 w-4" />}
      </ControlButton>

      <ControlButton
        active={voice.isScreenSharing}
        label={voice.isScreenSharing ? "Stop sharing" : "Share screen"}
        onClick={() => void voice.toggleScreenShare()}
      >
        {voice.isScreenSharing ? (
          <ScreenShareOff className="h-4 w-4" />
        ) : (
          <ScreenShare className="h-4 w-4" />
        )}
      </ControlButton>

      <ControlButton
        active={!voice.isDeafened}
        label={voice.isDeafened ? "Undeafen" : "Deafen"}
        onClick={() => voice.toggleDeafen(!voice.isDeafened)}
      >
        <Headphones className="h-4 w-4" />
      </ControlButton>

      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            size="icon"
            aria-label="Leave the call"
            onClick={() => void voice.leave()}
            className="h-9 w-9 rounded-full bg-destructive text-destructive-foreground hover:bg-destructive/90"
          >
            <LogOut className="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>Leave</TooltipContent>
      </Tooltip>
    </div>
  );
}

function ControlButton({
  active,
  label,
  onClick,
  children,
}: {
  active: boolean;
  label: string;
  onClick: () => void;
  children: React.ReactNode;
}): React.JSX.Element {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          size="icon"
          aria-label={label}
          onClick={onClick}
          className={`h-9 w-9 rounded-full ${
            active ? "bg-black/30 text-foreground" : "bg-destructive text-destructive-foreground"
          }`}
        >
          {children}
        </Button>
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}