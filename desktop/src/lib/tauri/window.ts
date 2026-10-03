import { isTauri } from "./desktop";

type MaximizedListener = (maximized: boolean) => void;

const maximizedListeners = new Set<MaximizedListener>();
let maximized = false;
let resizeBound = false;

/** Invokes a window command, degrading to a no-op in a plain browser. */
async function run(command: string): Promise<void> {
  try {
    if (!isTauri()) return;
    const { invoke } = await import("@tauri-apps/api/core");
    await invoke(command);
  } catch (error) {
    console.error(`[tauri] ${command} failed`, error);
  }
}

/** Backs the `drag-region` on the titlebar for the frameless window. */
export function startDragging(): void {
  void run("start_dragging");
}

export function minimize(): void {
  void run("minimize_window");
}

/** Returns the new maximised state so the button icon can follow it. */
export async function toggleMaximize(): Promise<boolean> {
  try {
    if (!isTauri()) {
      maximized = !maximized;
      maximizedListeners.forEach((listener) => listener(maximized));
      return maximized;
    }
    const { invoke } = await import("@tauri-apps/api/core");
    maximized = await invoke<boolean>("toggle_maximize_window");
    maximizedListeners.forEach((listener) => listener(maximized));
    return maximized;
  } catch (error) {
    console.error("[tauri] toggle_maximize_window failed", error);
    return maximized;
  }
}

export function close(): void {
  void run("close_window");
}

export function hide(): void {
  void run("hide_window");
}

export function show(): void {
  void run("show_window");
}

/**
 * Tracks the real window state through Tauri's resize event and notifies
 * subscribers; outside Tauri the local toggle flag is the only signal.
 */
export function isMaximized(): boolean {
  return maximized;
}

export function subscribeMaximized(listener: MaximizedListener): () => void {
  maximizedListeners.add(listener);
  listener(maximized);
  return () => {
    maximizedListeners.delete(listener);
  };
}

/** Binds the Tauri resize event once; safe to call on every component mount. */
export async function bindResizeListener(): Promise<() => void> {
  try {
    if (!isTauri() || resizeBound) return () => undefined;

    const { getCurrentWindow } = await import("@tauri-apps/api/window");
    const appWindow = getCurrentWindow();
    const unlisten = await appWindow.onResized(async () => {
      const isNowMaximized = await appWindow.isMaximized();
      if (isNowMaximized === maximized) return;
      maximized = isNowMaximized;
      maximizedListeners.forEach((listener) => listener(maximized));
    });

    maximized = await appWindow.isMaximized();
    resizeBound = true;
    return () => {
      unlisten.then((fn) => fn()).catch(() => undefined);
      resizeBound = false;
    };
  } catch (error) {
    console.error("[tauri] resize listener failed", error);
    return () => undefined;
  }
}