//! Commands callable from the frontend through `invoke`.
//!
//! Commands return `Result<T, String>` instead of panicking so a failure on
//! the Rust side surfaces as a rejected promise the settings dialog can show.

use std::fs;
use tauri::{AppHandle, Manager};
use tauri_plugin_autostart::ManagerExt;
use tauri_plugin_clipboard_manager::ClipboardExt;
use tauri_plugin_opener::OpenerExt;

use crate::notifications;
use crate::tray;
use crate::updater;
use crate::window;

/// File name holding the stable installation identifier.
const DEVICE_ID_FILE: &str = "device-id";

/// Schemes [`open_external`] is willing to hand to the operating system.
///
/// Restricting the list prevents a message body from turning into a
/// `file:` or `ms-settings:` launch when a link is rendered by the frontend.
const ALLOWED_EXTERNAL_SCHEMES: [&str; 2] = ["http://", "https://"];

/// Returns the version of the running desktop shell.
#[tauri::command]
pub fn app_version(app: AppHandle) -> String {
    app.package_info().version.to_string()
}

/// Registers or removes the application from the system start-up list.
#[tauri::command]
pub fn set_launch_on_boot(app: AppHandle, enabled: bool) -> Result<bool, String> {
    let manager = app.autolaunch();
    if enabled {
        manager.enable().map_err(|error| error.to_string())?;
    } else {
        manager.disable().map_err(|error| error.to_string())?;
    }

    manager.is_enabled().map_err(|error| error.to_string())
}

/// Reports whether the application is currently registered for start-up.
#[tauri::command]
pub fn is_launch_on_boot_enabled(app: AppHandle) -> Result<bool, String> {
    app.autolaunch()
        .is_enabled()
        .map_err(|error| error.to_string())
}

/// Opens an `http` or `https` link with the default system handler.
#[tauri::command]
pub fn open_external(app: AppHandle, url: String) -> Result<(), String> {
    let candidate = url.trim();
    if !ALLOWED_EXTERNAL_SCHEMES
        .iter()
        .any(|scheme| candidate.starts_with(scheme))
    {
        return Err(format!("refusing to open unsupported url: {candidate}"));
    }

    app.opener()
        .open_url(candidate, None::<&str>)
        .map_err(|error| error.to_string())
}

/// Writes text to the system clipboard.
#[tauri::command]
pub fn copy_to_clipboard(app: AppHandle, text: String) -> Result<(), String> {
    app.clipboard()
        .write_text(text)
        .map_err(|error| error.to_string())
}

/// Shows an operating system notification.
#[tauri::command]
pub fn show_native_notification(
    app: AppHandle,
    title: String,
    body: String,
) -> Result<(), String> {
    notifications::show_native_notification_if_allowed(&app, &title, &body)
}

/// Asks the frontend to display a notification of the given reason.
#[tauri::command]
pub fn request_frontend_notification(
    app: AppHandle,
    id: String,
    reason: notifications::NotifyReason,
    title: String,
    body: String,
    conversation_id: Option<String>,
) {
    notifications::emit_notify_request(
        &app,
        notifications::NotifyRequest {
            id,
            reason,
            title,
            body,
            conversation_id,
        },
    );
}

/// Reads the stable device identifier, creating it on first use.
///
/// The backend uses it to keep a device in the trusted device list across
/// sessions, so it must survive application updates and cache clears.
#[tauri::command]
pub fn device_id(app: AppHandle) -> Result<String, String> {
    let directory = app
        .path()
        .app_config_dir()
        .map_err(|error| error.to_string())?;

    fs::create_dir_all(&directory).map_err(|error| {
        format!(
            "failed to create config directory {}: {error}",
            directory.display()
        )
    })?;

    let path = directory.join(DEVICE_ID_FILE);

    match fs::read_to_string(&path) {
        Ok(existing) => {
            let trimmed = existing.trim().to_string();
            if !trimmed.is_empty() {
                return Ok(trimmed);
            }
        }
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => {}
        Err(error) => {
            return Err(format!(
                "failed to read device id from {}: {error}",
                path.display()
            ));
        }
    }

    let generated = uuid::Uuid::new_v4().to_string();
    fs::write(&path, &generated).map_err(|error| {
        format!(
            "failed to persist device id to {}: {error}",
            path.display()
        )
    })?;

    Ok(generated)
}

/// Returns the current tray state so the frontend can seed its own store.
#[tauri::command]
pub fn tray_state(app: AppHandle) -> tray::TrayState {
    tray::state_snapshot(&app)
}

/// Applies a mute, deafen or presence change coming from the frontend.
#[tauri::command]
pub fn set_tray_state(
    app: AppHandle,
    muted: Option<bool>,
    deafened: Option<bool>,
    status: Option<String>,
) -> tray::TrayState {
    tray::sync_state(&app, muted, deafened, status)
}

/// Runs the update check and reports the installed version, if any.
#[tauri::command]
pub async fn check_for_updates(app: AppHandle) -> Result<Option<String>, String> {
    updater::check_for_update(app).await
}

/// Minimizes the main window.
#[tauri::command]
pub fn minimize_window(app: AppHandle) -> Result<(), String> {
    let main = window::require_main(&app)?;
    window::minimize(&main)
}

/// Toggles the maximized state of the main window and reports the new state.
#[tauri::command]
pub fn toggle_maximize_window(app: AppHandle) -> Result<bool, String> {
    let main = window::require_main(&app)?;
    window::toggle_maximize(&main)
}

/// Hides the main window without exiting.
#[tauri::command]
pub fn hide_window(app: AppHandle) -> Result<(), String> {
    let main = window::require_main(&app)?;
    window::hide(&main)
}

/// Shows and focuses the main window.
#[tauri::command]
pub fn show_window(app: AppHandle) -> Result<(), String> {
    let main = window::require_main(&app)?;
    window::restore_and_focus(&main)
}

/// Closes the main window, which ends the application.
#[tauri::command]
pub fn close_window(app: AppHandle) -> Result<(), String> {
    let main = window::require_main(&app)?;
    window::close(&main)
}

/// Starts dragging the frameless window, backing the title bar drag region.
#[tauri::command]
pub fn start_dragging(app: AppHandle) -> Result<(), String> {
    let main = window::require_main(&app)?;
    window::start_dragging(&main)
}