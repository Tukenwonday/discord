import { useMemo, useState, type React } from "react";

export interface EmojiPickerProps {
  onSelect: (emoji: string) => void;
}

const CATEGORIES: { name: string; emojis: string[] }[] = [
  {
    name: "Smileys",
    emojis: [
      "😀", "😃", "😄", "😁", "😆", "😅", "🤣", "😂", "🙂", "🙃",
      "😉", "😊", "😇", "🥰", "😍", "🤩", "😘", "😋", "😜", "🤔",
      "🤗", "🤭", "🫡", "🤫", "😴", "😌", "😔", "😪", "🤤", "😎",
    ],
  },
  {
    name: "Gestures",
    emojis: [
      "👍", "👎", "👌", "✌️", "🤞", "🤟", "🤘", "👏", "🙌", "🤝",
      "🙏", "💪", "👋", "🫶", "✊", "👊", "🖐️", "🤙", "🫰", "☝️",
    ],
  },
  {
    name: "Hearts",
    emojis: [
      "❤️", "🧡", "💛", "💚", "💙", "💜", "🖤", "🤍", "🤎", "💔",
      "❣️", "💕", "💞", "💓", "💗", "💖", "💘", "💝", "💟", "♥️",
    ],
  },
  {
    name: "Objects",
    emojis: [
      "🎉", "🎊", "🎈", "🎁", "🏆", "🥇", "⭐", "🌟", "✨", "⚡",
      "🔥", "💯", "✅", "❌", "❓", "❗", "💡", "📌", "🔒", "🔑",
    ],
  },
  {
    name: "Animals",
    emojis: [
      "🐶", "🐱", "🐭", "🐹", "🐰", "🦊", "🐻", "🐼", "🐨", "🐯",
      "🦁", "🐮", "🐷", "🐸", "🐵", "🐔", "🐧", "🐦", "🦄", "🐝",
    ],
  },
];

/**
 * A plain popover grid of unicode emoji. Rendering them directly avoids a
 * multi-megabyte emoji package while covering the common set.
 */
export function EmojiPicker({ onSelect }: EmojiPickerProps): React.JSX.Element {
  const [category, setCategory] = useState(CATEGORIES[0].name);

  const emojis = useMemo(
    () => CATEGORIES.find((entry) => entry.name === category)?.emojis ?? [],
    [category],
  );

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap gap-1">
        {CATEGORIES.map((entry) => (
          <button
            key={entry.name}
            type="button"
            onClick={() => setCategory(entry.name)}
            className={`rounded px-1.5 py-0.5 text-[10px] transition ${
              entry.name === category
                ? "bg-secondary text-foreground"
                : "text-muted-foreground hover:bg-accent"
            }`}
          >
            {entry.name}
          </button>
        ))}
      </div>

      <div className="grid max-h-40 grid-cols-10 gap-0.5 overflow-y-auto">
        {emojis.map((emoji) => (
          <button
            key={emoji}
            type="button"
            onClick={() => onSelect(emoji)}
            className="flex h-7 w-7 items-center justify-center rounded text-lg transition hover:bg-accent"
          >
            {emoji}
          </button>
        ))}
      </div>
    </div>
  );
}