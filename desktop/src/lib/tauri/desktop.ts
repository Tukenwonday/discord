export interface TrayAction {
  action: string;
  value: string;
}

export interface NotifyRequest {
  id: string;
  reason: "dm" | "mention";
  title: string;
  body: string;
  conversationId: string | null;
}

export interface UpdaterStatus {
  available: boolean;
  version: string | null;
  notes: string | null;
}

export interface SelectedFile {
  name: string;
  /** A real path when the Tauri dialog produced one, otherwise a browser File. */
  path: string | null;
  file: File | null;
}

/** True when running inside the Tauri webview rather than a plain browser. */
export function isTauri(): boolean {
  return typeof window !== "undefined" && window.__TAURI_INTERNALS__ !== undefined;
}

async function invoke<T>(command: string, args?: Record<string, unknown>): Promise<T> {
  const { invoke: tauriInvoke } = await import("@tauri-apps/api/core");
  return tauriInvoke<T>(command, args);
}

/** Native notification with a Notification API fallback for browser dev. */
export async function notify(title: string, body: string): Promise<void> {
  try {
    if (isTauri()) {
      const { isPermissionGranted, requestPermission, sendNotification } = await import(
        "@tauri-apps/plugin-notification"
      );
      let granted = await isPermissionGranted();
      if (!granted) {
        granted = (await requestPermission()) === "granted";
      }
      if (granted) {
        sendNotification({ title, body });
        return;
      }
    }

    if (typeof Notification !== "undefined") {
      if (Notification.permission === "granted") {
        new Notification(title, { body });
      } else if (Notification.permission === "default") {
        void Notification.requestPermission();
      }
    }
  } catch (error) {
    console.error("[tauri] notify failed", error);
  }
}

/** Fire-and-forget notification used for DMs and mentions. */
export function notifyMention(request: NotifyRequest): void {
  void notify(request.title, request.body);
}

export interface OpenDialogOptions {
  multiple?: boolean;
  directory?: boolean;
  filters?: { name: string; extensions: string[] }[];
}

const IMAGE_EXTENSIONS = ["png", "jpg", "jpeg", "gif", "webp"];
/**
 * Opens the native picker when packaged. Returns either a filesystem path
 * (Tauri) or an in-memory File (browser `<input type="file">` fallback); the
 * upload module handles both shapes.
 */
export async function openFileDialog(
  options: OpenDialogOptions = {},
): Promise<SelectedFile[]> {
  try {
    if (isTauri()) {
      const { open } = await import("@tauri-apps/plugin-dialog");
      const selected = await open({
        multiple: options.multiple ?? true,
        directory: options.directory ?? false,
        filters: options.filters ?? [
          { name: "Images", extensions: IMAGE_EXTENSIONS },
          { name: "All files", extensions: ["*"] },
        ],
      });

      if (!selected) return [];
      const paths = Array.isArray(selected) ? selected : [selected];
      return paths.map((path) => ({
        name: path.split(/[\\/]/).pop() ?? "file",
        path,
        file: null,
      }));
    }
  } catch (error) {
    console.error("[tauri] dialog failed", error);
  }

  return openWithInput(options);
}

/** Browser-only picker used when the Tauri dialog plugin is unavailable. */
function openWithInput(options: OpenDialogOptions): Promise<SelectedFile[]> {
  return new Promise((resolve) => {
    const input = document.createElement("input");
    input.type = "file";
    input.multiple = options.multiple ?? true;
    input.style.display = "none";
    document.body.appendChild(input);

    const cleanup = () => input.remove();

    input.addEventListener("change", () => {
      const files = input.files ? Array.from(input.files) : [];
      cleanup();
      resolve(files.map((file) => ({ name: file.name, path: null, file })));
    });
    input.addEventListener("cancel", () => {
      cleanup();
      resolve([]);
    });

    input.click();
  });
}

export async function checkForUpdates(): Promise<UpdaterStatus> {
  try {
    if (!isTauri()) return { available: false, version: null, notes: null };

    const { check } = await import("@tauri-apps/plugin-updater");
    const update = await check();
    if (!update) return { available: false, version: null, notes: null };

    return { available: true, version: update.version, notes: update.body ?? null };
  } catch (error) {
    console.error("[tauri] update check failed", error);
    return { available: false, version: null, notes: null };
  }
}

export async function installUpdate(): Promise<void> {
  try {
    if (!isTauri()) return;
    const { check } = await import("@tauri-apps/plugin-updater");
    const { relaunch } = await import("@tauri-apps/plugin-process");
    const update = await check();
    if (!update) return;
    await update.downloadAndInstall();
    await relaunch();
  } catch (error) {
    console.error("[tauri] update install failed", error);
  }
}

export async function appVersion(): Promise<string> {
  try {
    if (isTauri()) return await invoke<string>("app_version");
  } catch (error) {
    console.error("[tauri] app_version failed", error);
  }
  return "dev";
}

export async function openExternal(url: string): Promise<void> {
  try {
    if (isTauri()) {
      await invoke<void>("open_external", { url });
      return;
    }
    window.open(url, "_blank", "noopener,noreferrer");
  } catch (error) {
    console.error("[tauri] open_external failed", error);
  }
}

export async function copyToClipboard(text: string): Promise<void> {
  try {
    if (isTauri()) {
      await invoke<void>("copy_to_clipboard", { text });
      return;
    }
    await navigator.clipboard.writeText(text);
  } catch (error) {
    console.error("[tauri] clipboard failed", error);
  }
}

export async function setAutostart(enabled: boolean): Promise<boolean> {
  try {
    if (isTauri()) return await invoke<boolean>("set_launch_on_boot", { enabled });
  } catch (error) {
    console.error("[tauri] set_launch_on_boot failed", error);
  }
  return enabled;
}

export async function isAutostartEnabled(): Promise<boolean> {
  try {
    if (isTauri()) return await invoke<boolean>("is_launch_on_boot_enabled");
  } catch (error) {
    console.error("[tauri] is_launch_on_boot_enabled failed", error);
  }
  return false;
}

export async function deviceId(): Promise<string> {
  try {
    if (isTauri()) return await invoke<string>("device_id");
  } catch (error) {
    console.error("[tauri] device_id failed", error);
  }
  return "browser";
}
/** Seeds the voice store from the tray's persisted mute/deafen state. */
export async function loadTrayState(): Promise<{ muted: boolean; deafened: boolean }> {
  try {
    if (!isTauri()) return { muted: false, deafened: false };
    return await invoke<{ muted: boolean; deafened: boolean }>("tray_state");
  } catch (error) {
    console.error("[tauri] tray_state failed", error);
    return { muted: false, deafened: false };
  }
}

export async function syncTrayState(state: {
  muted?: boolean;
  deafened?: boolean;
  status?: string;
}): Promise<void> {
  try {
    if (!isTauri()) return;
    await invoke<unknown>("set_tray_state", state);
  } catch (error) {
    console.error("[tauri] set_tray_state failed", error);
  }
}

export async function onTrayAction(
  handler: (action: TrayAction) => void,
): Promise<() => void> {
  try {
    if (!isTauri()) return () => undefined;
    const { listen } = await import("@tauri-apps/api/event");
    return await listen<TrayAction>("tray://action", (event) => handler(event.payload));
  } catch (error) {
    console.error("[tauri] tray listener failed", error);
    return () => undefined;
  }
}

export async function onNotifyRequest(
  handler: (request: NotifyRequest) => void,
): Promise<() => void> {
  try {
    if (!isTauri()) return () => undefined;
    const { listen } = await import("@tauri-apps/api/event");
    return await listen<NotifyRequest>("notify://request", (event) => handler(event.payload));
  } catch (error) {
    console.error("[tauri] notify listener failed", error);
    return () => undefined;
  }
}

export async function onUpdaterStatus(
  handler: (status: UpdaterStatus) => void,
): Promise<() => void> {
  try {
    if (!isTauri()) return () => undefined;
    const { listen } = await import("@tauri-apps/api/event");
    return await listen<UpdaterStatus>("updater://status", (event) => handler(event.payload));
  } catch (error) {
    console.error("[tauri] updater listener failed", error);
    return () => undefined;
  }
}