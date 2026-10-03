import { useCallback, useEffect, useRef, useState, type React } from "react";
import { Loader2 } from "lucide-react";
import { MessageItem } from "@/components/chat/MessageItem";
import { DateDivider } from "@/components/chat/DateDivider";
import { JumpToBottomButton } from "@/components/chat/JumpToBottomButton";
import { needsDayDivider } from "@/lib/utils";
import type { Message } from "@/types";

/** Distance from the top that triggers loading the previous page. */
const LOAD_THRESHOLD = 300;
/** Anything closer than this to the bottom counts as "at the bottom". */
const BOTTOM_THRESHOLD = 120;

export interface MessageListProps {
  messages: Message[];
  hasMore: boolean;
  isFetchingOlder: boolean;
  onLoadOlder: () => void;
  onJumpToLatest: () => void;
  compactMode: boolean;
}

/**
 * Scrollback with prepend-safe infinite loading: the scroll height is captured
 * before fetching and restored afterwards so older messages do not teleport the
 * viewport.
 */
export function MessageList({
  messages,
  hasMore,
  isFetchingOlder,
  onLoadOlder,
  onJumpToLatest,
  compactMode,
}: MessageListProps): React.JSX.Element {
  const containerRef = useRef<HTMLDivElement>(null);
  const [atBottom, setAtBottom] = useState(true);
  const previousScrollHeight = useRef<number>(0);
  const previousCount = useRef<number>(0);
  const wasAtBottom = useRef(true);

  // Track the bottom so new messages only auto-scroll when already there.
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const onScroll = () => {
      const distance = container.scrollHeight - container.scrollTop - container.clientHeight;
      wasAtBottom.current = distance <= BOTTOM_THRESHOLD;
      setAtBottom(wasAtBottom.current);

      if (container.scrollTop < LOAD_THRESHOLD && hasMore && !isFetchingOlder) {
        previousScrollHeight.current = container.scrollHeight;
        onLoadOlder();
      }
    };

    container.addEventListener("scroll", onScroll, { passive: true });
    return () => container.removeEventListener("scroll", onScroll);
  }, [hasMore, isFetchingOlder, onLoadOlder]);

  // After prepending a page, re-apply the saved offset so the view stays put.
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const grew = messages.length > previousCount.current;
    const appended = grew && previousScrollHeight.current > 0;

    if (appended) {
      const delta = container.scrollHeight - previousScrollHeight.current;
      container.scrollTop += delta;
      previousScrollHeight.current = 0;
    } else if (wasAtBottom.current && grew) {
      container.scrollTop = container.scrollHeight;
    }

    previousCount.current = messages.length;
  }, [messages]);

  // Jump to the newest message when the conversation changes.
  useEffect(() => {
    const container = containerRef.current;
    if (container) container.scrollTop = container.scrollHeight;
  }, [messages.length === 0]);

  const jumpToLatest = useCallback(() => {
    const container = containerRef.current;
    if (!container) return;
    container.scrollTop = container.scrollHeight;
    wasAtBottom.current = true;
    setAtBottom(true);
    onJumpToLatest();
  }, [onJumpToLatest]);

  return (
    <div className="relative min-h-0 flex-1">
      <div ref={containerRef} className="cordis-scroll h-full overflow-y-auto px-4">
        {isFetchingOlder ? (
          <div className="flex justify-center py-3">
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
          </div>
        ) : null}

        {messages.length === 0 && !isFetchingOlder ? (
          <p className="py-12 text-center text-sm text-muted-foreground">
            No messages yet. Say something to get things started.
          </p>
        ) : null}

        {messages.map((message, index) => {
          const previous = messages[index - 1];
          return (
            <div key={message.id}>
              {needsDayDivider(previous?.createdAt, message.createdAt) ? (
                <DateDivider iso={message.createdAt} />
              ) : null}
              <MessageItem
                message={message}
                previous={previous}
                compactMode={compactMode}
              />
            </div>
          );
        })}
      </div>

      <JumpToBottomButton atBottom={atBottom} onClick={jumpToLatest} />
    </div>
  );
}