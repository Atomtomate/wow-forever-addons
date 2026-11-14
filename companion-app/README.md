# Forever Friends companion app (skeleton)

A tray-first desktop app in Rust and Tauri 2. **This is a skeleton**: it shows a tray icon and a
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
  only; nothing reads it yet. Closing the window destroys it (less memory while idle, a slightly
  slower reopen: rows 9 and 10 below measure both).
- A second start does not open a second copy; it brings the first one's window forward.
- On the very first start it shows one notification, "Forever Friends is in your tray", and
  writes an empty file `first-start-done` in its own data folder so it never says it again.
- macOS: tray only, no dock icon.

Not here yet: linking to the hub, finding the game, installing addons, updates, code signing,
the `foreverfriends://` link, macOS and Linux installers. Windows first.

## Layout

`core/` is `forever-friends-core`: logic with no Tauri or GUI dependency, tested alone. `app/` is
`forever-friends`, the thin Tauri shell (`src/`, the plain HTML page in `ui/`, `capabilities/`,
`tauri.conf.json`, `icons/`: see `ASSETS.md`). Every operating-system difference is a function in
`app/src/platform.rs`. Dependencies follow the rule in the root `CLAUDE.md`; the app uses `tauri`,
four first-party plugins and `serde`.

## Build and test

Rust 1.90 or newer. On Linux, first install the packages listed as `LINUX_GUI_PACKAGES` in
`.github/workflows/companion-app.yml`.

From the repository root: `make rslint rstest` (format, clippy, tests; not part of `make check`).
The core crate alone needs no libraries: `cargo test -p forever-friends-core` in `companion-app/`.
Run it: `cargo run -p forever-friends`. Plain `cargo build` needs no Tauri CLI.

### Getting the Windows installer

The installer is unsigned and per-user. The installer fetches Microsoft's WebView2 runtime only
if the PC lacks it (Windows 11 and current Windows 10 have it); the app itself never uses the
network.

- **Before the pull request is merged:** open the pull request on GitHub, then **Checks**, then
  the run named "Companion app", and scroll to **Artifacts** on its Summary page. Download
  `forever-friends-windows-installer`. You must be signed in to GitHub. It arrives as a `.zip`
  holding the `.exe`.
- **After the merge:** Actions, "Companion app", **Run workflow**, then the same Artifacts
  section of that run.
- Locally: `cargo install tauri-cli --version 2.12.4 --locked`, then `cargo tauri build --bundles
  nsis` in `companion-app/app`.

## Verify on the PC

For the owner, on Windows, once. Nothing below is claimed to work until a result is written in.
Plain words are fine in the right-hand column: a word, a number, or the exact text on screen
(a phone photo of a warning is good). The installer is **unsigned**, so the warnings are the
baseline to record; signing is a later piece. A later session copies the results into
`docs/RESEARCH.md` as a dated "Observed" section. The `foreverfriends://` link test is not part
of this piece; it moved to a later one.

**Where to run it:** a normal PC you use, not a virtual machine. To open a folder path below,
paste it into the address bar of File Explorer and press Enter.

### Before you start

| # | Do this | Write down |
|---|---------|------------|
| A | Today's date. Settings, System, About: Windows edition and version. | Date, edition, version/build: |
| B | Open Edge and Chrome, then each one's menu, Help (or Settings), About. | Edge version, Chrome version: |
| C | Settings, System, Display: the scale (100 %, 125 %, ...). | Scale: |
| D | Settings, Apps, Installed apps: is there an entry "Microsoft Edge WebView2 Runtime"? (If not, the installer will download it from Microsoft, the one time it uses the network.) | Present before installing? Yes / no: |
| 0 | Settings, Privacy & security, Windows Security, App & browser control, Smart App Control. **Do not change it for this test.** | State (On / Evaluation / Off): |

If Smart App Control is **On**, it will probably stop the unsigned installer outright: write down
the message word for word, and stop. The remaining steps need a PC where it is Off or
Evaluation.

### The installer

| # | Do this | Write down |
|---|---------|------------|
| 1 | Download the installer (see above) in **Edge**. It is a `.zip` from a signed-in GitHub page, not what friends will get later, so browsers may warn differently. Does the browser warn? Then right-click the zip, **Extract all** (Windows' own; other tools may drop the "from the internet" mark). Repeat the download in **Chrome**. | Edge said / Chrome said (exact words and buttons). Did Defender say anything? |
| 2 | Run the installer. A blue SmartScreen window may appear. | Its title, whether it says "Unknown publisher" or a name, which buttons it offers (for example "More info", then "Run anyway"): |
| 3 | Watch the installer itself. Afterwards look in the Start menu. | Language and wording; a "Forever Friends" entry? Did Defender say anything? |
| 4 | Later, run the workflow again to get a second build that differs slightly, download it and run it on top. | Do the SmartScreen or browser warnings come back, or look different? |

### Using the app

| # | Do this | Write down |
|---|---------|------------|
| 5 | First start: do you get one notification, "Forever Friends is in your tray"? Then Quit (tray menu) and start again. | Shown the first time? Sender name and icon on it? Not shown the second time? Did Defender say anything at first start? |
| 6 | Find the tray icon (it may be under the ^ arrow); try dragging it next to the clock. Right-click it. | Shown or hidden under ^? Does the menu look like other tray menus (font, dark mode)? |
| 7 | Left-click the icon to open the window. Put it beside Windows Settings. Do this in light mode, then dark mode (Settings, Personalization, Colors). | Do font, size and colours look like they belong next to Settings? Light: / Dark: |
| 8 | Click "Choose game folder...". | Is it the normal Windows folder window? Is the choice shown afterwards? |
| 9 | Try web-page habits in the window: drag to select text; press F5 or Ctrl+R; Ctrl and +; press Tab and look at the focus outline; right-click. | Which of these behave like a web page (selection of labels, reload, zoom, outline, right-click menu): |
| 10 | Set the display scale to 125 %, then 150 % (Settings, System, Display) and look at the window and tray icon. | Sharp or blurry at each? (Set it back afterwards.) |
| 11 | Task Manager, **Processes** tab, expand the "Forever Friends" group (it lists only its own processes). Memory with no window open; with the window open; 30 seconds after closing the window. | Three numbers. After 30 seconds, are the "WebView2" processes gone? Yes / no: |
| 12 | Time from clicking the tray icon to a visible window: the first time, then the second time. Was there a white flash? | Seconds (two values), flash yes / no: |
| 13 | Tray menu: switch "Start with Windows" on (Defender may speak up). Sign out and in. | Did Defender say anything? Is the tray icon back without you starting it? |
| 14 | Start the app a second time from the Start menu while it runs. | Does the first window come forward, with no second tray icon? |

### Uninstalling

| # | Do this | Write down |
|---|---------|------------|
| 15 | Make sure "Start with Windows" is **on** again. Quit nothing: leave the app running. Uninstall from Settings, Apps. | Did the uninstaller close the running app, or ask you to? |
| 16 | Look at what is left. Task Manager, Startup apps: is a "Forever Friends" entry still listed? Then open each of these in File Explorer: `%LOCALAPPDATA%\Forever Friends` (the install folder), `%LOCALAPPDATA%\io.github.atomtomate.foreverfriends` (the web view's data), `%APPDATA%\io.github.atomtomate.foreverfriends` (holds `first-start-done`). Also the Start menu. | For each: still there? (Note: the uninstaller removes the two data folders only if you tick its "delete application data" box; say whether you ticked it.) |
