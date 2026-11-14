//! The one owner of what the page shows: the chosen game folder, in memory only. Writers call a
//! setter; the page reads `snapshot` (command) or the `ui-state` event, and both come from the
//! same function so they cannot disagree. The next state field goes here too.

use std::sync::{Mutex, MutexGuard};

use serde::Serialize;
use tauri::{AppHandle, Emitter, Manager};

/// Event name the page listens for.
pub const UI_STATE_EVENT: &str = "ui-state";

#[derive(Default)]
struct Inner {
    folder: Option<String>,
}

/// Managed state. Not persisted: a restart starts empty.
#[derive(Default)]
pub struct UiState(Mutex<Inner>);

impl UiState {
    fn lock(&self) -> MutexGuard<'_, Inner> {
        // A poisoned lock only means a panic elsewhere; the string is still valid.
        self.0.lock().unwrap_or_else(|poisoned| poisoned.into_inner())
    }
}

/// What the page receives, as a command result and as the event payload.
#[derive(Clone, Serialize)]
pub struct Snapshot {
    pub version: &'static str,
    pub folder: Option<String>,
}

pub fn snapshot(app: &AppHandle) -> Snapshot {
    let state = app.state::<UiState>();
    let inner = state.lock();
    Snapshot {
        version: forever_friends_core::version(),
        folder: inner.folder.clone(),
    }
}

/// Snapshot and tell the page. With no window open the event reaches nobody; the page asks for
/// the snapshot itself when it loads.
fn publish(app: &AppHandle) {
    if let Err(error) = app.emit_to("main", UI_STATE_EVENT, snapshot(app)) {
        crate::debug_log(&format!("could not emit {UI_STATE_EVENT}: {error}"));
    }
}

pub fn set_folder(app: &AppHandle, folder: String) {
    app.state::<UiState>().lock().folder = Some(folder);
    publish(app);
}
