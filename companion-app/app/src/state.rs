//! What the page shows: the chosen game folder, in memory only. When something other than the
//! page starts writing state, that is the time to add a way to tell the page.

use std::sync::Mutex;

use serde::Serialize;
use tauri::{AppHandle, Manager};

/// Managed state. Not persisted: a restart starts empty.
#[derive(Default)]
pub struct UiState(Mutex<Option<String>>);

/// What the page receives.
#[derive(Clone, Serialize)]
pub struct Snapshot {
    pub version: &'static str,
    pub folder: Option<String>,
}

pub fn snapshot(app: &AppHandle) -> Snapshot {
    let folder = app.state::<UiState>().0.lock().expect("state lock").clone();
    Snapshot {
        version: forever_friends_core::version(),
        folder,
    }
}

pub fn set_folder(app: &AppHandle, folder: String) {
    *app.state::<UiState>().0.lock().expect("state lock") = Some(folder);
}
