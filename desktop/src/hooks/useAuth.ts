import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useAuthStore } from "@/stores/auth.store";
import type { User } from "@/types";

export interface UseAuthResult {
  user: User | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  login: (login: string, password: string) => Promise<boolean>;
  register: (input: {
    username: string;
    email: string;
    password: string;
    displayName?: string;
  }) => Promise<boolean>;
  logout: () => Promise<void>;
  updateUser: (user: User) => void;
}

export function useAuth(): UseAuthResult {
  const user = useAuthStore((state) => state.user);
  const accessToken = useAuthStore((state) => state.accessToken);
  const status = useAuthStore((state) => state.status);
  const error = useAuthStore((state) => state.error);
  const loginAction = useAuthStore((state) => state.login);
  const registerAction = useAuthStore((state) => state.register);
  const logoutAction = useAuthStore((state) => state.logout);
  const setUser = useAuthStore((state) => state.setUser);

  return {
    user,
    accessToken,
    isAuthenticated: status === "authenticated" && accessToken !== null,
    isLoading: status === "loading" || status === "idle",
    error,
    login: loginAction,
    register: registerAction,
    logout: logoutAction,
    updateUser: setUser,
  };
}

/** Resolves the stored session once and follows the user to the right screen. */
export function useRequireAuth(): UseAuthResult & { ready: boolean } {
  const auth = useAuth();
  const bootstrap = useAuthStore((state) => state.bootstrap);
  const navigate = useNavigate();

  useEffect(() => {
    void bootstrap();
    // Bootstrap is a one-shot on mount for the authenticated tree.
  }, [bootstrap]);

  const ready = auth.isLoading === false;

  useEffect(() => {
    if (ready && !auth.isAuthenticated) {
      navigate("/login", { replace: true });
    }
  }, [ready, auth.isAuthenticated, navigate]);

  return { ...auth, ready };
}

/** Sends an authenticated user away from /login and /register. */
export function useRedirectIfAuthenticated(): void {
  const isAuthenticated = useAuthStore((state) => state.status === "authenticated");
  const accessToken = useAuthStore((state) => state.accessToken !== null);
  const bootstrap = useAuthStore((state) => state.bootstrap);
  const navigate = useNavigate();

  useEffect(() => {
    void bootstrap();
  }, [bootstrap]);

  useEffect(() => {
    if (isAuthenticated && accessToken) {
      navigate("/channels", { replace: true });
    }
  }, [isAuthenticated, accessToken, navigate]);
}