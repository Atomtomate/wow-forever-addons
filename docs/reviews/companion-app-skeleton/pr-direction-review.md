# pr-direction-review: `companion-app/` skeleton (PR #7)

Head `f3436a5`, base `main` `d31312e`. Held against `CLAUDE.md`, `.claude/agents.local.md`, and the
direction in open PR #5 at `112ed06` (D9 with the owner's answers of 2026-11-14, `docs/HANDOFF.md`
piece 1, `docs/RESEARCH.md` §7 "the one-hour experiment"), which this PR merges after.

**Verdict:** Yes, with one blocking point. The plan read's blockers are gone: no URL scheme, no
§5 edit here (the proposal is in #5's body), no third-party actions, no `license` field, WebView2
`downloadBootstrapper` now an owner-approved exception (D9, decision 7), the identifier and the
autostart-before-linking question are open points in D9. The code stays inside `companion-app/`,
the `Makefile`, `.gitignore` and the new workflow; no forbidden path is touched. What does not yet
serve the direction is the "Verify on the PC" sheet: it is the whole reason piece 1 reaches the
owner's PC (`docs/HANDOFF.md` piece 1; `docs/RESEARCH.md` §7 hands the experiment to it), and it
misses several things the owner asked for and things D9's reversal triggers depend on.

### 1. The Verify sheet misses parts of the owner's brief and of the §7 experiment (blocking)

- **What** — `companion-app/README.md:64-87` leaves out or blurs these, each needed for the dated
  "Observed" section:
  - **Edge and Chrome** (`:73`): "your browser" only; §7 says both, each written down.
  - **A build one byte different** (missing): §7 asks for it, and D9 now claims an unsigned file
    "warns at every new version". Running the workflow twice gives two different files; download
    the second and compare.
  - **What was downloaded** (`:58-59`, `:73`): an Actions artifact arrives as a `.zip` from a
    signed-in GitHub page, not as the `.exe` friends will get, and browsers warn differently for
    each. The row must say so, and say to unzip with Windows' own "Extract all" (other tools
    may drop the "downloaded from the internet" mark, and SmartScreen with it).
  - **SmartScreen text** (`:74`): asks yes/no, not what the blue window says (title, "Unknown
    publisher" or a name, which buttons).
  - **Smart App Control** (`:75`): checked *after* running the installer, though its state decides
    what step 2 shows. Move it to step 0, add "do not change this setting for the test", and
    "if On: write down the message word for word; the remaining steps need a PC where it is Off
    or Evaluation".
  - **Defender** (`:76`): only "during the install"; the brief says at download, install, first
    start and when switching autostart on. One column "Defender said anything?" on rows 1, 2, 6
    and 11 does it.
  - **The tray menu itself** (missing): does the right-click menu look like other tray menus
    (font, dark mode)?
  - **The notification** (`:78`): shown or not, but not how it looks: what name and icon it
    carries (an unsigned app's notification can show an odd sender).
  - **Window beside Windows Settings, in light and in dark** (`:79`): "switch Windows to dark"
    only; no comparison with Settings and no light-mode check. "Looks native?" is too vague to
    fill: ask "put it next to Settings: does the font, size and colour look like it belongs?"
  - **Folder picker** (`:86`): asks whether the choice is kept, not whether it is the normal
    Windows folder window.
  - **Scaling** (missing): at 125 % or 150 % display scale (Settings, System, Display), sharp or
    blurry?
  - **Web-page tells** (`:79`): right-click only. Add: can text be selected by dragging; does
    Ctrl+R or F5 reload; does Ctrl and + zoom; does Tab show a web-style focus ring.
  - **Do the WebView2 processes exit** (`:81`): D9's reversal table says "a destroyed Tauri window
    leaves web-view processes behind ... → Avalonia first", so this is a decision input, not a
    number. Ask it outright: "wait 30 seconds after closing: are they gone?" And `msedgewebview2.exe`
    in Details also counts Widgets, Teams, Outlook and others; tell the owner to use the
    Processes tab and expand the "Forever Friends" group, which lists only its own.
  - **Context row** (missing): date, Windows edition and version (Settings, System, About), Edge
    and Chrome versions, display scale, whether "Microsoft Edge WebView2 Runtime" was already in
    Settings, Apps before installing (the one case where the installer goes to the network).
- **Where** — `companion-app/README.md:64-87`; bears on `docs/RESEARCH.md` §7 (#5, lines 357-363,
  and the reversal table above it) and `docs/HANDOFF.md` piece 1 (#5).
- **Why it matters** — The owner runs this once. Whatever the sheet does not ask is not observed,
  and the triggers in D9 (Avalonia first if processes linger or the window looks like a page;
  signing not optional if Smart App Control is On) get decided on a guess.
- **Suggested resolution** — Rewrite the table with the items above, still one plain question per
  cell; keep "Write down what happened" short (a word, a number, or the exact message). Add one
  line under the table: the `foreverfriends://` link of §7 step 2 is not tested here (D9,
  decision 6; piece 8), so the Observed section does not read as incomplete.

### 2. Uninstall row tests the easy case only (non-blocking)

- **What** — Row 12 switches autostart off before row 15 uninstalls, so "uninstall with
  autostart on" (the Run entry left behind, the leftover the brief asks about) is never seen; the
  install folder (`%LOCALAPPDATA%\Forever Friends\` per `plan.md`) and the Start-menu entry are not
  in the list, and the app is still running in the tray during uninstall.
- **Where** — `companion-app/README.md:84-87`; the §5 proposal in #5's body (install folder,
  Run entry, `EBWebView`).
- **Why it matters** — The §5 change in piece 4 or 5 is to be written from these leftovers.
- **Suggested resolution** — Switch "Start with Windows" on again before uninstalling; list the
  install folder and Start menu; ask "did the uninstaller close the running app or ask you to?".
  Tell the owner how to open the paths: paste them into File Explorer's address bar.

### 3. The pull request body does not yet name what only the code records (non-blocking)

- **What** — #7's body is a stub. D9 is silent on several choices this diff makes, and #5's
  proposed rule (`CLAUDE.md:65` there) is that crates are named in the pull request body for the
  owner.
- **Where** — PR #7 body; `companion-app/README.md:42-44,54`, `companion-app/app/src/platform.rs:6-10`,
  `companion-app/app/tauri.conf.json:31,46-47`, `companion-app/app/src/window.rs:1-2`.
- **Why it matters** — These are owner-visible choices; in the README alone they read as settled.
- **Suggested resolution** — The body lists, as asks: the crates (`tauri`, `tauri-build`, four
  first-party plugins, `serde`); `rslint`/`rstest` outside `make check` and the pre-commit hook;
  macOS without a dock icon (`Accessory`); installer publisher "Forever Friends" (signing in
  piece 5 will show the owner's legal name instead); English-only installer; the window destroyed
  on close (lower idle memory, slower reopen, which rows 9 and 10 measure).

### 4. `ASSETS.md` points at a licence note that does not exist (non-blocking)

- **What** — `companion-app/ASSETS.md:7` says "the owner decides (see the root `README.md`)"; the
  root `README.md` says nothing about a licence. The decision lives in D9, decision 5 (deferred).
- **Where** — `companion-app/ASSETS.md:7-8`.
- **Why it matters** — On the question asked: "no licence stated" is acceptable for original art.
  The hard rule is provenance (original or free, recorded), and the row records that well: hand
  drawn here, plain shapes, the generator and its version named. With no repository licence the
  icon simply falls under whatever the owner chooses for the repository. Only the pointer is wrong.
- **Suggested resolution** — Column text: "Original. Covered by the repository's licence once the
  owner chooses one (D9, decision 5: deferred)." Same for row 8.

### 5. `docs/reviews/companion-app-skeleton/plan.md` still describes the deep link (non-blocking)

- **What** — The committed plan carries the `foreverfriends://connect?code=` scheme, its core
  parser (§0.5) and an HKCU scheme key for §5; D9 removed all of it from this piece.
- **Where** — `docs/reviews/companion-app-skeleton/plan.md` §0.5, "Changes the other session
  makes", §3, §4.
- **Why it matters** — It is a record, but the next session reading the review folder could take
  it as the design. (It is also not strictly a review report, the one `docs/` exception.)
- **Suggested resolution** — One line at its top: "Superseded where D9 (#5) differs: no URL
  scheme in piece 1 (decision 6)." Or drop it and let the two plan-read reports carry the record.

## Decisions

- Rust + Tauri 2 tray app in `companion-app/`, Go stays until parity — recorded (D9, #5).
- No URL scheme in this piece — recorded (D9, decision 6).
- WebView2 `downloadBootstrapper`, silent — recorded (D9, decision 7; `README.md:61-62`).
- Identifier `io.github.atomtomate.foreverfriends` — recorded as open, owner confirms (D9).
- Autostart toggle usable before linking — recorded as open (D9); `HANDOFF.md` piece 1 allows it.
- Version `0.0.1`, apart from the addon's number — recorded as open (D9).
- Unsigned per-user NSIS, Windows-only bundle, CI build on three systems — recorded (`HANDOFF.md`
  piece 1, D9 decision 1).
- On-PC footprint ahead of §5 — recorded (`HANDOFF.md` piece 1; proposal in #5's body).
- Crates `tauri`, plugins, `serde` — in `README.md:42-44`; body not yet (finding 3).
- `rslint`/`rstest` outside `make check` and pre-commit — in `README.md:54` only (finding 3).
- macOS `Accessory`, no dock icon — in `README.md:21` only (finding 3).
- Installer publisher "Forever Friends", English only — silent (finding 3).
- Window destroyed on close, rebuilt on open — in code comments only (finding 3).
- Page may call two commands and listen; plugins driven from Rust only — recorded
  (`capabilities/default.json`).
- Only first-party CI actions, runner's own `rustup` — recorded (`companion-app.yml:3,27`).

## Follow-ups

- `docs/RESEARCH.md` §7 (#5, line 360) still lists "click a `foreverfriends://` link" in the
  experiment piece 1 carries; #5 can mark it as moved to piece 8.

## What carried it

`docs/RESEARCH.md` §7 in #5 (the one-hour experiment and the reversal table): it turned "is the
sheet complete" from taste into a checklist with consequences.
