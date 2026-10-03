import type * as React from "react";

import { ChannelHeader } from "@/components/channels/ChannelHeader";
import { MessageList } from "@/components/chat/MessageList";
import { MessageInput } from "@/components/chat/MessageInput";
import { TypingIndicator } from "@/components/chat/TypingIndicator";
import { VideoCallGrid } from "@/components/voice/VideoCallGrid";
import { useServers } from "@/hooks/useServers";
import { useMessageActions, useMessages } from "@/hooks/useMessages";
import { useTyping } from "@/hooks/useTyping";
import { useVoice } from "@/hooks/useVoice";
import { useSettingsStore } from "@/stores/settings.store";
import type { Channel } from "@/types";

export interface ChatViewProps {
  channel: Channel | null;
}

/** The message column: header, scrollback, typing row and composer. */
export function ChatView({ channel }: ChatViewProps): React.JSX.Element {
  const { activeChannelId } = useServers();
  const { messages, fetchOlder, hasMore, isFetchingOlder, loadNewest } = useMessages(activeChannelId);
  const actions = useMessageActions(activeChannelId);
  const { typingUsers, notifyTyping, notifyStopped } = useTyping(activeChannelId);
  const sendOnEnter = useSettingsStore((state) => state.sendOnEnter);
  const compactMode = useSettingsStore((state) => state.compactMode);
  const voice = useVoice();

  const inVoiceChannel =
    voice.channelId === channel?.id && (channel.type === "voice" || channel.type === "video");

  if (!channel) {
    return (
      <section className="flex h-full flex-1 items-center justify-center bg-cordis-chat">
        <p className="text-sm text-muted-foreground">Select a channel to start chatting</p>
      </section>
    );
  }

  return (
    <section className="flex h-full min-w-0 flex-1 flex-col bg-cordis-chat">
      <ChannelHeader channel={channel} />

      {inVoiceChannel && channel.type === "video" ? <VideoCallGrid /> : null}

      <MessageList
        messages={messages}
        hasMore={hasMore}
        isFetchingOlder={isFetchingOlder}
        onLoadOlder={fetchOlder}
        onJumpToLatest={loadNewest}
        compactMode={compactMode}
      />

      <TypingIndicator users={typingUsers(activeChannelId)} />

      <MessageInput
        channel={channel}
        sendOnEnter={sendOnEnter}
        onTyping={notifyTyping}
        onStopTyping={notifyStopped}
        onSend={(content, replyToId, attachments) => {
          void actions.send({ content, replyToId, attachments });
          notifyStopped(activeChannelId ?? "");
        }}
      />
    </section>
  );
}