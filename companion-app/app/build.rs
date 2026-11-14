fn main() {
    // Listing the app's own commands here makes each one need an explicit permission in
    // `capabilities/`, so the page can call only what a capability names.
    let app = tauri_build::AppManifest::new().commands(&["pick_game_folder", "get_ui_state"]);
    tauri_build::try_build(tauri_build::Attributes::new().app_manifest(app)).expect("failed to run tauri-build");
}
