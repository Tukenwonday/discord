import { useMemo, type React } from "react";
import { useQuery } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { MessageItem } from "@/components/chat/MessageItem";
import { MessageInput } from "@/components/chat/MessageInput";
import { TypingIndicator } from "@/components/chat/TypingIndicator";
import { useDmMessages, useMessageActions } from "@/hooks/useMessages";
import { useTyping } from "@/hooks/useTyping";
import { useSettingsStore } from "@/stores/settings.store";
import { listDmRecipients } from "@/lib/api/dms";
import { useAuth } from "@/hooks/useAuth";
import { avatarHue, initials } from "@/lib/utils";
import type { Channel } from "@/types";

export interface DmViewProps {
  dmId: string | null;
}

/**
 * A direct message conversation. Reuses the channel composer by handing it a
 * synthetic Channel whose id is the conversation id.
 */
export function DmView({ dmId }: DmViewProps): React.JSX.Element {
  const { user } = useAuth();
  const { messages, fetchOlder, hasMore, isFetchingOlder } = useDmMessages(dmId);
  const actions = useMessageActions(dmId);
  const { typingUsers, notifyTyping, notifyStopped } = useTyping(dmId);
  const sendOnEnter = useSettingsStore((state) => state.sendOnEnter);

  const recipients = useQuery({
    queryKey: ["dm-recipients", dmId],
    queryFn: () => listDmRecipients(dmId as string),
    enabled: Boolean(dmId),
  });

  const other = useMemo(
    () => (recipients.data ?? []).find((person) => person.id !== user?.id),
    [recipients.data, user],
  );

  const title = other ? other.displayName || other.username : "Direct message";

  // The composer only reads the id and name, so a channel shape is enough.
  const pseudoChannel = useMemo<Channel>(
    () => ({
      id: dmId ?? "",
      serverId: "",
      parentId: null,
      name: title,
      topic: "",
      type: "text",
      position: 0,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    }),
    [dmId, title],
  );

  if (!dmId) {
    return (
      <section className="flex h-full flex-1 items-center justify-center bg-cordis-chat">
        <p className="text-sm text-muted-foreground">Select a conversation to start chatting</p>
      </section>
    );
  }

  return (
    <section className="flex h-full min-w-0 flex-1 flex-col bg-cordis-chat">
      <header className="flex h-12 shrink-0 items-center gap-2 border-b border-black/20 px-4">
        {other ? (
          <Avatar className="h-6 w-6">
            {other.avatarUrl ? <AvatarImage src={other.avatarUrl} alt="" /> : null}
            <AvatarFallback
              style={{ backgroundColor: `hsl(${avatarHue(other.id)} 45% 35%)` }}
              className="text-[9px] text-white"
            >
              {initials(title)}
            </AvatarFallback>
          </Avatar>
        ) : null}
        <h1 className="truncate text-sm font-bold">{title}</h1>
      </header>

      <div className="cordis-scroll min-h-0 flex-1 overflow-y-auto px-4 py-2">
        {isFetchingOlder ? (
          <div className="flex justify-center py-3">
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
          </div>
        ) : null}

        {messages.map((message, index) => (
          <MessageItem
            key={message.id}
            message={message}
            previous={messages[index - 1]}
            compactMode={false}
          />
        ))}

        {messages.length === 0 && !isFetchingOlder ? (
          <p className="py-12 text-center text-sm text-muted-foreground">
            This is the beginning of your conversation with {title}.
          </p>
        ) : null}

        {hasMore ? (
          <div className="flex justify-center py-3">
            <button
              type="button"
              onClick={fetchOlder}
              className="text-xs text-muted-foreground hover:text-foreground"
            >
              Load earlier messages
            </button>
          </div>
        ) : null}
      </div>

      <TypingIndicator users={typingUsers(dmId)} />

      <MessageInput
        channel={pseudoChannel}
        sendOnEnter={sendOnEnter}
        onTyping={notifyTyping}
        onStopTyping={notifyStopped}
        onSend={(content) => {
          void actions.sendDm(dmId, content);
          notifyStopped(dmId);
        }}
      />
    </section>
  );
}