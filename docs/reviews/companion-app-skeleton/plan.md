> Superseded where it mentions the deep link: the owner removed it from this piece on 2026-11-14 (D9, PR #5).

# Plan: `companion-app/`, the Forever Friends tray app skeleton (Rust + Tauri 2)

Base `main` at `d31312e`. Mode: large (a new party), no change to `docs/CONTRACT.md` formats.
One specialist, `impl-backend`, one worktree, one pull request. Checked on 2026-11-14 against
crates.io and the crate sources (v2.tauri.app is blocked from this container; docs.rs not needed).

## 0. Open decisions, first (owner or coordinating session)

1. **This contradicts D7 as written and `docs/IDEAS.md` ("a tray icon ... later").** D7 says the
   companion is Go, standard library only, one console executable. The owner's brief is the
   direction, but `DECISIONS.md` is append-only: the coordinating session writes **D9** (Rust +
   Tauri 2 desktop app as the future companion; Go companion stays the shipped one until the
   app reaches parity; the Tauri crates below are the approved dependency set; WebView2 is a
   runtime dependency on Windows) before or alongside this PR. Not by the specialist.
2. **Identifier.** Proposed `io.github.atomtomate.foreverfriends` (owner's own namespace, no hub
   address; must not end in `.app`). It names the app data folder, the Windows notification
   identity (AUMID) and the Run-key entry, so changing it later moves the marker file (one more
   "first start" notification) — harmless before the first release. Alternative: a neutral
   `org.foreverfriends.app`-style name the owner owns. Owner picks; the specialist uses the
   proposal until told.
3. **WebView2 install mode: `downloadBootstrapper` (silent), the default — recommended.** Win11
   and current Win10 ship the Evergreen runtime, so the bootstrapper almost never runs.
   `embedBootstrapper` adds ~1.8 MB and *still downloads* the runtime, so it buys nothing;
   only `offlineInstaller` (~130 MB) works offline. Owner confirms.
4. **Should `make check` include `rslint rstest`?** Recommended: **no** for now. They need the
   Rust toolchain plus webkit2gtk/appindicator dev libraries on Linux; `check` would then fail on
   every machine that only works on Lua or Go. Pre-commit hook unchanged (it runs `make lint
   test` semantics and Go checks only). Owner decides; if yes, it is a one-line change later.
5. **Deep-link parsing in the core crate** (sketch, recommended): the brief says core holds
   only a version fn. Proposal: also `parse_connect_link(&str) -> Option<String>` there, pure
   std, so it is tested by `cargo test -p` core with no system libraries. Owner may veto; then
   it lives in the app crate with its tests.
6. **macOS dock icon:** `ActivationPolicy::Accessory` (no dock icon, tray only) — recommended,
   consistent with tray-first. Side effect: an open window has no dock/Cmd-Tab entry. Owner may
   prefer `Regular`.

### Changes the other (coordinating) session makes — not in this PR
- `docs/DECISIONS.md`: D9 (item 1). `docs/IDEAS.md`: tray icon entry moves/points to D9.
- `docs/CONTRACT.md` §5: outside the game folder the app keeps a **marker file
  `first-start-done` in the app data dir** (`%APPDATA%\<identifier>\` on Windows,
  `~/Library/Application Support/<identifier>/`, `~/.local/share/<identifier>/`), an **HKCU Run
  entry** when "Start with Windows" is on, an **HKCU URL-scheme registration
  `foreverfriends://`** (written by the installer, removed by uninstall), its install folder
  (`%LOCALAPPDATA%\Forever Friends\` under currentUser NSIS) and WebView2's own data folder
  (`%LOCALAPPDATA%\<identifier>\EBWebView`). The Go companion's folder is
  `%LOCALAPPDATA%\ForeverFriends\` — different, no collision.
- `.claude/agents.local.md`: add `companion-app/` to `impl-backend`'s row; add a Rust standard
  for the architecture reviewer (core crate has no Tauri dep and no I/O; app crate thin).
- `CLAUDE.md` working agreement: `make rslint rstest` when a file under `companion-app/` changed.
- `docs/HANDOFF.md` and later `docs/RESEARCH.md` "Observed" from the Verify sheet.

## 1. Files to create (all by `impl-backend`)

```
companion-app/
  Cargo.toml            workspace: members ["core","app"], resolver "3", [workspace.package]
                        version "0.0.1", edition "2024", rust-version "1.90", license matches repo
  Cargo.lock            committed (binary)
  .gitignore            /target, app/gen/schemas
  README.md             layout, build, Verify on the PC sheet (section 6)
  ASSETS.md             icon origin: drawn here, SVG source, command that made the PNG/ICO/ICNS
  core/Cargo.toml       name "forever-friends-core", no dependencies
  core/src/lib.rs       pub fn version() -> &'static str (env!("CARGO_PKG_VERSION")); link parser; tests
  app/Cargo.toml        name "forever-friends", [[bin]] name "forever-friends"; deps below
  app/build.rs          tauri_build with an AppManifest listing the app commands (so each needs
                        an explicit permission — sketch)
  app/tauri.conf.json   section 3
  app/capabilities/default.json
  app/src/main.rs       cfg_attr(not(debug_assertions), windows_subsystem = "windows"); builder
  app/src/tray.rs, window.rs, links.rs, first_start.rs   (split is a sketch)
  app/ui/index.html, style.css, main.js                  (frontendDist "ui", relative to tauri.conf.json)
  app/icons/icon.svg + generated 32x32.png 128x128.png 128x128@2x.png icon.png icon.ico icon.icns
.github/workflows/companion-app.yml
Makefile                (two targets appended; existing ones untouched)
```

## 2. Crates (constraint: nothing else without asking)

| crate | version | features |
|---|---|---|
| `tauri` | `2.12.5` | `tray-icon` (defaults kept) |
| `tauri-build` (build-dep) | `2.7.4` | default |
| `tauri-plugin-single-instance` | `2.5.5` | `deep-link` |
| `tauri-plugin-deep-link` | `2.6.3` | — |
| `tauri-plugin-autostart` | `2.7.0` | — |
| `tauri-plugin-notification` | `2.8.1` | — |
| `tauri-plugin-dialog` | `2.8.4` | default (`gtk3`) |
| `serde` (+`serde_json` only if `generate_context`/commands need it) | `1` | `derive` |
| `forever-friends-core` | path | — |

All MSRV 1.90; container has 1.97. URL parsing uses `tauri::Url` (re-export), no `url` dep.
Plain `cargo build` works without the CLI: `tauri-codegen` 2.7.4 embeds `frontendDist` when no
`devUrl` is set (checked in `context.rs`), so no `custom-protocol` feature dance and no devUrl.

## 3. Config shape

`tauri.conf.json`: `productName "Forever Friends"`, `identifier` (0.2), `version` from Cargo;
`build.frontendDist "ui"`, no `devUrl`, no `beforeBuildCommand`. `app.withGlobalTauri: true`
(decided: no npm, `window.__TAURI__.core.invoke` / `.event.listen`). `app.windows`: one entry
`label "main"`, **`create: false`** (template only; built on demand with
`WebviewWindowBuilder::from_config`), `visible: false`, `backgroundColor` neutral, size ~420x320,
`center`. `app.security.csp`: `default-src 'self'; script-src 'self'; style-src 'self';
img-src 'self'; connect-src ipc: http://ipc.localhost; object-src 'none'; base-uri 'none';
form-action 'none'` (no inline script/style, Tauri adds its hashes). `app.security.freezePrototype: true`.
`plugins.deep-link.desktop.schemes: ["foreverfriends"]`.
`bundle`: `active true`, `targets ["nsis"]`, icons, `publisher`, `shortDescription`;
`bundle.windows.webviewInstallMode {type: "downloadBootstrapper", silent: true}`;
`bundle.windows.nsis {installMode: "currentUser", displayLanguageSelector: false}` (Start-menu
shortcut and Settings > Apps uninstall entry are NSIS defaults). No `plugins.updater`, no
`createUpdaterArtifacts`.

`capabilities/default.json`: `windows ["main"]`, permissions exactly:
`core:event:allow-listen`, `core:event:allow-unlisten`, `allow-pick-game-folder`,
`allow-get-ui-state` (the two app commands). No `core:default`, no `dialog:*`,
`notification:*`, `deep-link:*`, `autostart:*` — those are driven from Rust only, so the
page can do nothing but ask for a folder and read its own state.

## 4. Behaviour and order (constraints marked C)

- **C: plugin order** — `single_instance` (callback: open/focus window) **first**, then
  `deep_link`, `autostart` (`Builder::new().app_name("Forever Friends")`, `MacosLauncher::LaunchAgent`,
  never enabled at start), `notification`, `dialog`. Single-instance with `deep-link` feature
  forwards a second start's argv to deep-link, so `on_open_url` fires in the first instance.
- `setup`: macOS Accessory policy (0.6); tray; `deep_link().on_open_url(..)`; cold start reads
  `deep_link().get_current()`; first-start check; in **debug builds only** on Windows/Linux
  `deep_link().register_all()` (release relies on the installer; Linux release scheme is out of scope).
- Tray: `TrayIconBuilder::with_id`, `default_window_icon`, tooltip, menu: "Open Forever Friends";
  "Not connected yet" (disabled); separator; `CheckMenuItem` "Start with Windows" (label per OS:
  "Open at login" on macOS, "Start at login" on Linux — sketch), checked from `is_enabled()`;
  "Quit" (`app.exit(0)`). `show_menu_on_left_click(false)`; left-click Up opens the window.
  Toggle: enable/disable, on error revert the check and log. Disable removes the Run entry.
- **C: stays in tray** — `run(|app, e|)`: `RunEvent::ExitRequested { code: None, api }` →
  `api.prevent_exit()`; Quit passes `Some(0)` and exits. Window close keeps default (destroy).
- Window open: if `get_webview_window("main")` exists → unminimize, show, set_focus; else build
  from config; show it from `on_page_load` Finished (no white flash; no window permission needed).
- Links: accept only `foreverfriends://connect?code=X`, X 1–32 chars of `[A-Za-z0-9-]`, else
  ignored. Rust keeps the code and the chosen path in a `Mutex` state (memory only), emits
  `ui-state` to the window, opens it. **C:** page writes with `textContent` only.
- Folder: command `pick_game_folder` uses `dialog().file().pick_folder` (async callback, not
  blocking on the main thread), stores the path, returns it. Nothing persisted.
- First start: if `app_data_dir()/first-start-done` is absent → one notification
  ("Forever Friends is in your tray") then create the marker; errors logged, never fatal.
- Page: system font (`system-ui`), `color-scheme: light dark` + `prefers-color-scheme`,
  `user-select: none` on chrome (path/code text selectable), `contextmenu` preventDefault.

## 5. CI and Makefile

`.github/workflows/companion-app.yml`: `on: pull_request` and `push: main` both with
`paths: [companion-app/**, .github/workflows/companion-app.yml]`; `workflow_dispatch`.
`defaults.run.working-directory: companion-app`. Rust via `dtolnay/rust-toolchain@stable`
(`components: rustfmt, clippy`); cache `Swatinem/rust-cache@v2` with
`workspaces: companion-app -> target` (actions, not dependencies). Linux apt: `libwebkit2gtk-4.1-dev
libayatana-appindicator3-dev librsvg2-dev libxdo-dev libssl-dev build-essential file`.
- `check` (ubuntu): `cargo fmt --all --check`; `cargo clippy --workspace --all-targets --locked -- -D warnings`;
  `cargo test --workspace --locked`.
- `build` matrix windows/macos/ubuntu (`fail-fast: false`): plain `cargo build --release --locked`
  (no CLI needed; proves it compiles and links everywhere).
- `installer` (windows, `if: github.event_name == 'workflow_dispatch'`): `cargo install tauri-cli
  --version 2.12.4 --locked` (cached bin by rust-cache), `cargo tauri build --bundles nsis`,
  `actions/upload-artifact@v4` of `target/release/bundle/nsis/*.exe`. Bundler fetches NSIS from
  GitHub at build time (fine in CI). No release, no tag, `permissions: contents: read`.
Existing `ci.yml`, `release.yml` untouched.

`Makefile`: add `rslint` (`cd companion-app && cargo fmt --all --check && cargo clippy
--workspace --all-targets -- -D warnings`) and `rstest` (`cd companion-app && cargo test
--workspace`), add both to `.PHONY`; `check` unchanged pending 0.4. Prove `make lint test golint
gotest` still pass (gofmt walks `companion-app/target`: no `.go` files there, check it is harmless).

## 6. README "Verify on the PC" (owner, non-technical, fill-in table)

Steps with a result column for: download warnings in the browser; SmartScreen ("More info" →
"Run anyway"); Smart App Control state (Settings > Privacy & security > Windows Security > App &
browser control) and whether it blocks the unsigned installer outright; Defender; installer
wording and Start-menu entry; first-start notification once only; native feel (font, dark mode,
no right-click menu, no white flash); tray icon in the overflow (^) and pinning; Task Manager
memory and count of `msedgewebview2.exe` with no window / window open / after closing; open time
first and second; autostart on → sign out/in → tray present; off → entry gone (Task Manager >
Startup apps); second start focuses the window; deep link from a browser address bar and from
Win+R `foreverfriends://connect?code=TEST` shows TEST, and `foreverfriends://connect?code=<script>`
shows nothing; uninstall from Settings > Apps and what remains (`%APPDATA%\<identifier>`,
`%LOCALAPPDATA%\<identifier>`, Run entry, scheme). Ends: "a later session records these in
`docs/RESEARCH.md` as a dated Observed section".

## 7. Gotchas the specialist must handle
- Linux: tray click events are never emitted, `show_menu_on_left_click` unsupported; the menu's
  "Open" is the only way in — fine. Icon may not show without a menu (we always set one).
- macOS: deep links only work from a bundled `.app` (Info.plist); not testable here — README says so.
- Windows: notifications from an unbundled exe show under a generic identity; only the installed
  build shows the app name. Deep link from an unbundled run needs the debug `register_all`.
- WebView2 may keep `msedgewebview2.exe` alive briefly after the window is destroyed — a Verify item.
- `windows_subsystem` hides stdout in release: log to stderr in debug only; no log file (none on §5).

## 8. Verification
In the container: `apt-get install` of the libs above works (Ubuntu 24.04, dry-run succeeded,
not installed). Specialist runs `make rslint rstest`, `cargo build --release`, and the Lua/Go
targets. The Windows build, NSIS installer and everything in section 6 are proved only by CI and
the owner's PC; nothing there is claimed as working. Core tests: version non-empty; link parser
accepts `connect?code=TEST`, rejects other hosts, schemes, extra length, bad chars, missing code.

## 9. Not in scope
Linking, hub address, any network request, game-folder detection or persistence, addon install,
updater, signing, macOS/Linux bundles, edits to `docs/*`, `CLAUDE.md`, `.claude/*`, `companion/`,
`addons/`, `ci.yml`, `release.yml`, hooks. No tag, no release.

## Split
One specialist (`impl-backend`) owns all of it; nothing parallel worth the overhead. The plan
read (`pr-direction-review`, `pr-architecture-review`) runs before code because of item 0.1.
