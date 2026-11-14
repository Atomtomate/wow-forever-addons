//! Every operating-system difference, one small function each. Callers (`main`, `tray`) carry
//! no `cfg`; the next OS-specific need goes here too.

use tauri::App;

/// macOS: tray only, no dock icon. Elsewhere there is nothing to do.
#[cfg(target_os = "macos")]
pub fn apply_activation_policy(app: &mut App) {
    app.set_activation_policy(tauri::ActivationPolicy::Accessory);
}

#[cfg(not(target_os = "macos"))]
pub fn apply_activation_policy(_app: &mut App) {}

/// What the autostart toggle in the tray menu is called on this system.
pub fn autostart_label() -> &'static str {
    if cfg!(target_os = "windows") {
        "Start with Windows"
    } else if cfg!(target_os = "macos") {
        "Open at login"
    } else {
        "Start at login"
    }
}
