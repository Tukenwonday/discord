import type * as React from "react";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { avatarHue, initials } from "@/lib/utils";
import type { Message } from "@/types";

export interface ReplyPreviewProps {
  message: Message;
}

/** The quoted line shown above a reply's own body. */
export function ReplyPreview({ message }: ReplyPreviewProps): React.JSX.Element {
  const author = message.author.displayName || message.author.username;

  return (
    <div className="mt-1 flex items-center gap-2 rounded bg-black/20 px-2 py-1 text-xs text-muted-foreground">
      <Avatar className="h-4 w-4">
        {message.author.avatarUrl ? (
          <AvatarImage src={message.author.avatarUrl} alt="" />
        ) : null}
        <AvatarFallback
          style={{ backgroundColor: `hsl(${avatarHue(message.author.id)} 45% 35%)` }}
          className="text-[8px] text-white"
        >
          {initials(author)}
        </AvatarFallback>
      </Avatar>

      <span className="font-semibold text-foreground/80">{author}</span>
      <span className="truncate">{message.content || "Click to see the attachment"}</span>
    </div>
  );
}