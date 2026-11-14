//! The one window, built on demand from the template in `tauri.conf.json` (`create: false`) and
//! destroyed on close. `open_main` is the only way it is ever shown.

use tauri::webview::PageLoadEvent;
use tauri::{AppHandle, Manager, WebviewWindow, WebviewWindowBuilder};

const LABEL: &str = "main";

/// Show the window: bring the existing one forward, or build it. Safe to call from any thread.
pub fn open_main(app: &AppHandle) {
    let on_main = app.clone();
    if let Err(error) = app.run_on_main_thread(move || open_now(&on_main)) {
        crate::debug_log(&format!("could not open the window: {error}"));
    }
}

fn open_now(app: &AppHandle) {
    if let Some(window) = app.get_webview_window(LABEL) {
        bring_forward(&window);
        return;
    }
    let Some(config) = app.config().app.windows.iter().find(|w| w.label == LABEL) else {
        crate::debug_log("no window template named main in tauri.conf.json");
        return;
    };
    let built = WebviewWindowBuilder::from_config(app, config).and_then(|builder| {
        // Hidden until the page has loaded, so there is no white flash.
        builder
            .on_page_load(|window, payload| {
                if payload.event() == PageLoadEvent::Finished {
                    bring_forward(&window);
                }
            })
            .build()
    });
    if let Err(error) = built {
        crate::debug_log(&format!("could not build the window: {error}"));
    }
}

fn bring_forward(window: &WebviewWindow) {
    // Each step can fail harmlessly (window gone, not minimized); none is worth stopping for.
    let _ = window.unminimize();
    let _ = window.show();
    let _ = window.set_focus();
}
