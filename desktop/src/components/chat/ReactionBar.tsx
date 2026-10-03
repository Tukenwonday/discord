import { useState, type React } from "react";
import { Plus, Smile } from "lucide-react";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Input } from "@/components/ui/input";
import { EmojiPicker } from "@/components/chat/EmojiPicker";
import { useAuth } from "@/hooks/useAuth";
import { cn } from "@/lib/utils";
import type { Reaction } from "@/types";

export interface ReactionBarProps {
  reactions: Reaction[];
  onToggle: (emoji: string) => void;
}

const QUICK_EMOJI = ["👍", "❤️", "😂", "🎉", "👀", "🔥"];

/** Existing reactions as toggleable pills, plus a picker for more. */
export function ReactionBar({ reactions, onToggle }: ReactionBarProps): React.JSX.Element {
  const user = useAuth();
  const [custom, setCustom] = useState("");

  // Collapse duplicates into one pill carrying a count, as Discord does.
  const counts = new Map<string, { count: number; mine: boolean }>();
  for (const reaction of reactions) {
    const entry = counts.get(reaction.emoji) ?? { count: 0, mine: false };
    entry.count += 1;
    if (reaction.userId === user?.id) entry.mine = true;
    counts.set(reaction.emoji, entry);
  }

  return (
    <div className="mt-1 flex flex-wrap items-center gap-1">
      {Array.from(counts.entries()).map(([emoji, entry]) => (
        <button
          key={emoji}
          type="button"
          onClick={() => onToggle(emoji)}
          aria-label={`${emoji} ${entry.count}`}
          className={cn(
            "flex items-center gap-1 rounded-md border px-1.5 py-0.5 text-xs transition",
            entry.mine
              ? "border-cordis-brand bg-cordis-brand/20"
              : "border-white/10 bg-black/20 hover:border-white/20",
          )}
        >
          <span>{emoji}</span>
          <span className="text-[10px] text-muted-foreground">{entry.count}</span>
        </button>
      ))}

      <Popover>
        <PopoverTrigger asChild>
          <button
            type="button"
            aria-label="Add a reaction"
            className="flex h-6 w-6 items-center justify-center rounded-md border border-dashed border-white/20 text-muted-foreground transition hover:border-white/40 hover:text-foreground"
          >
            <Smile className="h-3 w-3" />
          </button>
        </PopoverTrigger>
        <PopoverContent align="start" className="w-64 p-2">
          <div className="mb-2 flex flex-wrap gap-1">
            {QUICK_EMOJI.map((emoji) => (
              <button
                key={emoji}
                type="button"
                onClick={() => onToggle(emoji)}
                className="flex h-8 w-8 items-center justify-center rounded text-lg transition hover:bg-accent"
              >
                {emoji}
              </button>
            ))}
          </div>

          <EmojiPicker onSelect={onToggle} />

          <form
            className="mt-2 flex items-center gap-1"
            onSubmit={(event) => {
              event.preventDefault();
              const trimmed = custom.trim();
              // The backend accepts 1 to 16 unicode runes.
              if (!trimmed || Array.from(trimmed).length > 16) return;
              onToggle(trimmed);
              setCustom("");
            }}
          >
            <Input
              value={custom}
              onChange={(event) => setCustom(event.target.value)}
              placeholder="Custom emoji"
              className="h-8 text-xs"
            />
            <button
              type="submit"
              aria-label="Add custom emoji"
              className="flex h-8 w-8 items-center justify-center rounded bg-secondary hover:bg-accent"
            >
              <Plus className="h-3.5 w-3.5" />
            </button>
          </form>
        </PopoverContent>
      </Popover>
    </div>
  );
}
