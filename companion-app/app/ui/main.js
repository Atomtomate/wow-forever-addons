"use strict";

// The page asks Rust for its state and listens for changes. Anything that came from outside
// (the folder path) is written with textContent only, never as markup.
(function () {
  var tauri = window.__TAURI__;
  var el = function (id) { return document.getElementById(id); };

  function render(state) {
    el("version").textContent = state.version || "";
    el("folder").textContent = state.folder || "Not chosen yet";
  }

  // No right-click menu: this is an app window, not a web page.
  document.addEventListener("contextmenu", function (event) { event.preventDefault(); });

  el("pick").addEventListener("click", function () {
    // The result arrives through the "ui-state" event too; the catch is for a refused call.
    tauri.core.invoke("pick_game_folder").catch(function () {});
  });

  tauri.event.listen("ui-state", function (event) { render(event.payload); });
  tauri.core.invoke("get_ui_state").then(render).catch(function () {});
})();
