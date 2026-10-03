//! Cordis desktop shell.
//!
//! The Rust side owns everything the webview cannot do on its own: the system
//! tray, window management, the operating system integrations (clipboard,
//! external links, start-up registration, notifications) and the update flow.
//! Everything it exposes to the frontend is either a `#[tauri::command]` or an
//! event carrying a documented payload.

mod commands;
mod notifications;
mod tray;
mod updater;
mod window;

use std::sync::Mutex;
use tauri::{Emitter, Manager, WindowEvent};
use tauri_plugin_autostart::MacosLauncher;

/// Event emitted once the shell finished booting and the tray is ready.
pub const READY_EVENT: &str = "app://ready";

/// Payload delivered with [`READY_EVENT`].
#[derive(Debug, Clone, serde::Serialize)]
#[serde(rename_all = "camelCase")]
struct ReadyPayload {
    /// Version of the running shell.
    version: String,
    /// Stable identifier of this installation.
    device_id: Result<String, String>,
    /// Tray state the frontend should seed its presence store with.
    tray: tray::TrayState,
}

/// Builds and runs the Cordis desktop application.
///
/// This is the only entry point of the crate, `main.rs` forwards to it so the
/// binary stays a thin shell around a reusable library.
pub fn run() {
    let app = tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .plugin(tauri_plugin_process::init())
        .plugin(tauri_plugin_os::init())
        .plugin(tauri_plugin_store::Builder::new().build())
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_clipboard_manager::init())
        // macOS registers a Launch Agent plist, Windows writes a registry run
        // key, so both platforms start the app without a visible window.
        .plugin(tauri_plugin_autostart::init(MacosLauncher::LaunchAgent, None))
        .manage(Mutex::new(tray::TrayState::default()))
        .setup(|app| {
            let handle = app.handle().clone();

            if let Err(error) = tray::build_tray(&handle) {
                eprintln!("[cordis] failed to build tray icon: {error}");
            }

            let payload = ReadyPayload {
                version: handle.package_info().version.to_string(),
                device_id: commands::device_id(handle.clone()),
                tray: tray::state_snapshot(&handle),
            };

            if let Err(error) = handle.emit(READY_EVENT, payload) {
                eprintln!("[cordis] failed to emit ready event: {error}");
            }

            Ok(())
        })
        .on_window_event(|window, event| {
            if let WindowEvent::CloseRequested { .. } = event {
                // The tray keeps the process alive, so closing the window only
                // hides it and the user can bring it back from the tray.
                if window.label() == window::MAIN_WINDOW {
                    if let Err(error) = window.hide() {
                        eprintln!("[cordis] failed to hide window on close: {error}");
                    }
                }
            }
        })
        .invoke_handler(tauri::generate_handler![
            commands::app_version,
            commands::set_launch_on_boot,
            commands::is_launch_on_boot_enabled,
            commands::open_external,
            commands::copy_to_clipboard,
            commands::device_id,
            commands::show_native_notification,
            commands::request_frontend_notification,
            commands::tray_state,
            commands::set_tray_state,
            commands::check_for_updates,
            commands::minimize_window,
            commands::toggle_maximize_window,
            commands::hide_window,
            commands::show_window,
            commands::close_window,
            commands::start_dragging,
        ])
        .build(tauri::generate_context!())
        .expect("failed to build the Cordis desktop application");

    app.run(|_handle, _event| {
        // The application exits through the tray Quit entry or when the main
        // window is closed, both of which end the event loop already.
    });
}