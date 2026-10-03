//! In-app update flow.
//!
//! Every step of the update lifecycle is mirrored to the frontend through the
//! `updater://status` event so the settings dialog can render a progress bar
//! without polling. The check itself is delegated to the official updater
//! plugin, which verifies the minisign signature of the downloaded bundle
//! before installing it.

use serde::Serialize;
use std::sync::{Arc, Mutex};
use tauri::{AppHandle, Emitter, Runtime};
use tauri_plugin_updater::UpdaterExt;

/// Event carrying every update lifecycle change.
pub const UPDATER_STATUS_EVENT: &str = "updater://status";

/// Lifecycle phase of an update operation.
#[derive(Debug, Clone, Copy, Serialize, PartialEq, Eq)]
#[serde(rename_all = "camelCase")]
pub enum UpdatePhase {
    /// Nothing is happening.
    Idle,
    /// A check against the release endpoint is in flight.
    Checking,
    /// A newer version was announced by the release endpoint.
    Available,
    /// The bundle is being downloaded.
    Downloading,
    /// The bundle finished downloading and is about to be installed.
    Downloaded,
    /// The installer is running.
    Installing,
    /// The operation failed, the message carries the reason.
    Error,
}

/// Payload delivered with [`UPDATER_STATUS_EVENT`].
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct UpdaterStatus {
    /// Current phase of the operation.
    pub phase: UpdatePhase,
    /// Download progress between 0 and 100, zero for the other phases.
    pub percent: u64,
    /// Version offered by the release endpoint, when one was found.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub version: Option<String>,
    /// Version currently installed on this machine.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub current_version: Option<String>,
    /// Release notes of the available update.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub body: Option<String>,
    /// Failure reason, only present for [`UpdatePhase::Error`].
    #[serde(skip_serializing_if = "Option::is_none")]
    pub message: Option<String>,
}

/// Accumulates the downloaded byte count across `on_chunk` callbacks.
///
/// The updater callback reports the size of each chunk plus the total length,
/// so the running total has to be kept by the caller to compute a percentage.
#[derive(Debug, Default)]
pub struct ProgressTracker {
    downloaded: Mutex<u64>,
    total: Mutex<u64>,
}

impl ProgressTracker {
    /// Creates a tracker with no progress recorded yet.
    pub fn new() -> Self {
        Self::default()
    }

    /// Records a chunk and returns the percentage completed so far.
    fn record(&self, chunk: usize, total: Option<u64>) -> u64 {
        let (Ok(mut downloaded), Ok(mut known_total)) = (self.downloaded.lock(), self.total.lock())
        else {
            // A poisoned lock only means a previous callback panicked, and the
            // download cannot be resumed meaningfully, so report no progress.
            return 0;
        };

        *downloaded += chunk as u64;
        if let Some(total) = total {
            *known_total = total;
        }
        if *known_total == 0 {
            return 0;
        }

        ((*downloaded * 100) / *known_total).min(100)
    }
}

/// Builds an [`UpdaterStatus`] with only the fields that are always present.
fn status(phase: UpdatePhase, percent: u64) -> UpdaterStatus {
    UpdaterStatus {
        phase,
        percent,
        version: None,
        current_version: None,
        body: None,
        message: None,
    }
}

/// Emits an update status to the frontend, logging transport failures.
fn emit<R: Runtime>(app: &AppHandle<R>, payload: UpdaterStatus) {
    if let Err(error) = app.emit(UPDATER_STATUS_EVENT, payload) {
        eprintln!("[cordis] failed to emit updater status: {error}");
    }
}

/// Builds a status carrying the versions known at the time of the call.
fn versioned(
    phase: UpdatePhase,
    percent: u64,
    version: Option<String>,
    current_version: Option<String>,
) -> UpdaterStatus {
    UpdaterStatus {
        version,
        current_version,
        ..status(phase, percent)
    }
}

/// Compares two dotted version strings ignoring an optional `v` prefix.
///
/// The updater plugin only offers releases that are newer than the running
/// build, this helper exists so the shell can log and report the comparison
/// without pulling the semver crate into the dependency tree.
pub fn is_newer(candidate: &str, current: &str) -> bool {
    let parse = |value: &str| -> Vec<u64> {
        value
            .trim()
            .trim_start_matches(['v', 'V'])
            .split(['.', '-'])
            .map(|part| part.parse::<u64>().unwrap_or(0))
            .collect()
    };

    let candidate = parse(candidate);
    let current = parse(current);

    for index in 0..candidate.len().max(current.len()) {
        let left = candidate.get(index).copied().unwrap_or(0);
        let right = current.get(index).copied().unwrap_or(0);
        if left != right {
            return left > right;
        }
    }

    false
}

/// Runs the full update cycle: check, download and install.
///
/// Returns the version that was installed, or `None` when the machine already
/// runs the latest release. Errors are reported through the event stream and
/// returned as `Err` so a Rust caller can react to them as well.
pub async fn check_for_update<R: Runtime>(app: AppHandle<R>) -> Result<Option<String>, String> {
    let current_version = app.package_info().version.to_string();
    let current = Some(current_version.clone());

    emit(&app, versioned(UpdatePhase::Checking, 0, None, current.clone()));

    let updater = match app.updater() {
        Ok(updater) => updater,
        Err(error) => {
            let message = error.to_string();
            emit(
                &app,
                UpdaterStatus {
                    message: Some(message.clone()),
                    ..versioned(UpdatePhase::Error, 0, None, current)
                },
            );
            return Err(message);
        }
    };

    let update = match updater.check().await {
        Ok(update) => update,
        Err(error) => {
            let message = error.to_string();
            emit(
                &app,
                UpdaterStatus {
                    message: Some(message.clone()),
                    ..versioned(UpdatePhase::Error, 0, None, current)
                },
            );
            return Err(message);
        }
    };

    let Some(update) = update else {
        emit(&app, versioned(UpdatePhase::Idle, 0, None, current));
        return Ok(None);
    };

    let version = update.version.clone();
    let announced = Some(version.clone());

    if !is_newer(&version, &current_version) {
        emit(&app, versioned(UpdatePhase::Idle, 0, announced, current));
        return Ok(None);
    }

    emit(
        &app,
        UpdaterStatus {
            body: update.body.clone(),
            ..versioned(UpdatePhase::Available, 0, announced.clone(), current.clone())
        },
    );
    emit(
        &app,
        versioned(
            UpdatePhase::Downloading,
            0,
            announced.clone(),
            current.clone(),
        ),
    );

    let progress_app = app.clone();
    let progress_version = version.clone();
    let tracker = Arc::new(ProgressTracker::new());

    let install_result = update
        .download_and_install(
            {
                let tracker = Arc::clone(&tracker);
                move |chunk, total| {
                    let percent = tracker.record(chunk, total);
                    emit(
                        &progress_app,
                        versioned(
                            UpdatePhase::Downloading,
                            percent,
                            Some(progress_version.clone()),
                            None,
                        ),
                    );
                }
            },
            {
                let finished_app = app.clone();
                let finished_version = version.clone();
                move || {
                    emit(
                        &finished_app,
                        versioned(
                            UpdatePhase::Downloaded,
                            100,
                            Some(finished_version.clone()),
                            None,
                        ),
                    );
                }
            },
        )
        .await;

    match install_result {
        Ok(()) => {
            // On Windows the installer replaces this executable, so the
            // frontend learns the install started before the process goes away.
            emit(&app, versioned(UpdatePhase::Installing, 100, announced, current));
            Ok(Some(version))
        }
        Err(error) => {
            let message = error.to_string();
            emit(
                &app,
                UpdaterStatus {
                    message: Some(message.clone()),
                    ..versioned(UpdatePhase::Error, 0, announced, current)
                },
            );
            Err(message)
        }
    }
}