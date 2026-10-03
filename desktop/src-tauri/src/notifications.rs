//! Notification plumbing.
//!
//! The frontend renders in-app toasts and asks the shell to raise an OS level
//! notification for direct messages and mentions. `show_native_notification`
//! is the fallback used when the webview notification API is unavailable, and
//! `emit_notify_request` forwards the decision to the frontend so the user
//! preferences in the settings dialog stay authoritative.

use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Emitter, Runtime};
use tauri_plugin_notification::{NotificationExt, PermissionState};

/// Event emitted when the shell wants the frontend to display a notification.
pub const NOTIFY_REQUEST_EVENT: &str = "notify://request";

/// Reasons the shell can ask the frontend for a notification.
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub enum NotifyReason {
    /// A new direct message arrived.
    DirectMessage,
    /// The current user was mentioned in a channel.
    Mention,
    /// A friend came online.
    FriendOnline,
    /// Any other notification the backend asked for.
    Generic,
}

/// Payload delivered with [`NOTIFY_REQUEST_EVENT`].
#[derive(Debug, Clone, Serialize)]
pub struct NotifyRequest {
    /// Stable identifier so the frontend can deduplicate repeated requests.
    pub id: String,
    /// Why the notification was raised.
    pub reason: NotifyReason,
    /// Notification heading.
    pub title: String,
    /// Notification body, already formatted for display.
    pub body: String,
    /// Conversation identifier to open when the notification is activated.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub conversation_id: Option<String>,
}

/// Checks whether the operating system allows notifications.
///
/// Desktop applications are granted the permission implicitly, so this only
/// ever reports `Granted`, but it keeps the call sites honest about failures.
pub fn permission_state<R: Runtime>(app: &AppHandle<R>) -> PermissionState {
    match app.notification().permission_state() {
        Ok(state) => state,
        Err(error) => {
            eprintln!("[cordis] failed to read notification permission: {error}");
            PermissionState::Denied
        }
    }
}

/// Requests the notification permission and reports the resulting state.
pub fn request_permission<R: Runtime>(app: &AppHandle<R>) -> PermissionState {
    match app.notification().request_permission() {
        Ok(state) => state,
        Err(error) => {
            eprintln!("[cordis] failed to request notification permission: {error}");
            PermissionState::Denied
        }
    }
}

/// Emits [`NOTIFY_REQUEST_EVENT`] to the main window.
pub fn emit_notify_request<R: Runtime>(app: &AppHandle<R>, request: NotifyRequest) {
    if let Err(error) = app.emit(NOTIFY_REQUEST_EVENT, request) {
        eprintln!("[cordis] failed to emit notify request: {error}");
    }
}

/// Shows an operating system notification.
///
/// Failures are logged instead of propagated because a notification that
/// cannot be displayed must never interrupt the message that triggered it.
pub fn show_native_notification<R: Runtime>(
    app: &AppHandle<R>,
    title: &str,
    body: &str,
) -> Result<(), String> {
    app.notification()
        .builder()
        .title(title)
        .body(body)
        .show()
        .map_err(|error| error.to_string())
}

/// Shows a native notification only when the user granted the permission.
pub fn show_native_notification_if_allowed<R: Runtime>(
    app: &AppHandle<R>,
    title: &str,
    body: &str,
) -> Result<(), String> {
    if permission_state(app) != PermissionState::Granted {
        request_permission(app);
    }
    show_native_notification(app, title, body)
}