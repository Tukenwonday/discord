import { useEffect, useState, type React } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import type { Message } from "@/types";

export interface EditMessageDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  message: Message;
  onSave: (content: string) => void;
}

/** Inline editor for the author's own message. */
export function EditMessageDialog({
  open,
  onOpenChange,
  message,
  onSave,
}: EditMessageDialogProps): React.JSX.Element {
  const [content, setContent] = useState(message.content);

  // Re-seed whenever a different message is opened for editing.
  useEffect(() => {
    if (open) setContent(message.content);
  }, [open, message.id, message.content]);

  function submit(): void {
    const trimmed = content.trim();
    if (!trimmed) return;
    onSave(trimmed);
    onOpenChange(false);
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Edit message</DialogTitle>
          <DialogDescription>
            Editing marks the message as edited for everyone.
          </DialogDescription>
        </DialogHeader>

        <Textarea
          autoFocus
          value={content}
          onChange={(event) => setContent(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter" && !event.shiftKey) {
              event.preventDefault();
              submit();
            }
          }}
          className="min-h-[100px]"
        />

        <DialogFooter>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={submit} disabled={!content.trim()}>
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}