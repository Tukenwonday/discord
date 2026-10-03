import { useEffect, type React } from "react";
import { Navigate, Outlet } from "react-router-dom";
import { useAuthStore } from "@/stores/auth.store";
import { useAuth } from "@/hooks/useAuth";
import { bootstrapSettings } from "@/stores/settings.store";

/**
 * Guards the authenticated application. While the stored session is being
 * validated nothing is redirected, so a reload does not bounce to /login.
 */
export function AuthRoute(): React.JSX.Element {
  const status = useAuthStore((state) => state.status);
  const accessToken = useAuthStore((state) => state.accessToken);
  const bootstrap = useAuthStore((state) => state.bootstrap);
  const { isLoading } = useAuth();

  // The theme must be applied before any protected screen paints.
  useEffect(() => {
    bootstrapSettings();
  }, []);

  useEffect(() => {
    if (status === "idle") {
      void bootstrap();
    }
  }, [status, bootstrap]);

  if (status === "idle" || isLoading) {
    return (
      <div className="flex h-full w-full items-center justify-center bg-background">
        <p className="text-sm text-muted-foreground">Loading Cordis…</p>
      </div>
    );
  }

  if (status !== "authenticated" || !accessToken) {
    return <Navigate to="/login" replace />;
  }

  return <Outlet />;
}