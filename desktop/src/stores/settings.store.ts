import { create } from "zustand";
import { trimTrailingSlash } from "@/lib/utils";
import type { UserStatus } from "@/types";

export type Theme = "dark" | "light" | "system";

const STORAGE_KEY = "cordis.settings";

export interface SettingsState {
  apiBaseUrl: string;
  wsUrl: string;
  theme: Theme;
  status: UserStatus;
  compactMode: boolean;
  sendOnEnter: boolean;
  showTypingIndicator: boolean;
  notificationsEnabled: boolean;
  notifyOnMention: boolean;
  notifyOnDM: boolean;
  autostart: boolean;
  applyTheme: () => void;
  setApiBaseUrl: (url: string) => void;
  setWsUrl: (url: string) => void;
  setTheme: (theme: Theme) => void;
  setStatus: (status: UserStatus) => void;
  setCompactMode: (value: boolean) => void;
  setSendOnEnter: (value: boolean) => void;
  setShowTypingIndicator: (value: boolean) => void;
  setNotificationsEnabled: (value: boolean) => void;
  setNotifyOnMention: (value: boolean) => void;
  setNotifyOnDM: (value: boolean) => void;
  setAutostart: (value: boolean) => void;
  persist: () => void;
}

type PersistedShape = Pick<
  SettingsState,
  | "apiBaseUrl"
  | "wsUrl"
  | "theme"
  | "status"
  | "compactMode"
  | "sendOnEnter"
  | "showTypingIndicator"
  | "notificationsEnabled"
  | "notifyOnMention"
  | "notifyOnDM"
  | "autostart"
>;

// IP-ONLY: TODO - bare IP deployment fallback. These only apply when no build-time
// VITE_API_BASE_URL / VITE_WS_URL is present and the user has not saved a value in
// Settings -> Connection. Restore 127.0.0.1 for local development.
const DEFAULT_API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL || "http://34.165.143.31:8080";
const DEFAULT_WS_URL =
  import.meta.env.VITE_WS_URL || "ws://34.165.143.31:8080/ws";

const defaults: PersistedShape = {
  apiBaseUrl: trimTrailingSlash(DEFAULT_API_BASE_URL),
  wsUrl: DEFAULT_WS_URL,
  theme: "dark",
  status: "online",
  compactMode: false,
  sendOnEnter: true,
  showTypingIndicator: true,
  notificationsEnabled: true,
  notifyOnMention: true,
  notifyOnDM: true,
  autostart: false,
};

function loadPersisted(): PersistedShape {
  if (typeof localStorage === "undefined") return defaults;
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return defaults;
    const parsed = JSON.parse(raw) as Partial<PersistedShape>;
    return {
      ...defaults,
      ...parsed,
      apiBaseUrl: trimTrailingSlash(
        parsed.apiBaseUrl?.trim() ? parsed.apiBaseUrl : DEFAULT_API_BASE_URL,
      ),
      wsUrl: parsed.wsUrl?.trim() ? parsed.wsUrl.trim() : DEFAULT_WS_URL,
    };
  } catch {
    return defaults;
  }
}

/** Writes the `dark` class plus the matching colour-scheme on <html>. */
function resolveTheme(theme: Theme): void {
  const prefersDark =
    typeof window !== "undefined" &&
    typeof window.matchMedia === "function" &&
    window.matchMedia("(prefers-color-scheme: dark)").matches;
  const dark = theme === "dark" || (theme === "system" && prefersDark);
  document.documentElement.classList.toggle("dark", dark);
  document.documentElement.style.colorScheme = dark ? "dark" : "light";
}

const initial = loadPersisted();

export const useSettingsStore = create<SettingsState>((set, get) => ({
  ...initial,
  applyTheme: () => resolveTheme(get().theme),
  setApiBaseUrl: (url) => set({ apiBaseUrl: trimTrailingSlash(url.trim()) }),
  setWsUrl: (url) => set({ wsUrl: url.trim() }),
  setTheme: (theme) => {
    set({ theme });
    resolveTheme(theme);
    get().persist();
  },
  setStatus: (status) => {
    set({ status });
    get().persist();
  },
  setCompactMode: (value) => {
    set({ compactMode: value });
    get().persist();
  },
  setSendOnEnter: (value) => {
    set({ sendOnEnter: value });
    get().persist();
  },
  setShowTypingIndicator: (value) => {
    set({ showTypingIndicator: value });
    get().persist();
  },
  setNotificationsEnabled: (value) => {
    set({ notificationsEnabled: value });
    get().persist();
  },
  setNotifyOnMention: (value) => {
    set({ notifyOnMention: value });
    get().persist();
  },
  setNotifyOnDM: (value) => {
    set({ notifyOnDM: value });
    get().persist();
  },
  setAutostart: (value) => {
    set({ autostart: value });
    get().persist();
  },
  persist: () => {
    const state = get();
    const payload: PersistedShape = {
      apiBaseUrl: state.apiBaseUrl,
      wsUrl: state.wsUrl,
      theme: state.theme,
      status: state.status,
      compactMode: state.compactMode,
      sendOnEnter: state.sendOnEnter,
      showTypingIndicator: state.showTypingIndicator,
      notificationsEnabled: state.notificationsEnabled,
      notifyOnMention: state.notifyOnMention,
      notifyOnDM: state.notifyOnDM,
      autostart: state.autostart,
    };
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(payload));
    } catch {
      // A full or blocked storage quota must never break the session.
    }
  },
}));

/** Applies the persisted theme during boot, before React mounts. */
export function bootstrapSettings(): void {
  resolveTheme(useSettingsStore.getState().theme);
  useSettingsStore.getState().applyTheme();
}
