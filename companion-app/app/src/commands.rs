//! The two commands the page may call; `build.rs` and `capabilities/default.json` name them.

use tauri::{AppHandle, WebviewWindow};
use tauri_plugin_dialog::DialogExt;

use crate::state::{self, Snapshot};

/// The page's own state, read when it loads.
#[tauri::command]
pub fn get_ui_state(app: AppHandle) -> Snapshot {
    state::snapshot(&app)
}

/// Let the member choose the game folder. The dialog blocks a worker thread (this command is
/// async, so not the main thread) until it closes. Held in memory only; nothing reads the folder
/// yet. `None` means the dialog was cancelled.
#[tauri::command]
pub async fn pick_game_folder(app: AppHandle, window: WebviewWindow) -> Option<String> {
    let picked = app.dialog().file().set_parent(&window).blocking_pick_folder()?;
    let folder = picked.to_string();
    state::set_folder(&app, folder.clone());
    Some(folder)
}
