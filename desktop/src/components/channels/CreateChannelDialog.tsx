import { useState, type React } from "react";
import { Loader2, Plus } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useServersStore } from "@/stores/servers.store";
import type { ChannelType } from "@/types";

export interface CreateChannelDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  serverId: string;
}

const NAME_PATTERN = /^[a-z0-9-_]{1,64}$/;

/** Channel names follow the contract: lowercase slug, 1 to 64 characters. */
export function CreateChannelDialog({
  open,
  onOpenChange,
  serverId,
}: CreateChannelDialogProps): React.JSX.Element {
  const createChannel = useServersStore((state) => state.createChannel);
  const [name, setName] = useState("");
  const [topic, setTopic] = useState("");
  const [type, setType] = useState<ChannelType>("text");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function onSubmit(event: React.FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    const slug = name.trim().toLowerCase().replace(/\s+/g, "-");

    if (!NAME_PATTERN.test(slug)) {
      setError("Use 1 to 64 lowercase letters, numbers, hyphens or underscores");
      return;
    }
    if (!serverId) {
      setError("Select a server first");
      return;
    }

    setSubmitting(true);
    setError(null);
    const channel = await createChannel(serverId, { name: slug, type, topic: topic.trim() });
    setSubmitting(false);

    if (!channel) {
      setError("Could not create the channel. Please try again.");
      return;
    }

    setName("");
    setTopic("");
    setType("text");
    onOpenChange(false);
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Plus className="h-5 w-5 text-cordis-brand" />
            Create a channel
          </DialogTitle>
          <DialogDescription>Channels are where conversations happen.</DialogDescription>
        </DialogHeader>

        <form onSubmit={(event) => void onSubmit(event)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="channel-type">Channel type</Label>
            <Select
              value={type}
              onValueChange={(value) => setType(value as ChannelType)}
            >
              <SelectTrigger id="channel-type">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="text">Text</SelectItem>
                <SelectItem value="voice">Voice</SelectItem>
                <SelectItem value="video">Video</SelectItem>
                <SelectItem value="category">Category</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="channel-name">Channel name</Label>
            <Input
              id="channel-name"
              autoFocus
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="general"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="channel-topic">Channel topic</Label>
            <Input
              id="channel-topic"
              value={topic}
              onChange={(event) => setTopic(event.target.value)}
              placeholder="What is this channel about?"
            />
          </div>

          {error ? (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          ) : null}

          <DialogFooter>
            <Button type="button" variant="secondary" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={submitting}>
              {submitting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
              Create channel
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}