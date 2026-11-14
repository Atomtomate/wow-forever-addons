# pr-tech-review: companion-app/skeleton (PR #7)

Range `d31312e..f3436a5`, read whole files under `companion-app/`, the workflow and the Makefile, and
checked each Tauri question against the crate sources in `~/.cargo/registry` (tauri 2.12.5,
tauri-build/codegen 2.7.4, tauri-cli 2.12.4, tauri-bundler 2.10.4, the four plugins, muda 0.20,
auto-launch 0.6). Built the app (`cargo build -p forever-friends --locked`, debug) into a scratch
target dir. The GUI was not run.

**Verdict:** one blocking finding: the installer the Verify sheet needs cannot be built before
merge. The Tauri wiring itself holds up. Two non-blocking defects, one minor.

### 1. Blocking: the installer job cannot run before merge

- **`.github/workflows/companion-app.yml:82`** (`if: github.event_name == 'workflow_dispatch'`),
  with `:10`, and `companion-app/README.md:58`: the installer is only built on `workflow_dispatch`,
  and GitHub only dispatches a workflow whose file is on the default branch.
- **Failure:** `companion-app.yml` is not on `main`. Actions shows no "Run workflow" button for it,
  `gh workflow run companion-app.yml --ref companion-app/skeleton` is refused, and the head commit's
  `installer` check run is `skipped`. The owner cannot get the `.exe` for the Verify sheet until he
  has merged the very PR the sheet is meant to check.
- **Confidence:** certain (GitHub docs: "only trigger a workflow run if the workflow file exists on
  the default branch"; check run seen as skipped on f3436a5).
- **Fix:** `pull_request` and `push` runs read the workflow from the commit itself, so let the job
  run on pull requests too: `if: github.event_name != 'push'`. The existing `paths` filter already
  limits it to `companion-app/**` changes, and `pull_request` runs may upload artifacts. If a Windows
  run on every push to the PR is too much, gate it with a label instead: add
  `types: [opened, synchronize, reopened, labeled]` under `pull_request` and
  `if: github.event_name == 'workflow_dispatch' || contains(github.event.pull_request.labels.*.name, 'installer')`.
  Either way, change README:58 to say where the artifact comes from before merge.

### 2. Non-blocking: the cached `cargo-tauri.exe` ignores the version pin

- **`.github/workflows/companion-app.yml:93,97-98,101`**: the cache key is built only from
  `Cargo.lock`, `restore-keys` falls back to any older installer cache, and
  `command -v cargo-tauri ||` skips the install whenever a binary is present.
- **Failure:** change `--version 2.12.4` to `2.13.x` in this file, or let a lockfile bump move
  `tauri` past the CLI. The old cache, or the newest one from the prefix, restores 2.12.4,
  `command -v` succeeds, and the bundle is built with a CLI other than the one the file names.
  Nothing in the log says so.
- **Confidence:** certain.
- **Fix:** put the CLI version in the key (`cargo-installer-${{ runner.os }}-cli2.12.4-${{ hashFiles(...) }}`,
  and the same in `restore-keys`), and test the version rather than the presence:
  `cargo tauri --version 2>/dev/null | grep -q ' 2.12.4$' || cargo install tauri-cli --version 2.12.4 --locked --force`.

### 3. Non-blocking: a double-click on "Choose game folder…" opens two dialogs

- **`companion-app/app/ui/main.js:17-19`** with `companion-app/app/src/commands.rs:397`: each
  `click` sends its own `pick_game_folder`, and nothing stops a second call while one is open.
- **Failure:** a double-click fires `click` twice before the first dialog appears, because the
  plugin first hops to the main thread with `run_on_main_thread`
  (`tauri-plugin-dialog/src/desktop.rs:172-181`). That gives two `blocking_pick_folder` calls on
  two worker threads, and rfd shows two folder dialogs, both on the same parent (side by side on
  Windows, one after the other on GTK). Whichever closes last sets the folder.
- **Confidence:** likely (from the code path; not run).
- **Fix:** in `main.js`, set `el("pick").disabled = true` before the invoke and back to `false` in
  a `.finally(...)`. WebView2, WKWebView and WebKitGTK all support `Promise.prototype.finally`.

### 4. Minor: on Windows and macOS a failed first-start notice is never logged

- **`companion-app/app/src/first_start.rs:2,23-26`**: the module says failures are logged, but on
  Windows and macOS `show()` hands the toast to `async_runtime::spawn` and throws its result away
  (`tauri-plugin-notification/src/desktop.rs`, `show` under `cfg(any(windows, target_os = "macos"))`).
- **Failure:** if the toast is refused (notifications off, or no AUMID shortcut), the debug build
  logs nothing and the marker is still written. That outcome is intended; the comment and the log
  line just don't hold on those systems.
- **Confidence:** certain.
- **Fix:** add one line to the comment ("on Windows and macOS the plugin reports no failure"). Step 6
  of the Verify sheet already covers the real behaviour.

## Questions in the brief that check out (no finding)

- **Plain `cargo build --release` embeds the page and does not need `devUrl`.** With no
  `custom-protocol`, `cfg(dev)` is on, but codegen embeds `frontendDist` whenever `devUrl` is unset
  (`tauri-codegen/src/context.rs:178-208`). `get_app_url` falls back to `tauri://localhost` /
  `http://tauri.localhost` (`tauri/src/manager/mod.rs:345-358`). The built binary contains the
  asset paths. In dev mode the CSP falls back to `csp`, because `devCsp` is unset.
- **`cargo tauri build` sets `tauri/custom-protocol` and `--no-default-features`**
  (`tauri-cli/src/interface/rust.rs:481`, `:429-445`). The CLI's manifest rewrite leaves
  `tray-icon` alone, since it is excluded from the CLI-managed features.
- **Window lifecycle.** `run_on_main_thread`, then `create_window` on the main thread, is handled
  inline (`tauri-runtime-wry/src/lib.rs:263-277`), so it does not deadlock, and a second
  `open_main` finds the window. `ExitRequested { code: None }` fires only when the last window is
  destroyed (`:4256-4268`). `app.exit(0)` arrives as `Some(0)`, so Quit still quits.
- **Tray.** The `TrayIcon` is registered in the manager, so dropping the handle keeps it. muda
  flips a `CheckMenuItem` before it dispatches the event on Windows, macOS and GTK, so
  `toggle_autostart`'s "already flipped" holds. On GTK the revert `set_checked` is not
  re-dispatched (`is_syncing`). `is_enabled()` on Windows also reads Task Manager's
  StartupApproved override, so the box starts unchecked if the member disabled it there.
- **Uninstall and autostart.** The NSIS uninstaller deletes HKCU Run `${PRODUCTNAME}`
  (`installer.nsi:910`). That only works because `.app_name("Forever Friends")` equals
  `productName`, so keep the two equal.
- **Single instance.** On Windows the plugin takes a mutex named after the identifier and exits the
  second process during plugin init, before `setup`. The callback goes through `open_main`, so any
  thread is fine.
- **Notifications.** Installed: `app_id = identifier`, and NSIS stamps the same AUMID on the Start
  menu shortcut (`installer.nsi:994-997`). Run from `target\release`: no app_id, so the toast shows
  as PowerShell. The Verify sheet runs the installed copy, so this is expected, not a defect.
- **Capabilities and CSP.** `withGlobalTauri` arrives as an initialisation script, which the CSP
  does not filter. `connect-src ipc: http://ipc.localhost` covers both IPC transports.
  `emit_to("main", …)` reaches a JS `listen` whose target is `Any` (`event/listener.rs:346-351`).
  `tauri-build` validated the four permission identifiers at build time.
- **Dialog.** `blocking_pick_folder` waits on a tokio worker, not the main thread. The plugin runs
  the dialog itself through `run_on_main_thread`.
- **CI and Makefile.** The apt package names match Ubuntu 24.04, and the Linux and macOS builds
  are green. Running "Core crate alone" before the GUI libraries go in is sound. `make rslint
  rstest` are correct as written.

**Follow-ups:** none beyond the fixes above.

**What carried it:** reading `tauri-codegen/src/context.rs` and `tauri-cli/src/interface/rust.rs`
side by side answered the `custom-protocol` question from source instead of from memory.
