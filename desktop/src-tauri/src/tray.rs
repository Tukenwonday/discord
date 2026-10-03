//! System tray integration.
//!
//! The tray owns the mute, deafen and presence state of the shell. Clicking a
//! check item never mutates the menu on its own: it updates [`TrayState`],
//! rebuilds the menu so the check marks stay truthful and then tells the
//! frontend what the user asked for through the `tray://action` event, which
//! makes the WebSocket the single source of truth for the session.

use serde::Serialize;
use serde_json::{json, Value};
use std::sync::Mutex;
use tauri::menu::{CheckMenuItem, Menu, MenuBuilder, MenuItem, PredefinedMenuItem, SubmenuBuilder};
use tauri::tray::{
    MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent,
};
use tauri::{AppHandle, Emitter, Manager, Runtime};

use crate::window;

/// Identifier of the tray icon, shared with `tauri.conf.json`.
pub const TRAY_ID: &str = "cordis-tray";

/// Event carrying a user interaction with the tray.
pub const TRAY_ACTION_EVENT: &str = "tray://action";

/// Presence value meaning the user is reachable.
pub const STATUS_ONLINE: &str = "online";
/// Presence value meaning the user is reachable but idle.
pub const STATUS_IDLE: &str = "idle";
/// Presence value meaning notifications are suppressed.
pub const STATUS_DND: &str = "dnd";
/// Presence value meaning the user appears offline.
pub const STATUS_INVISIBLE: &str = "invisible";

/// Every presence value the status submenu offers, in display order.
pub const STATUS_OPTIONS: [&str; 4] =
    [STATUS_ONLINE, STATUS_IDLE, STATUS_DND, STATUS_INVISIBLE];

/// Tray owned session flags.
///
/// It is serializable because the frontend seeds its own presence store from
/// [`crate::commands::tray_state`].
#[derive(Debug, Clone, Serialize)]
pub struct TrayState {
    /// Whether the microphone is muted.
    pub muted: bool,
    /// Whether the microphone and speakers are deafened.
    pub deafened: bool,
    /// Current presence, one of [`STATUS_OPTIONS`].
    pub status: String,
}

impl Default for TrayState {
    fn default() -> Self {
        Self {
            muted: false,
            deafened: false,
            status: STATUS_ONLINE.to_string(),
        }
    }
}

/// Payload delivered with [`TRAY_ACTION_EVENT`].
///
/// `value` is a boolean for `mute` and `deafen` and a string for `status`,
/// which keeps the frontend handler free of per action parsing.
#[derive(Debug, Clone, Serialize)]
pub struct TrayAction {
    /// Identifier of the interacted control.
    pub action: String,
    /// New value of the control.
    pub value: Value,
}

/// Type alias for the managed tray state.
pub type SharedTrayState = Mutex<TrayState>;

/// Emits a tray action to the frontend.
fn emit_action<R: Runtime>(app: &AppHandle<R>, action: &str, value: Value) {
    let payload = TrayAction {
        action: action.to_string(),
        value,
    };
    if let Err(error) = app.emit(TRAY_ACTION_EVENT, payload) {
        eprintln!("[cordis] failed to emit tray action: {error}");
    }
}

/// Reads the current tray state, falling back to the default when the lock is
/// poisoned by a panicking menu callback.
fn snapshot<R: Runtime>(app: &AppHandle<R>) -> TrayState {
    app.try_state::<SharedTrayState>()
        .and_then(|state| state.lock().ok().map(|inner| inner.clone()))
        .unwrap_or_default()
}

/// Applies a mutation to the tray state.
fn mutate<R: Runtime>(app: &AppHandle<R>, apply: impl FnOnce(&mut TrayState)) -> TrayState {
    if let Some(state) = app.try_state::<SharedTrayState>() {
        match state.lock() {
            Ok(mut inner) => {
                apply(&mut inner);
                return inner.clone();
            }
            Err(error) => eprintln!("[cordis] tray state lock poisoned: {error}"),
        }
    }
    TrayState::default()
}

/// Builds the whole tray menu for a given state.
///
/// The menu is rebuilt instead of toggling individual items because the status
/// submenu has to move its check mark between four radio-like entries.
pub fn build_menu<R: Runtime>(
    app: &AppHandle<R>,
    state: &TrayState,
) -> tauri::Result<Menu<R>> {
    let show = MenuItem::with_id(app, "show", "Show Cordis", true, None::<&str>)?;
    let separator_top = PredefinedMenuItem::separator(app)?;
    let mute = CheckMenuItem::with_id(app, "mute", "Mute", true, state.muted, None::<&str>)?;
    let deafen =
        CheckMenuItem::with_id(app, "deafen", "Deafen", true, state.deafened, None::<&str>)?;
    let separator_bottom = PredefinedMenuItem::separator(app)?;
    let quit = MenuItem::with_id(app, "quit", "Quit", true, None::<&str>)?;

    let online = CheckMenuItem::with_id(
        app,
        "status-online",
        "Online",
        true,
        state.status == STATUS_ONLINE,
        None::<&str>,
    )?;
    let idle = CheckMenuItem::with_id(
        app,
        "status-idle",
        "Idle",
        true,
        state.status == STATUS_IDLE,
        None::<&str>,
    )?;
    let dnd = CheckMenuItem::with_id(
        app,
        "status-dnd",
        "Do Not Disturb",
        true,
        state.status == STATUS_DND,
        None::<&str>,
    )?;
    let invisible = CheckMenuItem::with_id(
        app,
        "status-invisible",
        "Invisible",
        true,
        state.status == STATUS_INVISIBLE,
        None::<&str>,
    )?;

    let status = SubmenuBuilder::new(app, "Status")
        .items(&[&online, &idle, &dnd, &invisible])
        .build()?;

    let items: [&dyn tauri::menu::IsMenuItem<R>; 7] = [
        &show,
        &separator_top,
        &mute,
        &deafen,
        &status,
        &separator_bottom,
        &quit,
    ];

    MenuBuilder::new(app).items(&items).build()
}

/// Handles a click on a tray menu item.
///
/// Unknown identifiers are ignored on purpose so a future menu entry can be
/// added without breaking older builds of the frontend.
pub fn handle_menu_event<R: Runtime>(app: &AppHandle<R>, id: &str) {
    match id {
        "show" => {
            window::focus_main_window(app);
            emit_action(app, "show", json!(true));
        }
        "mute" => {
            let state = mutate(app, |state| state.muted = !state.muted);
            emit_action(app, "mute", json!(state.muted));
        }
        "deafen" => {
            let state = mutate(app, |state| {
                // Deafening implies muting so the tray never shows an active
                // microphone while the user asked for silence.
                state.deafened = !state.deafened;
                state.muted = state.deafened || state.muted;
            });
            emit_action(app, "deafen", json!(state.deafened));
        }
        "status-online" | "status-idle" | "status-dnd" | "status-invisible" => {
            let status = match id {
                "status-idle" => STATUS_IDLE,
                "status-dnd" => STATUS_DND,
                "status-invisible" => STATUS_INVISIBLE,
                _ => STATUS_ONLINE,
            };
            mutate(app, |state| state.status = status.to_string());
            emit_action(app, "status", json!(status));
        }
        "quit" => app.exit(0),
        other => eprintln!("[cordis] unhandled tray menu id: {other}"),
    }
}

/// Applies a state change coming from the frontend and refreshes the menu.
///
/// This keeps the tray truthful when the user mutes from the title bar or when
/// the session expires and the backend forces the presence back to online.
pub fn sync_state<R: Runtime>(
    app: &AppHandle<R>,
    muted: Option<bool>,
    deafened: Option<bool>,
    status: Option<String>,
) -> TrayState {
    let updated = mutate(app, |state| {
        if let Some(muted) = muted {
            state.muted = muted;
        }
        if let Some(deafened) = deafened {
            state.deafened = deafened;
            if deafened {
                state.muted = true;
            }
        }
        if let Some(status) = status {
            if STATUS_OPTIONS.contains(&status.as_str()) {
                state.status = status;
            }
        }
    });

    if let Err(error) = refresh_menu(app, &updated) {
        eprintln!("[cordis] failed to refresh tray menu: {error}");
    }

    updated
}

/// Rebuilds the menu of the existing tray icon.
///
/// Returns a message instead of a `tauri::Error` because the only realistic
/// failure is the tray icon being absent, which is a shell state problem and
/// not a Tauri level fault.
pub fn refresh_menu<R: Runtime>(app: &AppHandle<R>, state: &TrayState) -> Result<(), String> {
    let menu = build_menu(app, state).map_err(|error| error.to_string())?;
    match app.tray_by_id(TRAY_ID) {
        Some(tray) => tray
            .set_menu(Some(menu))
            .map_err(|error| error.to_string()),
        None => Err(format!("tray icon {TRAY_ID} is not registered")),
    }
}

/// Builds the tray icon and registers its menu and click handlers.
pub fn build_tray(app: &AppHandle) -> tauri::Result<()> {
    let state = snapshot(app);
    let menu = build_menu(app, &state)?;

    let mut builder = TrayIconBuilder::with_id(TRAY_ID)
        .tooltip("Cordis")
        .menu(&menu)
        // The menu opens on right click, the left click is reserved for
        // toggling the window visibility.
        .show_menu_on_left_click(false)
        .on_menu_event(|app, event| {
            handle_menu_event(app, event.id().as_ref());
        })
        .on_tray_icon_event(|tray, event| {
            if let TrayIconEvent::Click {
                button: MouseButton::Left,
                button_state: MouseButtonState::Up,
                ..
            } = event
            {
                window::toggle_visibility(tray.app_handle());
            }
        });

    if let Some(icon) = app.default_window_icon() {
        builder = builder.icon(icon.clone());
    }

    builder.build(app)?;
    Ok(())
}

/// Returns the tray state as JSON so the frontend can seed its own store.
pub fn state_snapshot(app: &AppHandle) -> TrayState {
    snapshot(app)
}