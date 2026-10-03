import { useState, type React } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useVoice } from "@/hooks/useVoice";
import { useServersStore } from "@/stores/servers.store";
import { useVoiceStore } from "@/stores/voice.store";
import type { Channel } from "@/types";

export interface JoinVoiceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  channel: Channel | null;
}

/** Requests a LiveKit token and joins the channel's room. */
export function JoinVoiceDialog({
  open,
  onOpenChange,
  channel,
}: JoinVoiceDialogProps): React.JSX.Element {
  const voice = useVoice();
  const loadServer = useServersStore((state) => state.loadServer);
  const activeServerId = useServersStore((state) => state.activeServerId);
  const [joining, setJoining] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function join(): Promise<void> {
    if (!channel) return;
    setJoining(true);
    setError(null);

    await voice.join(channel);

    // The store is the source of truth after the attempt settles.
    const state = useVoiceStore.getState();
    setJoining(false);

    if (state.state !== "connected") {
      setError(state.error ?? "Could not join the voice channel");
      return;
    }

    onOpenChange(false);
    if (activeServerId) await loadServer(activeServerId);
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Join {channel?.name ?? "voice channel"}</DialogTitle>
          <DialogDescription>
            You will join with your microphone enabled. You can mute at any time.
          </DialogDescription>
        </DialogHeader>

        {error ? (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : null}

        <DialogFooter>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={() => void join()} disabled={joining}>
            {joining ? "Connecting…" : "Join"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}