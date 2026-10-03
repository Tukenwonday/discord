import { useEffect, useRef, useState, type React } from "react";
import { Paperclip, SendHorizontal, Smile, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { EmojiPicker } from "@/components/chat/EmojiPicker";
import { ReplyPreview } from "@/components/chat/ReplyPreview";
import { REPLY_EVENT } from "@/components/chat/MessageItem";
import { uploadFile, toAttachmentInput } from "@/lib/api/upload";
import { openFileDialog } from "@/lib/tauri/desktop";
import { errorMessage } from "@/lib/utils";
import type { AttachmentInput, Channel, Message } from "@/types";

/** Matches the backend's MAX_MESSAGE_LEN so we fail fast, not after a round trip. */
const MAX_LENGTH = 4000;

export interface MessageInputProps {
  channel: Channel;
  sendOnEnter: boolean;
  onTyping: (channelId: string) => void;
  onStopTyping: (channelId: string) => void;
  onSend: (content: string, replyToId?: string, attachments?: AttachmentInput[]) => void;
}

/** Composer with reply preview, emoji, uploads and drag-and-drop. */
export function MessageInput({
  channel,
  sendOnEnter,
  onTyping,
  onStopTyping,
  onSend,
}: MessageInputProps): React.JSX.Element {
  const [value, setValue] = useState("");
  const [replyTo, setReplyTo] = useState<Message | null>(null);
  const [attachments, setAttachments] = useState<AttachmentInput[]>([]);
  const [dragging, setDragging] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // Switching channels clears a draft that belongs to the previous one.
  useEffect(() => {
    setValue("");
    setReplyTo(null);
    setAttachments([]);
    setError(null);
  }, [channel.id]);

  useEffect(() => {
    const handler = (event: Event) => {
      const message = (event as CustomEvent<Message>).detail;
      setReplyTo(message);
      textareaRef.current?.focus();
    };
    window.addEventListener(REPLY_EVENT, handler);
    return () => window.removeEventListener(REPLY_EVENT, handler);
  }, []);

  function send(): void {
    const content = value.trim();
    if (!content && attachments.length === 0) return;
    if (content.length > MAX_LENGTH) {
      setError(`Messages are limited to ${MAX_LENGTH} characters`);
      return;
    }

    onSend(content, replyTo?.id, attachments.length > 0 ? attachments : undefined);
    setValue("");
    setReplyTo(null);
    setAttachments([]);
    setError(null);
    onStopTyping(channel.id);
  }

  function onKeyDown(event: React.KeyboardEvent<HTMLTextAreaElement>): void {
    if (event.key !== "Enter") return;
    const shouldSend = event.shiftKey ? !sendOnEnter : sendOnEnter;
    if (!shouldSend) return;
    event.preventDefault();
    send();
  }

  async function handleFiles(files: File[]): Promise<void> {
    if (files.length === 0) return;
    setUploading(true);
    setError(null);

    try {
      const uploaded = await Promise.all(files.map((file) => uploadFile(file)));
      setAttachments((current) => [...current, ...uploaded.map(toAttachmentInput)]);
    } catch (uploadError) {
      setError(errorMessage(uploadError, "Upload failed"));
    } finally {
      setUploading(false);
    }
  }

  async function pickFiles(): Promise<void> {
    const selected = await openFileDialog({ multiple: true });
    await handleFiles(
      selected.map((item) => item.file).filter((file): file is File => file !== null),
    );
  }
return (
    <div
      className="shrink-0 px-4 pb-5 pt-1"
      onDragOver={(event) => {
        event.preventDefault();
        setDragging(true);
      }}
      onDragLeave={() => setDragging(false)}
      onDrop={(event) => {
        event.preventDefault();
        setDragging(false);
        void handleFiles(Array.from(event.dataTransfer.files));
      }}
    >
      {replyTo ? (
        <div className="mb-1 flex items-center gap-2 rounded-t-lg bg-cordis-sidebar px-3 py-1.5">
          <ReplyPreview message={replyTo} />
          <button
            type="button"
            aria-label="Cancel reply"
            className="ml-auto text-muted-foreground hover:text-foreground"
            onClick={() => setReplyTo(null)}
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      ) : null}

      {attachments.length > 0 ? (
        <div className="mb-1 flex flex-wrap gap-1.5 rounded-t-lg bg-cordis-sidebar px-3 py-2">
          {attachments.map((attachment) => (
            <span
              key={attachment.url}
              className="flex items-center gap-1.5 rounded bg-black/20 px-2 py-1 text-xs"
            >
              {attachment.filename}
              <button
                type="button"
                aria-label={`Remove ${attachment.filename}`}
                onClick={() =>
                  setAttachments((current) =>
                    current.filter((item) => item.url !== attachment.url),
                  )
                }
                className="text-muted-foreground hover:text-foreground"
              >
                <X className="h-3 w-3" />
              </button>
            </span>
          ))}
        </div>
      ) : null}

      <div
        className={`flex items-end gap-1 rounded-lg bg-cordis-sidebar px-2 py-1.5 ${
          dragging ? "ring-2 ring-cordis-brand" : ""
        }`}
      >
        <Button
          variant="ghost"
          size="icon"
          aria-label="Attach a file"
          className="h-8 w-8 shrink-0 text-muted-foreground hover:text-foreground"
          disabled={uploading}
          onClick={() => void pickFiles()}
        >
          <Paperclip className="h-4 w-4" />
        </Button>

        <Popover>
          <PopoverTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              aria-label="Insert an emoji"
              className="h-8 w-8 shrink-0 text-muted-foreground hover:text-foreground"
            >
              <Smile className="h-4 w-4" />
            </Button>
          </PopoverTrigger>
          <PopoverContent side="top" className="w-64">
            <EmojiPicker
              onSelect={(emoji) => {
                setValue((current) => current + emoji);
                textareaRef.current?.focus();
              }}
            />
          </PopoverContent>
        </Popover>

        <textarea
          ref={textareaRef}
          rows={1}
          value={value}
          placeholder={`Message #${channel.name}`}
          onChange={(event) => {
            setValue(event.target.value);
            if (event.target.value) onTyping(channel.id);
            else onStopTyping(channel.id);
          }}
          onKeyDown={onKeyDown}
          className="selectable max-h-40 min-h-[20px] flex-1 resize-none bg-transparent py-1 text-[15px] text-foreground outline-none placeholder:text-muted-foreground"
        />

        <Button
          size="icon"
          aria-label="Send message"
          disabled={(!value.trim() && attachments.length === 0) || uploading}
          onClick={send}
          className="h-8 w-8 shrink-0 rounded bg-cordis-brand text-white hover:bg-cordis-brand-hover"
        >
          <SendHorizontal className="h-4 w-4" />
        </Button>
      </div>

      {error ? <p className="mt-1 text-xs text-destructive">{error}</p> : null}
    </div>
  );
}