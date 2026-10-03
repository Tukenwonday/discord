import type * as React from "react";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { TooltipProvider } from "@/components/ui/tooltip";
import { Toaster } from "@/components/ui/toaster";
import { AuthRoute } from "@/routes/AuthRoute";
import { AppRoute } from "@/routes/AppRoute";
import { LoginPage } from "@/pages/LoginPage";
import { RegisterPage } from "@/pages/RegisterPage";
import { AppPage } from "@/pages/AppPage";

// Retries are handled explicitly by the refresh chain, so the default 3 would
// multiply requests against a backend that is simply rejecting a stale token.
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 15_000,
    },
  },
});

export function App(): React.JSX.Element {
  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider delayDuration={200}>
        <BrowserRouter>
          <Routes>
            <Route element={<AppRoute />}>
              <Route path="/login" element={<LoginPage />} />
              <Route path="/register" element={<RegisterPage />} />
            </Route>

            <Route element={<AuthRoute />}>
              <Route path="/channels" element={<AppPage />} />
              <Route path="/channels/:channelId" element={<AppPage />} />
              <Route path="/dm/:dmId" element={<AppPage />} />
              <Route path="/settings" element={<AppPage />} />
            </Route>

            <Route path="*" element={<Navigate to="/channels" replace />} />
          </Routes>
        </BrowserRouter>
        <Toaster />
      </TooltipProvider>
    </QueryClientProvider>
  );
}