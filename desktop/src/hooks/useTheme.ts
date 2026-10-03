import { useEffect, useState } from "react";
import { useSettingsStore, type Theme } from "@/stores/settings.store";

export interface UseThemeResult {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  isDark: boolean;
}

/**
 * Applies the `dark` class to <html> and keeps the OS setting in sync while
 * the theme is `system`.
 */
export function useTheme(): UseThemeResult {
  const theme = useSettingsStore((state) => state.theme);
  const setTheme = useSettingsStore((state) => state.setTheme);
  const applyTheme = useSettingsStore((state) => state.applyTheme);

  useEffect(() => {
    applyTheme();
  }, [theme, applyTheme]);

  useEffect(() => {
    if (theme !== "system") return;
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") return;

    const query = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => applyTheme();
    query.addEventListener("change", onChange);
    return () => query.removeEventListener("change", onChange);
  }, [theme, applyTheme]);

  const [isDark, setIsDark] = useState(() =>
    typeof document !== "undefined"
      ? document.documentElement.classList.contains("dark")
      : true,
  );

  // Reading the class after it is written keeps the value in step with the DOM.
  useEffect(() => {
    setIsDark(document.documentElement.classList.contains("dark"));
  }, [theme]);

  return { theme, setTheme, isDark };
}