# Forever Friends companion app (skeleton)

A tray-first desktop app in Rust and Tauri 2. **This is a skeleton**: it shows a tray icon, a
small window, and does nothing for the game yet. It makes no network request and contains no hub
address. The shipped companion is still the Go program in `companion/`; this app replaces it
only after it reaches parity.

## What it does today

- Lives in the tray (notification area). Closing its window does not quit it; **Quit** in the
  tray menu does.
- Tray menu: Open Forever Friends, "Not connected yet" (placeholder), a **start with the
  computer** toggle (off until you switch it on; switching it off removes the entry), Quit.
  Left click on the icon opens the window (not on Linux, which sends no click events: use the
  menu).
- The window shows the version and lets you pick a game folder. The choice is kept in memory
  only; nothing reads it yet.
- A second start does not open a second copy; it brings the first one's window forward.
- On the very first start it shows one notification, "Forever Friends is in your tray", and
  writes an empty file `first-start-done` in its own data folder so it never says it again.
- macOS: tray only, no dock icon.

Not here yet: linking to the hub, finding the game, installing addons, updates, code signing,
macOS and Linux installers. Windows first.

## Layout

```
companion-app/
  core/   forever-friends-core: std only, no GUI, no I/O (version; later, logic)
  app/    forever-friends: the Tauri shell
    src/       main, tray, window, state, commands, first_start, platform
    ui/        the window's page: plain HTML, CSS and JS, no npm, no framework
    capabilities/default.json   what the page may call (two commands and event listening)
    tauri.conf.json
    icons/     see ASSETS.md
```

`platform.rs` holds every operating-system difference, one small function each. `state.rs` owns
what the page shows. The core crate must never depend on Tauri.

Crates (nothing else without asking): `tauri`, `tauri-build`, the first-party plugins
`single-instance`, `autostart`, `notification`, `dialog`, and `serde` (derive) for the page's
state. App identifier: `io.github.atomtomate.foreverfriends` (names the app data folder).

## Build and test

Rust 1.90 or newer. On Linux, first install the GUI libraries:

```
sudo apt-get install libwebkit2gtk-4.1-dev libayatana-appindicator3-dev librsvg2-dev libxdo-dev libssl-dev build-essential file
```

From the repository root: `make rslint rstest` (format, clippy, tests; not part of `make check`).
The core crate alone needs no libraries: `cargo test -p forever-friends-core` in `companion-app/`.
Run it: `cargo run -p forever-friends`. Plain `cargo build` needs no Tauri CLI.

The Windows installer (unsigned, per-user NSIS) is built by hand: Actions, "Companion app", Run
workflow, then download `forever-friends-installer` from the run. Locally:
`cargo install tauri-cli --version 2.12.4 --locked`, then `cargo tauri build --bundles nsis` in
`companion-app/app`. The installer fetches Microsoft's WebView2 runtime only if the PC lacks it
(Windows 11 and current Windows 10 have it); the app itself never uses the network.

## Verify on the PC

For the owner, on Windows. Nothing below is claimed to work until a result is written in. The
installer is **unsigned**, so the warnings in steps 1 to 4 are the baseline to record; signing
is a later piece. A later session copies the results into `docs/RESEARCH.md` as a dated
"Observed" section.

| # | Do this | Write down what happened |
|---|---------|--------------------------|
| 1 | Download the installer in your browser. | Any warning from the browser (what it says): |
| 2 | Run it. | SmartScreen: does it appear? Do you need "More info", then "Run anyway"? |
| 3 | Settings, Privacy & security, Windows Security, App & browser control, Smart App Control. | State (on / evaluation / off). Does it block the installer outright? |
| 4 | Watch Microsoft Defender during the install. | Any alert or scan prompt: |
| 5 | Read the installer's wording. Afterwards look in the Start menu. | Language and wording; is there a "Forever Friends" entry? |
| 6 | First start: do you get one notification, "Forever Friends is in your tray"? Quit, start again. | Shown the first time? Not shown the second time? |
| 7 | Open the window. Look at font, dark mode (switch Windows to dark), right-click inside it. | Looks native? Follows dark mode? Right-click menu absent? White flash on open? |
| 8 | Find the tray icon (it may be under the ^ arrow); try pinning it next to the clock. | Where it showed; pinning worked? |
| 9 | Task Manager, Details: memory of `forever-friends.exe`, count of `msedgewebview2.exe`: with no window open, with the window open, after closing it. | Numbers (3 rows): |
| 10 | Time from clicking the tray icon to a visible window: first time, second time. | Seconds (2 values): |
| 11 | Tray menu: switch "Start with Windows" on, sign out and in. | Tray icon back without starting it? |
| 12 | Switch it off. Task Manager, Startup apps. | Entry gone? |
| 13 | Start the app a second time (Start menu) while it runs. | Window of the first copy comes forward, no second tray icon? |
| 14 | In the window, choose a game folder. Quit and start again. | Folder shown after choosing? Gone after restart (expected)? |
| 15 | Uninstall from Settings, Apps. Look at `%APPDATA%\io.github.atomtomate.foreverfriends`, `%LOCALAPPDATA%\io.github.atomtomate.foreverfriends`, and Task Manager, Startup apps. | What is left behind: |
