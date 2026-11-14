# pr-parsimony-review: PR #7, `companion-app/skeleton` (f3436a5 against d31312e)

**Verdict:** About right for a skeleton, with one real piece of surplus: the `ui-state` event
channel, which has no writer except the page's own command. The deep-link removal left nothing
behind (no scheme, plugin, config or `argv` handling; grep is clean). The rest is small trims.
Read first: `CLAUDE.md`, `.claude/agents.local.md`, the plan in this directory. Files and
reviews under `docs/reviews/` are not counted.

### 1. The `ui-state` event, `publish` and its permissions have no second writer
- **What** `companion-app/app/src/state.rs:535-581` (`UI_STATE_EVENT`, `publish`, `Emitter`),
  `companion-app/app/ui/main.js:818`, `companion-app/app/capabilities/default.json:6-7`
  (`core:event:allow-listen`, `allow-unlisten`). The only caller of `set_folder` is
  `pick_game_folder` (`commands.rs:383-388`), which the page itself invoked and which already
  returns the folder; `main.js:813-816` throws that return value away ("arrives through the
  event"). `state.rs:526-529` argues the event and the command "cannot disagree", but with one
  writer they cannot disagree anyway. The plan's tray status line is static text, not state.
- **Why it recurs** Only when something other than the page writes state (tray, a scanner).
  That is a later piece and can add the event with its first real writer.
- **The smaller version** `pick_game_folder` returns `Option<Snapshot>` (store the folder, return
  `state::snapshot`); the page calls `render(result)` in its `.then`. Delete `publish`, the
  constant, the `listen` line and the two `core:event` permissions. `set_folder` stays a plain
  setter. Capabilities shrink to the two app commands, which is also closer to "minimal".
- **Net effect** About 12 lines of Rust, 1 JS line, 2 permissions; one concept fewer (an event
  protocol next to a command protocol), and a smaller capability file to audit.

### 2. `Inner` struct, poison recovery and `lock()` guard a single `Option<String>`
- **What** `state.rs:538-552`. `Inner` wraps one field; `lock()` recovers from poisoning. The
  release profile sets `panic = "abort"` (`companion-app/Cargo.toml:180`), so a lock can never be
  poisoned in the shipped binary; the branch is reachable only in debug and tests, where no test
  panics while holding it.
- **Why it recurs** The comment `state.rs:528` says "the next state field goes here too": that
  is the speculative part. A second field is the moment to introduce `Inner`.
- **The smaller version** `pub struct UiState(Mutex<Option<String>>)`, and
  `.lock().unwrap()` (or `.expect("state lock")`) at the two sites. Snapshot takes the clone.
- **Net effect** About 10 lines and two concepts (inner struct, poison policy). Together with
  finding 1, `state.rs` falls from 56 lines to roughly 30.

### 3. The release profile is size tuning nobody asked for
- **What** `companion-app/Cargo.toml:176-181`: `lto`, `codegen-units = 1`, `opt-level = "s"`,
  `panic = "abort"`, `strip`. The brief lists none of it. It also makes every CI `build` leg
  (`companion-app.yml:82`, three OSes, `cargo build --release`) and the installer slower, and
  `panic = "abort"` changes failure behaviour (a panic in a Tauri callback kills the tray app
  instead of being contained), which a skeleton has not earned.
- **Why it recurs** No; it is one block. But it is a decision (size over build time and panic
  semantics) made without a DECISIONS entry.
- **The smaller version** Delete the `[profile.release]` table; add it back when the installer
  size is measured (Verify row 9 is where the numbers would come from).
- **Net effect** 6 lines, faster CI, no hidden behaviour change.

### 4. Platform label and macOS policy for systems with no deliverable
- **What** `companion-app/app/src/platform.rs:511-520` (`autostart_label`, three-way) and
  `:502-509` (macOS `Accessory`, plus the `#[cfg]` twin). The brief names "Start with Windows";
  the README says "Windows first" and lists macOS and Linux installers as not here. CI builds
  the other two systems only to prove they compile.
- **Why it recurs** Each added OS adds a branch to every such function; `platform.rs` is
  designed to be that place (argued in its header and `main.rs`), so the structure is fine.
  Only the unneeded content is the weight.
- **The smaller version** `tray.rs` uses the literal `"Start with Windows"`; delete
  `autostart_label`. Keep `apply_activation_policy` only if the owner wants macOS to behave
  from day one (plan item 6 recommended it); otherwise drop it and the `App` import, which
  leaves `platform.rs` empty and removable. The README bullet "macOS: tray only" goes with it.
- **Net effect** 10 to 24 lines and one file; the next OS need recreates `platform.rs` when
  there is a second function to put in it. Owner's call on the macOS half.

### 5. Duplicated setup in the workflow
- **What** `.github/workflows/companion-app.yml`: three copies of "Rust toolchain" +
  `actions/cache` (lines 33-44, 64-75, 91-103), and an Ubuntu compile twice, once in `check` and
  once as a `build` matrix leg. The runner images ship stable Rust with `clippy` and `rustfmt`;
  `rustup toolchain install stable --profile minimal` is a reinstall of what is there (it is
  the pinning-free path, so it buys nothing over the default), at least on the two build jobs.
- **Why it recurs** A fourth job (a release job later) copies the block again.
- **The smaller version** Drop the toolchain step from `build` and `installer` (verify once on
  the three images). Fold `check` into the matrix: `fmt`, `clippy` and `test` guarded by
  `if: runner.os == 'Linux'`, so Linux compiles once. Keep the "Core crate alone" step ahead of
  the GUI-library install; that is argued in the plan and is real. Keep the three-OS build.
- **Net effect** About 15 lines and one fewer Linux job; the cache key and toolchain live in
  one place. Lower confidence on the toolchain half (needs one CI run to confirm).

### 6. Committed generated permission files
- **What** `companion-app/app/permissions/autogenerated/get_ui_state.toml`, `pick_game_folder.toml`
  (22 lines). Their header says "Automatically generated - DO NOT EDIT"; `tauri-build` rewrites
  them from `build.rs:310` on every build. The neighbouring `/app/gen/schemas` is already ignored
  as generated output.
- **Why it recurs** Every new command adds another generated file and a diff line in review.
- **The smaller version** Add `/app/permissions/autogenerated` to `companion-app/.gitignore`
  and `git rm` the two files. The command list in `build.rs` stays the single source.
- **Net effect** 22 generated lines out of review, none in future command PRs.

### 7. Smaller items (each about one line)
- `.gitignore:5` (`companion-app/target/`) repeats `companion-app/.gitignore:1` (`/target`):
  delete the root line.
- `core/src/lib.rs` is `version()` plus its one test: the leftover of the removed deep-link
  parser, kept because the brief scopes a core crate. Fine as a seam; do not add to it, and note
  in the PR body that the crate is intentionally near-empty. `rstest` (`Makefile:138`) and
  `check` (`companion-app.yml:47`) both run the core tests twice (`-p core`, then
  `--workspace`); in CI that is the argued no-GUI-libraries proof, but in the Makefile it proves
  nothing (the developer has the libraries): `rstest` can be `cargo test --workspace --locked`.
- `companion-app/README.md:216-226` (Layout tree and the paragraph after it) restates
  `Cargo.toml` and `main.rs`, and "Crates (nothing else without asking)" `:228-230` restates the
  `CLAUDE.md` dependency rule; the tree is the part that drifts. Keep two lines on `core/` and
  `app/` and link `CLAUDE.md` for the rule (about 8 lines).
- The Linux package list is typed twice (`companion-app.yml:25`, `README.md:237`) and the
  `tauri-cli` version thrice (`companion-app.yml:106`, `README.md:246`, `ASSETS.md:158`):
  README can say "the packages in `LINUX_GUI_PACKAGES` of the workflow"; ASSETS may keep its
  pin, as it records a generation event.
- Icons: `icon.icns` (92 KB), `128x128@2x.png` and `icon.png` serve macOS and Linux bundles
  that `bundle.targets: ["nsis"]` never builds (`tauri.conf.json:743-749`, ASSETS.md:158). Only
  the `.ico` and one PNG are needed on Windows. Check that `tauri-build` on the macOS CI leg
  does not demand the `.icns` before deleting; if it does, leave it.

## Not findings (argued or required, left alone)
`build.rs` command manifest and per-command permissions (the brief's "minimal capabilities");
`window.rs` `run_on_main_thread` wrapper (the single-instance callback is off the main thread);
`debug_log` (seven callers); `first_start.rs` marker file (the brief); the 15-row verify sheet;
`ASSETS.md`; the tray left-click handler beside the menu (the brief's "Open"; Linux comment is
useful); CSP and `freezePrototype`.

**Follow-ups:** None.
**Friction:** None.
**What carried it:** `git grep -i deep` over `f3436a5`, which proved the scope removal was clean
in one command, and `Cargo.toml:180` (`panic = "abort"`), which made the poison check in
`state.rs` visibly dead.
