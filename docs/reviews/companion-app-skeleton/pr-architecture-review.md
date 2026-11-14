# pr-architecture-review: `companion-app/` skeleton (PR #7)

Range: `d31312e..f3436a5`, excluding `companion-app/Cargo.lock` and `companion-app/app/icons`.
Standard held to: the Go standard in `.claude/agents.local.md` read for Rust (thin shell; logic
in a package that reaches the network, clock or file system only through what the caller
passes in, so it is tested without a PC; OS code behind one small function each). No layering
script exists; crate edges and `use` lists were read directly, and each symbol the app crate
uses was followed to its definition (`state::snapshot` -> `forever_friends_core::version`,
`platform::*`, `window::open_main`).

**Verdict:** Well placed. The one crate edge is `app -> core`, core has no dependencies, every
OS difference sits in `platform.rs`, `setup` and `tray.rs` carry no `cfg`, the page's state has
one owner, and the window has one way in. All four plan-read findings are applied (below). One
finding, non-blocking for this PR but to be settled before piece 2: core's written boundary is
narrower than the standard, and as written it sends the Go logic that touches I/O into the
crate that cannot be tested without GUI libraries.

## Plan-read findings, checked

1. Link rule's home: moot. The deep link and URL scheme are gone; core holds only `version()`
   (`companion-app/core/src/lib.rs:8`), and no `on_open_url`, `tauri::Url` or scheme code
   remains in the app crate.
2. OS branches: applied. `companion-app/app/src/platform.rs:7-24` holds
   `apply_activation_policy` and `autostart_label`; callers `companion-app/app/src/main.rs:46`
   and `companion-app/app/src/tray.rs:22` have no `cfg`. The only remaining `cfg` outside
   `platform.rs` is the crate-root `windows_subsystem` attribute (`main.rs:6`), which Rust
   allows only there, and the `cfg!(debug_assertions)` in `debug_log` (`main.rs:19`), which is
   build mode, not OS.
3. UI state owner: applied. `companion-app/app/src/state.rs` owns the struct, the setter and
   the single `snapshot`; the command (`commands.rs:11`) and the event (`state.rs:48`) both
   come from it, and the folder picker writes only through `state::set_folder`
   (`commands.rs:21`). It stays in the app crate, so core keeps no `serde`.
4. Core reachable alone: applied. `.github/workflows/companion-app.yml:40-42` runs
   `cargo test -p forever-friends-core --locked` before the GUI packages are installed, and
   `Makefile` `rstest` runs it first too. A GUI dependency in core now fails CI.

Also confirmed: `window::open_main` (`companion-app/app/src/window.rs:10`) is the only path that
shows the window — single-instance callback (`main.rs:29`), tray menu (`tray.rs:37`), tray click
(`tray.rs:49`).

### 1. `companion-app/core/src/lib.rs:3-5`, `companion-app/core/Cargo.toml:3`: core's boundary is "no dependencies, no I/O", which leaves the I/O-touching Go logic no testable home
- **file:line:** `core/src/lib.rs:3-5` ("No dependencies, no I/O, no GUI ... Logic that does
  not need the GUI goes here") and `core/Cargo.toml:3` ("std only, no GUI and no I/O") fix a
  boundary stricter than, and different from, the standard the crate exists to serve.
- **The crossing:** the Go packages pieces 2-3 port are not I/O-free. `companion/internal/`
  `config`, `atomicfile`, `install` and `dataaddon` write files under a root passed in;
  `hubapi` and `linkflow` speak HTTP through a client passed in; `syncer` and `internal/app`
  orchestrate all of them through `app.Env`, a set of function values that tests replace;
  `wire`, `savedvars`, `hubapi` and `config` need JSON, SHA-256 and Ed25519. Read literally,
  core may take none of these, so each lands in `forever-friends` (the Tauri crate), whose tests
  need webkit2gtk and appindicator — the split the plan read guarded against, at the scale of
  the whole companion instead of one link parser. The CI step at
  `companion-app.yml:40-42` enforces only "no GUI libraries", so the code and its documented
  boundary already disagree on what core may hold.
- **The rule:** "logic lives in packages ... that do not reach for the network, the clock or the
  file system except through what the caller passes in, so they are tested without a PC. For
  writing files, a root directory passed in counts as that" (`.claude/agents.local.md`, Go). The
  line is *how* I/O is reached, not *whether*; `internal/app`'s `Env` is the pattern.
- **The fix:** before piece 2, restate core's boundary as the standard's: no Tauri or GUI
  dependency; network, clock and file system only through what the caller passes in (a root
  path, a trait or closure the app crate implements, as Go's `Env`); its own dependencies as D9
  approves them. The flow that is `companion/internal/app` today then ports into core behind an
  `Env`-like trait, and the app crate stays the `companion/main.go` equivalent. Two doc lines
  now; the trait arrives with piece 2 (adds code then, not now). Whether core may take crates
  such as `serde_json` or `ed25519-dalek` is an owner question for D9, not this axis.

## Layout for pieces 2-3

With finding 1 settled, the layout ports without restructuring: Go packages become modules of
`forever-friends-core` (`wire`, `savedvars`, `gamefind`, `install`, `syncer` ... with the same
acyclic edges), `platform.rs` takes the GUI-free OS functions of `companion/internal/platform`
and `gamefind`'s registry lookups behind one function each, and `state.rs` takes the new fields
(connection status, detected folder). Nothing in the skeleton has to move. If core grows large
or its dependencies split cleanly (pure parsing vs. HTTP), a second GUI-free crate is an
additive change to `Cargo.toml` `members`, not a restructure.

**Follow-ups**
- `companion-app/app/src/tray.rs:16`: the tray's "Not connected yet" item is a second display
  of connection state that does not go through `state.rs`. Static today, so no finding; when
  piece 2 makes it real, `state.rs` should own it (its `publish` updates the tray item as well
  as emitting), not a separate writer in `tray.rs`.
- `.claude/agents.local.md` still has no Rust standard; this review read the Go one across.
  Writing it (with finding 1's wording for core) is the parallel "Companion app 0" PR's, and
  `pr-direction-review`'s to track.

**What carried it:** listing the imports of each `companion/internal/` package against
`core/src/lib.rs:3`'s "no I/O": the skeleton is fine on its own, and only the next pieces' Go
sources show where its stated boundary would put them.
