import { useEffect, type React } from "react";
import { Navigate, Outlet } from "react-router-dom";
import { useAuthStore } from "@/stores/auth.store";
import { bootstrapSettings } from "@/stores/settings.store";

/**
 * Guards /login and /register. An authenticated session is sent to the channel
 * view; a fresh session still has to resolve before deciding.
 */
export function AppRoute(): React.JSX.Element {
  const status = useAuthStore((state) => state.status);
  const accessToken = useAuthStore((state) => state.accessToken);
  const bootstrap = useAuthStore((state) => state.bootstrap);

  useEffect(() => {
    bootstrapSettings();
  }, []);

  useEffect(() => {
    if (status === "idle") {
      void bootstrap();
    }
  }, [status, bootstrap]);

  if (status === "idle") {
    return (
      <div className="flex h-full w-full items-center justify-center bg-background">
        <p className="text-sm text-muted-foreground">Loading Cordis…</p>
      </div>
    );
  }

  if (status === "authenticated" && accessToken) {
    return <Navigate to="/channels" replace />;
  }

  return <Outlet />;
}