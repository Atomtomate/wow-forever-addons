//! The tray icon and its menu. No operating-system branches here: labels come from `platform`.

use tauri::AppHandle;
use tauri::menu::{CheckMenuItem, Menu, MenuItem, PredefinedMenuItem};
use tauri::tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent};
use tauri_plugin_autostart::ManagerExt;

use crate::{platform, window};

const ID_OPEN: &str = "open";
const ID_AUTOSTART: &str = "autostart";
const ID_QUIT: &str = "quit";

pub fn build(app: &AppHandle) -> tauri::Result<()> {
    let open = MenuItem::with_id(app, ID_OPEN, "Open Forever Friends", true, None::<&str>)?;
    let status = MenuItem::with_id(app, "status", "Not connected yet", false, None::<&str>)?;
    let separator = PredefinedMenuItem::separator(app)?;
    let enabled = app.autolaunch().is_enabled().unwrap_or(false);
    let autostart = CheckMenuItem::with_id(
        app,
        ID_AUTOSTART,
        platform::autostart_label(),
        true,
        enabled,
        None::<&str>,
    )?;
    let quit = MenuItem::with_id(app, ID_QUIT, "Quit", true, None::<&str>)?;
    let menu = Menu::with_items(app, &[&open, &status, &separator, &autostart, &quit])?;

    let mut builder = TrayIconBuilder::with_id("main")
        .tooltip("Forever Friends")
        .menu(&menu)
        // Left click opens the window (not on Linux, which sends no click events: the menu's
        // "Open" is the way in there); right click shows the menu.
        .show_menu_on_left_click(false)
        .on_menu_event(move |app, event| match event.id().as_ref() {
            ID_OPEN => window::open_main(app),
            ID_AUTOSTART => toggle_autostart(app, &autostart),
            ID_QUIT => app.exit(0),
            _ => {}
        })
        .on_tray_icon_event(|tray, event| {
            if let TrayIconEvent::Click {
                button: MouseButton::Left,
                button_state: MouseButtonState::Up,
                ..
            } = event
            {
                window::open_main(tray.app_handle());
            }
        });
    if let Some(icon) = app.default_window_icon() {
        builder = builder.icon(icon.clone());
    }
    builder.build(app)?;
    Ok(())
}

/// The check box has already flipped when this runs; make the system agree, or flip it back.
fn toggle_autostart(app: &AppHandle, item: &CheckMenuItem<tauri::Wry>) {
    let want = item.is_checked().unwrap_or(false);
    let manager = app.autolaunch();
    let result = if want { manager.enable() } else { manager.disable() };
    if let Err(error) = result {
        crate::debug_log(&format!("could not change autostart: {error}"));
        let _ = item.set_checked(!want);
    }
}
