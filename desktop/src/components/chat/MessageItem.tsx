import { useState, type React } from "react";
import { MoreHorizontal, Reply, Pencil, Trash2 } from "lucide-react";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ReactionBar } from "@/components/chat/ReactionBar";
import { AttachmentList } from "@/components/chat/AttachmentList";
import { ReplyPreview } from "@/components/chat/ReplyPreview";
import { EditMessageDialog } from "@/components/chat/EditMessageDialog";
import { ConfirmDialog } from "@/components/common/ConfirmDialog";
import { useAuth } from "@/hooks/useAuth";
import { useMessageActions } from "@/hooks/useMessages";
import { usePresence } from "@/hooks/usePresence";
import { useServersStore } from "@/stores/servers.store";
import { avatarHue, cn, formatMessageTime, initials, shouldGroup } from "@/lib/utils";
import type { Message } from "@/types";

export interface MessageItemProps {
  message: Message;
  previous?: Message;
  compactMode: boolean;
}

/** One row: avatar, author, content, attachments and the hover actions. */
export function MessageItem({
  message,
  previous,
  compactMode,
}: MessageItemProps): React.JSX.Element {
  const { user } = useAuth();
  const activeChannelId = useServersStore((state) => state.activeChannelId);
  const actions = useMessageActions(activeChannelId);
  const { statusOf } = usePresence();
  const [editOpen, setEditOpen] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);

  const isMine = user?.id === message.author.id;
  const grouped = shouldGroup(previous, message);

  return (
    <div
      className={cn(
        "group relative px-2 py-0.5 transition-colors hover:bg-black/20",
        grouped && "py-0",
        !compactMode && !grouped && "mt-2",
      )}
    >
      <div className="flex gap-3">
        <div className="w-10 shrink-0 pt-0.5">
          {grouped ? (
            <span className="block text-[10px] leading-5 text-transparent group-hover:text-muted-foreground">
              {formatMessageTime(message.createdAt)}
            </span>
          ) : (
            <Avatar className="h-10 w-10">
              {message.author.avatarUrl ? (
                <AvatarImage src={message.author.avatarUrl} alt="" />
              ) : null}
              <AvatarFallback
                style={{ backgroundColor: `hsl(${avatarHue(message.author.id)} 45% 35%)` }}
                className="text-xs font-semibold text-white"
              >
                {initials(message.author.displayName || message.author.username)}
              </AvatarFallback>
            </Avatar>
          )}
        </div>

        <div className="min-w-0 flex-1">
          {!grouped ? (
            <div className="flex items-baseline gap-2">
              <span className="text-sm font-semibold text-foreground">
                {message.author.displayName || message.author.username}
              </span>
              <span className="text-[11px] text-muted-foreground">
                {formatMessageTime(message.createdAt)}
              </span>
              {statusOf(message.author.id) === "dnd" ? (
                <span className="text-[10px] uppercase text-cordis-dnd">Do not disturb</span>
              ) : null}
            </div>
          ) : null}

          {message.replyTo ? <ReplyPreview message={message.replyTo} /> : null}

          {message.content ? (
            <p className="selectable whitespace-pre-wrap break-words text-[15px] leading-relaxed text-foreground/90">
              {message.content}
            </p>
          ) : null}

          <AttachmentList attachments={message.attachments} />

          {message.reactions.length > 0 ? (
            <ReactionBar
              messageId={message.id}
              reactions={message.reactions}
              onToggle={(emoji) => void actions.toggleReaction(message.id, emoji)}
            />
          ) : null}

          {message.editedAt ? (
            <span className="text-[10px] text-muted-foreground/70">(edited)</span>
          ) : null}
        </div>
      </div>
<div className="absolute -top-2 right-4 hidden items-center rounded border border-black/30 bg-cordis-chat shadow group-hover:flex">
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              aria-label="React with a thumbs up"
              className="h-7 w-7 text-muted-foreground hover:text-foreground"
              onClick={() => void actions.toggleReaction(message.id, "👍")}
            >
              <span className="text-sm">👍</span>
            </Button>
          </TooltipTrigger>
          <TooltipContent>React</TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              aria-label="Reply"
              className="h-7 w-7 text-muted-foreground hover:text-foreground"
              onClick={() => emitReplyTarget(message)}
            >
              <Reply className="h-3.5 w-3.5" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Reply</TooltipContent>
        </Tooltip>

        {isMine ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                aria-label="Edit message"
                className="h-7 w-7 text-muted-foreground hover:text-foreground"
                onClick={() => setEditOpen(true)}
              >
                <Pencil className="h-3.5 w-3.5" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Edit</TooltipContent>
          </Tooltip>
        ) : null}

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              aria-label="More actions"
              className="h-7 w-7 text-muted-foreground hover:text-foreground"
            >
              <MoreHorizontal className="h-3.5 w-3.5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {isMine ? (
              <DropdownMenuItem onSelect={() => setEditOpen(true)}>
                <Pencil className="h-4 w-4" /> Edit
              </DropdownMenuItem>
            ) : null}
            <DropdownMenuItem
              className="text-destructive"
              onSelect={() => setConfirmOpen(true)}
            >
              <Trash2 className="h-4 w-4" /> Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <EditMessageDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        message={message}
        onSave={(content) => void actions.edit(message.id, content)}
      />

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title="Delete message?"
        description="This cannot be undone."
        confirmLabel="Delete"
        destructive
        onConfirm={() => {
          setConfirmOpen(false);
          void actions.remove(message.id);
        }}
      />
    </div>
  );
}

/**
 * Reply targets cross component boundaries, so the composer listens for this
 * window event instead of threading a callback down through the message list.
 */
export const REPLY_EVENT = "cordis:reply";

export function emitReplyTarget(message: Message): void {
  window.dispatchEvent(new CustomEvent<Message>(REPLY_EVENT, { detail: message }));
}
