import { ArrowDown } from "lucide-react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface JumpToBottomButtonProps {
  /** Whether the scroll container is already resting on the newest message. */
  atBottom: boolean;
  /** Number of messages that arrived while the user was scrolled away. */
  unseenCount?: number;
  onClick: () => void;
}

/**
 * Floating affordance that returns the reader to the newest message. It hides
 * itself while the list is already pinned to the bottom so it never covers a
 * message the user is currently reading.
 */
export function JumpToBottomButton({
  atBottom,
  unseenCount = 0,
  onClick,
}: JumpToBottomButtonProps) {
  return (
    <Button
      variant="secondary"
      size="icon"
      aria-label="Jump to latest messages"
      title="Jump to latest"
      onClick={onClick}
      data-testid="jump-to-bottom"
      className={cn(
        "absolute bottom-4 right-6 h-9 w-9 rounded-full shadow-lg transition-all",
        "bg-secondary/90 text-secondary-foreground backdrop-blur hover:bg-accent",
        atBottom ? "pointer-events-none scale-0 opacity-0" : "scale-100 opacity-100",
      )}
    >
      <ArrowDown className="h-4 w-4" />
      {unseenCount > 0 ? (
        <span className="absolute -right-1 -top-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold text-primary-foreground">
          {unseenCount > 99 ? "99+" : unseenCount}
        </span>
      ) : null}
    </Button>
  );
}

export default JumpToBottomButton;
