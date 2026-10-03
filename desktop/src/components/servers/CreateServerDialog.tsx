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
import { Textarea } from "@/components/ui/textarea";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useServersStore } from "@/stores/servers.store";
import { useNavigate } from "react-router-dom";

export interface CreateServerDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/** Creates a server; the backend seeds @everyone, general and voice. */
export function CreateServerDialog({
  open,
  onOpenChange,
}: CreateServerDialogProps): React.JSX.Element {
  const createServer = useServersStore((state) => state.createServer);
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function onSubmit(event: React.FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    const trimmed = name.trim();
    if (trimmed.length < 2) {
      setError("Give your server a name");
      return;
    }

    setSubmitting(true);
    setError(null);
    const server = await createServer({
      name: trimmed,
      ...(description.trim() ? { description: description.trim() } : {}),
    });
    setSubmitting(false);

    if (!server) {
      setError("Could not create the server. Please try again.");
      return;
    }

    setName("");
    setDescription("");
    onOpenChange(false);
    navigate(`/channels/${useServersStore.getState().activeChannelId ?? ""}`);
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Plus className="h-5 w-5 text-cordis-brand" />
            Create a server
          </DialogTitle>
          <DialogDescription>
            Give your server a name with an image and personality.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={(event) => void onSubmit(event)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="server-name">Server name</Label>
            <Input
              id="server-name"
              autoFocus
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="Cordis HQ"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="server-description">Description</Label>
            <Textarea
              id="server-description"
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              placeholder="What is this server about?"
              className="min-h-[70px]"
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
              Create
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}