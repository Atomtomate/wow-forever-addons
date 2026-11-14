//! First start: say once, with one notification, that the app lives in the tray. A marker file in
//! the app's own data folder remembers it. Failures are logged and never stop the app.

use tauri::{AppHandle, Manager};
use tauri_plugin_notification::NotificationExt;

const MARKER: &str = "first-start-done";

pub fn run(app: &AppHandle) {
    let dir = match app.path().app_data_dir() {
        Ok(dir) => dir,
        Err(error) => return crate::debug_log(&format!("no app data folder: {error}")),
    };
    let marker = dir.join(MARKER);
    if marker.exists() {
        return;
    }
    let shown = app
        .notification()
        .builder()
        .title("Forever Friends")
        .body("Forever Friends is in your tray.")
        .show();
    if let Err(error) = shown {
        crate::debug_log(&format!("could not show the first-start notice: {error}"));
    }
    // Written even if the notice failed: better to stay quiet than to repeat it every start.
    if let Err(error) = std::fs::create_dir_all(&dir).and_then(|()| std::fs::write(&marker, b"")) {
        crate::debug_log(&format!("could not write the first-start marker: {error}"));
    }
}
