"use strict";

// The page asks Rust for its state and shows what comes back. Anything that came from outside
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
    var button = el("pick");
    // One dialog at a time: a double-click must not open two. A cancelled pick returns null.
    button.disabled = true;
    tauri.core.invoke("pick_game_folder")
      .then(function (state) { if (state) { render(state); } })
      .catch(function () {})
      .finally(function () { button.disabled = false; });
  });

  tauri.core.invoke("get_ui_state").then(render).catch(function () {});
})();
