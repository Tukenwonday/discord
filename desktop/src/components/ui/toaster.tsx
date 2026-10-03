import * as React from "react";
import {
  Toast,
  ToastAction,
  ToastClose,
  ToastDescription,
  ToastProvider,
  ToastTitle,
  ToastViewport,
  useToast,
  type ToastItem,
} from "./toast";
import { Toaster as SonnerToaster } from "sonner";
import { useTheme } from "@/hooks/useTheme";

/**
 * Renders the shadcn toast stack and sonner's toaster side by side. Components
 * may use either API; both are themed from the same tokens.
 */
export function Toaster(): React.JSX.Element {
  const { toasts } = useToast();
  const { isDark } = useTheme();

  return (
    <>
      <ToastProvider>
        {toasts.map(({ id, title, description, actionLabel, onAction, ...props }) => (
          <Toast key={id} {...props}>
            <div className="grid gap-1">
              {title ? <ToastTitle>{title}</ToastTitle> : null}
              {description ? <ToastDescription>{description}</ToastDescription> : null}
            </div>
            {actionLabel ? (
              <ToastAction altText={actionLabel} onClick={onAction}>
                {actionLabel}
              </ToastAction>
            ) : null}
            <ToastClose />
          </Toast>
        ))}
        <ToastViewport />
      </ToastProvider>

      <SonnerToaster theme={isDark ? "dark" : "light"} position="bottom-right" richColors closeButton />
    </>
  );
}