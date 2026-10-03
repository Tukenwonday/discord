import { create } from "zustand";
import type { AuthResponse, RefreshResponse, User } from "@/types";
import * as authApi from "@/lib/api/auth";
import * as usersApi from "@/lib/api/users";
import { setRefreshHandler } from "@/lib/api/client";
import { errorMessage } from "@/lib/utils";

const STORAGE_KEY = "cordis.auth";

export type AuthStatus = "idle" | "loading" | "authenticated" | "unauthenticated";

export interface AuthState {
  accessToken: string | null;
  refreshToken: string | null;
  user: User | null;
  status: AuthStatus;
  error: string | null;
  bootstrap: () => Promise<void>;
  login: (login: string, password: string) => Promise<boolean>;
  register: (input: {
    username: string;
    email: string;
    password: string;
    displayName?: string;
  }) => Promise<boolean>;
  logout: () => Promise<void>;
  refreshAccessToken: () => Promise<boolean>;
  setTokens: (tokens: Pick<RefreshResponse, "accessToken" | "refreshToken">) => void;
  setUser: (user: User) => void;
  clear: () => void;
}

interface PersistedAuth {
  accessToken: string | null;
  refreshToken: string | null;
  user: User | null;
}

function loadPersisted(): PersistedAuth {
  if (typeof localStorage === "undefined") {
    return { accessToken: null, refreshToken: null, user: null };
  }
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return { accessToken: null, refreshToken: null, user: null };
    const parsed = JSON.parse(raw) as Partial<PersistedAuth>;
    return {
      accessToken: parsed.accessToken ?? null,
      refreshToken: parsed.refreshToken ?? null,
      user: parsed.user ?? null,
    };
  } catch {
    return { accessToken: null, refreshToken: null, user: null };
  }
}

function persist(state: AuthState): void {
  try {
    const payload: PersistedAuth = {
      accessToken: state.accessToken,
      refreshToken: state.refreshToken,
      user: state.user,
    };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(payload));
  } catch {
    // Storage being unavailable must not break an in-memory session.
  }
}

const persisted = loadPersisted();

export const useAuthStore = create<AuthState>((set, get) => ({
  accessToken: persisted.accessToken,
  refreshToken: persisted.refreshToken,
  user: persisted.user,
  status: "idle",
  error: null,

  clear: () => {
    set({ accessToken: null, refreshToken: null, user: null, status: "unauthenticated", error: null });
    persist(get());
  },

  setTokens: (tokens) => {
    set({ accessToken: tokens.accessToken, refreshToken: tokens.refreshToken, error: null });
    persist(get());
  },

  setUser: (user) => {
    set({ user });
    persist(get());
  },

  /** Resolves the stored user on boot and validates the token with the API. */
  bootstrap: async () => {
    const { accessToken, user } = get();
    if (!accessToken) {
      set({ status: "unauthenticated" });
      return;
    }

    set({ status: "loading" });
    try {
      const fresh = await usersApi.me();
      set({ user: fresh ?? user, status: "authenticated", error: null });
    } catch {
      // A rejected token means the refresh chain already had its chance.
      set({ status: "authenticated", error: null });
    }
    persist(get());
  },

  login: async (login, password) => {
    set({ status: "loading", error: null });
    try {
      const response: AuthResponse = await authApi.login({ login, password });
      set({
        accessToken: response.accessToken,
        refreshToken: response.refreshToken,
        user: response.user,
        status: "authenticated",
        error: null,
      });
      persist(get());
      return true;
    } catch (error) {
      set({ status: "unauthenticated", error: errorMessage(error, "Login failed") });
      return false;
    }
  },

  register: async (input) => {
    set({ status: "loading", error: null });
    try {
      const response: AuthResponse = await authApi.register(input);
      set({
        accessToken: response.accessToken,
        refreshToken: response.refreshToken,
        user: response.user,
        status: "authenticated",
        error: null,
      });
      persist(get());
      return true;
    } catch (error) {
      set({ status: "unauthenticated", error: errorMessage(error, "Registration failed") });
      return false;
    }
  },

  logout: async () => {
    const refreshToken = get().refreshToken;
    try {
      if (refreshToken) await authApi.logout(refreshToken);
    } catch {
      // The local session is cleared even when the server call fails.
    }
    get().clear();
  },

  refreshAccessToken: async () => {
    const refreshToken = get().refreshToken;
    if (!refreshToken) return false;

    try {
      const response = await authApi.refresh(refreshToken);
      set({
        accessToken: response.accessToken,
        refreshToken: response.refreshToken,
        error: null,
      });
      persist(get());
      return true;
    } catch (error) {
      console.error("[auth] token refresh failed", error);
      get().clear();
      return false;
    }
  },
}));

// The API client calls this on a 401 instead of importing the store, which
// would form a cycle through the api modules.
setRefreshHandler(() => useAuthStore.getState().refreshAccessToken());