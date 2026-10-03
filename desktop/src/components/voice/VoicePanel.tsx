import type * as React from "react";

import { MicOff, LogOut } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { useVoice } from "@/hooks/useVoice";
import { avatarHue, initials } from "@/lib/utils";

/** Bottom-of-sidebar summary of the active voice channel. */
export function VoicePanel(): React.JSX.Element | null {
  const voice = useVoice();
  if (!voice.channelId || !voice.roomName) return null;

  return (
    <div className="shrink-0 border-t border-black/30 bg-black/20 px-3 py-2">
      <div className="mb-1 flex items-center gap-1.5 text-[11px] text-cordis-online">
        <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-cordis-online" />
        Voice connected
      </div>

      <div className="space-y-1">
        {voice.participants.map((participant) => (
          <div key={participant.identity} className="flex items-center gap-2">
            <Avatar className="h-5 w-5">
              {participant.avatar ? <AvatarImage src={participant.avatar} alt="" /> : null}
              <AvatarFallback
                style={{
                  backgroundColor: `hsl(${avatarHue(participant.identity ?? "x")} 45% 35%)`,
                }}
                className="text-[8px] text-white"
              >
                {initials(participant.name ?? participant.identity ?? "?")}
              </AvatarFallback>
            </Avatar>
            <span className="truncate text-[11px] text-muted-foreground">
              {participant.name ?? participant.identity}
            </span>
            {participant.isMicrophoneEnabled ? null : (
              <MicOff className="ml-auto h-3 w-3 text-cordis-dnd" />
            )}
          </div>
        ))}
      </div>

      <Button
        variant="ghost"
        size="sm"
        onClick={() => void voice.leave()}
        className="mt-1 h-6 w-full justify-start gap-1.5 px-1 text-[11px] text-destructive hover:bg-destructive/10"
      >
        <LogOut className="h-3 w-3" />
        Disconnect
      </Button>
    </div>
  );
}

