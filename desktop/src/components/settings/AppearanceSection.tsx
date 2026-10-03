import type * as React from "react";

import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Separator } from "@/components/ui/separator";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useSettingsStore, type Theme } from "@/stores/settings.store";

const THEMES: { value: Theme; label: string }[] = [
  { value: "dark", label: "Dark" },
  { value: "light", label: "Light" },
  { value: "system", label: "Match system" },
];

/** Theme, density and keyboard behaviour. */
export function AppearanceSection(): React.JSX.Element {
  const theme = useSettingsStore((state) => state.theme);
  const setTheme = useSettingsStore((state) => state.setTheme);
  const compactMode = useSettingsStore((state) => state.compactMode);
  const setCompactMode = useSettingsStore((state) => state.setCompactMode);
  const sendOnEnter = useSettingsStore((state) => state.sendOnEnter);
  const setSendOnEnter = useSettingsStore((state) => state.setSendOnEnter);
  const showTypingIndicator = useSettingsStore((state) => state.showTypingIndicator);
  const setShowTypingIndicator = useSettingsStore((state) => state.setShowTypingIndicator);

  return (
    <div className="space-y-6 py-4">
      <div className="space-y-2">
        <Label htmlFor="theme-select">Theme</Label>
        <Select value={theme} onValueChange={(value) => setTheme(value as Theme)}>
          <SelectTrigger id="theme-select" className="max-w-xs">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {THEMES.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <Separator />

      <ToggleRow
        id="compact-mode"
        label="Compact mode"
        description="Tighten the spacing between messages."
        checked={compactMode}
        onCheckedChange={setCompactMode}
      />

      <ToggleRow
        id="send-on-enter"
        label="Send with Enter"
        description="When off, Enter inserts a newline and Ctrl+Enter sends."
        checked={sendOnEnter}
        onCheckedChange={setSendOnEnter}
      />

      <ToggleRow
        id="typing-indicator"
        label="Show typing indicators"
        description="See when others are typing in a channel."
        checked={showTypingIndicator}
        onCheckedChange={setShowTypingIndicator}
      />
    </div>
  );
}

export interface ToggleRowProps {
  id: string;
  label: string;
  description: string;
  checked: boolean;
  onCheckedChange: (value: boolean) => void;
}

/** Labelled switch row, shared by every settings section. */
export function ToggleRow({
  id,
  label,
  description,
  checked,
  onCheckedChange,
}: ToggleRowProps): React.JSX.Element {
  return (
    <div className="flex items-center justify-between gap-4">
      <div className="min-w-0">
        <Label htmlFor={id} className="cursor-pointer">
          {label}
        </Label>
        <p className="text-xs text-muted-foreground">{description}</p>
      </div>
      <Switch id={id} checked={checked} onCheckedChange={onCheckedChange} />
    </div>
  );
}