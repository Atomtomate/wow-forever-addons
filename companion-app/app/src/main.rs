//! Forever Friends companion app: a tray-first Tauri 2 shell. Skeleton only, see `README.md`.
//!
//! This crate is thin glue. Logic that needs no GUI lives in `forever-friends-core`; every
//! operating-system difference is one function in `platform.rs`.

#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod commands;
mod first_start;
mod platform;
mod state;
mod tray;
mod window;

use tauri::RunEvent;

/// Debug builds print to stderr; release builds are silent (no log file, nothing leaves the app).
pub(crate) fn debug_log(message: &str) {
    if cfg!(debug_assertions) {
        eprintln!("[forever-friends] {message}");
    }
}

fn main() {
    let app = tauri::Builder::default()
        // Single-instance first, so a second start is caught before anything else; it just
        // brings the first instance's window forward.
        .plugin(tauri_plugin_single_instance::init(|app, _argv, _cwd| {
            window::open_main(app)
        }))
        // Never enabled at start; the tray toggle is the member's choice. macOS defaults to a
        // launch agent. The name must equal `productName` in tauri.conf.json: the NSIS
        // uninstaller removes the Run entry by that name.
        .plugin(
            tauri_plugin_autostart::Builder::new()
                .app_name("Forever Friends")
                .build(),
        )
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_dialog::init())
        .manage(state::UiState::default())
        .invoke_handler(tauri::generate_handler![
            commands::pick_game_folder,
            commands::get_ui_state
        ])
        .setup(|app| {
            platform::apply_activation_policy(app);
            let handle = app.handle();
            tray::build(handle)?;
            first_start::run(handle);
            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("error while building the Forever Friends app");

    app.run(|_app, event| {
        // Closing the window must not end the program. Quit in the tray calls `exit(0)`, which
        // carries a code and so is not prevented here.
        if let RunEvent::ExitRequested { code: None, api, .. } = event {
            api.prevent_exit();
        }
    });
}
