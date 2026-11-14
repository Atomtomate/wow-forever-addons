# pr-architecture-review: plan read, `companion-app/` skeleton (Rust + Tauri 2)

Plan: `scratchpad/plan.md` (base `main` at `d31312e`). Standard held to: the Go standard in
`.claude/agents.local.md` read for Rust (thin shell; logic in a core crate that touches the
network, clock or file system only through what is passed in; OS-specific code behind one small
function each), since D9 and the Rust standard land in the parallel "Companion app 0" PR. No
layering script exists; the plan's crate and module edges were read directly.

**Verdict:** The shape is right: a two-crate workspace with `app -> core` as the only edge, core
with no dependencies, its own directory beside the Go `companion/` so the two toolchains never
share a root manifest, a capability file that lets the page do two things and nothing else, and
plugins driven from Rust only. One placement must be settled before code (finding 1); the other
three are non-blocking but set patterns the next Rust feature will copy.

### 1. Plan §0.5, §2 ("URL parsing uses `tauri::Url`"), §4 "Links": the link rule has no fixed home (blocking)
- **Plan line:** §0.5 leaves `parse_connect_link` in core as an owner-vetoable option ("then it
  lives in the app crate"), while §2 says parsing uses `tauri::Url`. Read together, the
  specialist can build half the rule in each crate: `Url` methods in `links.rs` checking scheme
  and host, core checking the code.
- **The crossing:** `on_open_url` and `get_current()` hand over `Vec<tauri::Url>`. If `links.rs`
  takes it apart with `Url::scheme()` / `host_str()` / `query_pairs()` before calling core, the
  acceptance rule (scheme `foreverfriends`, host `connect`, one `code`, 1-32 of
  `[A-Za-z0-9-]`) is split across a crate that can only be tested with webkit2gtk installed and
  one that can be tested with nothing.
- **The rule:** "logic lives in packages ... so they are tested without a PC"
  (`agents.local.md`, Go standard). Parsing untrusted input from the browser is exactly the logic
  that rule exists for, and the plan's own §8 tests (other hosts, schemes, lengths, bad chars,
  missing code) only cover the whole rule if the whole rule is in core.
- **The fix:** state it as decided, not optional: core exposes `parse_connect_link(&str)` that
  owns the whole rule in std only, and `links.rs` does nothing but `parse_connect_link(url.as_str())`
  and act on the result; drop "URL parsing uses `tauri::Url`" (adds a small hand-written parser
  to core, removes nothing from the dependency list). This is not an owner question: it is what
  the standard already says.

### 2. Plan §4 (setup, tray labels) and §7: OS-specific branches are scattered through `setup` and `tray.rs`
- **Plan line:** §4 puts the macOS `ActivationPolicy::Accessory` call in `setup`, the debug-only
  Windows/Linux `register_all()` in `setup`, and the per-OS autostart label ("Start with
  Windows" / "Open at login" / "Start at login") inside the tray menu builder.
- **The crossing:** three `#[cfg(target_os = ...)]` blocks inside the two functions every later
  feature edits (`setup` and the tray builder), each reaching for a platform-only API inline.
- **The rule:** "Code for one operating system sits in `_windows.go` / `_other.go` files behind
  one small function each" (`agents.local.md`). The Rust reading is one `platform.rs` (or
  `platform/{windows,macos,linux}.rs`) with `fn autostart_label() -> &'static str`,
  `fn apply_activation_policy(&mut App)` and `fn register_schemes_for_dev(&AppHandle)`, called
  unconditionally from `setup` and `tray.rs`.
- **The fix:** add the module and the three functions; `setup` and `tray.rs` carry no `cfg`
  (adds one file of ~30 lines). Worth fixing now because this is the first Rust code here and
  later OS-specific needs (game-folder defaults, the Go companion's `_windows.go` equivalents)
  will copy wherever these land.

### 3. Plan §4 "Links" / "Folder" and §1 module split: the in-memory UI state has no owning module
- **Plan line:** the `Mutex` holding the code and the chosen path is written from the link
  handler, from the `pick_game_folder` command, and read by `get_ui_state` and the `ui-state`
  emit; §1's split (`tray.rs, window.rs, links.rs, first_start.rs`) names no home for it.
- **The crossing:** without an owner, `links.rs` and the command handler each lock the mutex
  and build the emitted payload themselves, so the shape the page reads (`get_ui_state` result
  and `ui-state` event, which must agree) is produced in two places.
- **The rule:** on its own terms: a fact the page depends on should be obtained from the one
  thing that owns it, not from whichever handler last held the value; the next state field
  (connection status, detected folder) otherwise goes into every writer.
- **The fix:** an app-crate `state.rs` owning the struct, its setters and the single
  "snapshot + emit" function, which links, the command and `get_ui_state` call (adds one small
  module). Keep it in the app crate: it needs `serde` and `Emitter`, and core stays dependency-free.

### 4. Plan §5 Makefile and §0.4: the no-dependency core is not reachable without the GUI libraries
- **Plan line:** `rstest` is `cargo test --workspace`, and §0.4 keeps both Rust targets out of
  `check` because they need webkit2gtk/appindicator dev packages.
- **The crossing:** `--workspace` compiles the app crate (and with it `tauri`, `wry`, `gtk`) to
  run the core tests, so the one property the split buys, core testable on any machine with
  only a Rust toolchain, is not used by any target or CI step.
- **The rule:** the reason for a core crate with no I/O and no Tauri dependency is that its
  tests need nothing (`agents.local.md`, "tested without a PC"); a target that cannot run them
  alone hides whether that still holds, and a later `tauri` import into core would go unnoticed.
- **The fix:** add a core-only step, `cargo test -p forever-friends-core --locked`, in CI's
  `check` job before the system-package install (or as a separate dependency-free job), which
  fails the moment core grows a GUI dependency; the Makefile may mirror it (adds one step;
  §0.4 is unaffected either way).

## Tauri 2 points checked, no finding on this axis
- Plugin order: `single_instance` first with the `deep-link` feature, then `deep_link`, is the
  documented requirement; second-start argv reaches `on_open_url` in the first instance.
- Window lifecycle: `create: false` template + `WebviewWindowBuilder::from_config`, close
  destroys, `RunEvent::ExitRequested { code: None }` -> `prevent_exit`, Quit via `exit(0)` gives
  `Some(0)` and is not prevented. One `open_main_window(&AppHandle)` should serve the tray,
  the single-instance callback and the link handler; the plan implies it, the specialist should
  make it the only path that shows the window.
- Capabilities: app-command permissions generated from `AppManifest` in `build.rs`, referenced
  unprefixed (`allow-pick-game-folder`), no `core:default`, no plugin permissions for the page,
  window shown from Rust so no window permission: minimal. CSP with `ipc:` /
  `http://ipc.localhost` and no inline script fits `withGlobalTauri` without npm.
- Tray: Linux emits no click events and ignores `show_menu_on_left_click`; the menu's "Open"
  as the only way in there is correct, and the plan says so (§7).
- CI: path filters, `Swatinem/rust-cache` with `companion-app -> target`, the apt list, and a
  `workflow_dispatch`-only NSIS job with `contents: read` keep the Rust toolchain out of
  `ci.yml`. Correctness of the plain `cargo build` (without `custom-protocol`) and whether
  `cargo tauri build` adds that feature itself are `pr-tech-review`'s to confirm.
- `make golint`: `gofmt -l .` walks `companion-app/target` (no `.go` files, harmless but slow);
  not a structural point.

**Follow-ups**
- `foreverfriends://connect?code=` is produced by the hub and consumed by the app, so it is a
  cross-party format whose home is `docs/CONTRACT.md`; the plan says "no change to
  `docs/CONTRACT.md` formats". That rests on the contract rule, so it is `pr-direction-review`'s
  to raise; named here because finding 1 places the companion's side of it.
- `.claude/agents.local.md` needs the Rust standard written (core crate, no Tauri dependency,
  no I/O except what is passed in, OS code behind `platform` functions) so findings 1-2 rest on
  a written line next time; the parallel "Companion app 0" PR is where it lands.

**What carried it:** the Go standard's "tested without a PC" sentence in `agents.local.md`,
read against §2's "URL parsing uses `tauri::Url`": the two plan lines together showed the link
rule could end up split across the testable and the untestable crate.
