//! Window helpers used by the tray, the title bar and the Rust commands.
//!
//! Every helper takes a window reference instead of looking one up so the
//! caller stays in control of which window is affected, and every failure is
//! returned as a message instead of panicking because a missing window must
//! never take the whole application down.

use tauri::{Manager, Runtime, WebviewWindow};

/// Label of the single application window declared in `tauri.conf.json`.
pub const MAIN_WINDOW: &str = "main";

/// Message returned by the commands when the window is already gone.
pub const MAIN_WINDOW_MISSING: &str = "main window not found";

/// Returns the main application window when it still exists.
///
/// Windows can disappear when the frontend navigates or when the user closes
/// the window, so callers must treat `None` as a normal outcome.
pub fn main_window<R: Runtime>(manager: &impl Manager<R>) -> Option<WebviewWindow<R>> {
    manager.get_webview_window(MAIN_WINDOW)
}

/// Returns the main window or the message the commands report to the frontend.
pub fn require_main<R: Runtime>(manager: &impl Manager<R>) -> Result<WebviewWindow<R>, String> {
    main_window(manager).ok_or_else(|| MAIN_WINDOW_MISSING.to_string())
}

/// Minimizes the window.
pub fn minimize<R: Runtime>(window: &WebviewWindow<R>) -> Result<(), String> {
    window.minimize().map_err(|error| error.to_string())
}

/// Brings the window back from a minimized state and focuses it.
///
/// The three steps are always attempted so a failing unminimize cannot leave
/// the window hidden, and the first error is reported to the caller.
pub fn restore_and_focus<R: Runtime>(window: &WebviewWindow<R>) -> Result<(), String> {
    let unminimized = window.unminimize().map_err(|error| error.to_string());
    let shown = window.show().map_err(|error| error.to_string());
    let focused = window.set_focus().map_err(|error| error.to_string());
    unminimized.and(shown).and(focused)
}

/// Toggles the maximized state of the window and reports the new state.
pub fn toggle_maximize<R: Runtime>(window: &WebviewWindow<R>) -> Result<bool, String> {
    let maximized = window.is_maximized().map_err(|error| error.to_string())?;
    if maximized {
        window.unmaximize().map_err(|error| error.to_string())?;
        Ok(false)
    } else {
        window.maximize().map_err(|error| error.to_string())?;
        Ok(true)
    }
}

/// Hides the window without exiting the application.
pub fn hide<R: Runtime>(window: &WebviewWindow<R>) -> Result<(), String> {
    window.hide().map_err(|error| error.to_string())
}

/// Shows the window without stealing focus.
pub fn show<R: Runtime>(window: &WebviewWindow<R>) -> Result<(), String> {
    window.show().map_err(|error| error.to_string())
}

/// Closes the window, which ends the application because it is the only one.
pub fn close<R: Runtime>(window: &WebviewWindow<R>) -> Result<(), String> {
    window.close().map_err(|error| error.to_string())
}

/// Starts moving the frameless window with the pointer.
///
/// This is the Rust counterpart of the `data-tauri-drag-region` attribute used
/// by the title bar.
pub fn start_dragging<R: Runtime>(window: &WebviewWindow<R>) -> Result<(), String> {
    window.start_dragging().map_err(|error| error.to_string())
}

/// Shows the window when it is hidden and hides it when it is visible.
///
/// This backs the left click behaviour of the tray icon, which has no way to
/// report an error to a user, so failures are logged.
pub fn toggle_visibility<R: Runtime>(manager: &impl Manager<R>) {
    let Some(window) = main_window(manager) else {
        return;
    };

    let visible = match window.is_visible() {
        Ok(visible) => visible,
        Err(error) => {
            eprintln!("[cordis] failed to read window visibility: {error}");
            return;
        }
    };

    let outcome = if visible {
        hide(&window)
    } else {
        restore_and_focus(&window)
    };

    if let Err(error) = outcome {
        eprintln!("[cordis] failed to toggle window visibility: {error}");
    }
}

/// Always reveals the window, used by the `Show Cordis` tray entry.
pub fn focus_main_window<R: Runtime>(manager: &impl Manager<R>) {
    if let Some(window) = main_window(manager) {
        if let Err(error) = restore_and_focus(&window) {
            eprintln!("[cordis] failed to focus the main window: {error}");
        }
    }
}