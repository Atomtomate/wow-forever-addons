# pr-direction-review: plan read, `companion-app/` skeleton (Rust + Tauri 2)

Plan: the session's scratchpad `plan.md`, base `main` at `d31312e`. Held against `CLAUDE.md`,
`.claude/agents.local.md`, `docs/DECISIONS.md` D5/D7/D8, `docs/CONTRACT.md` §3.1 and §5,
`docs/IDEAS.md`, `docs/HANDOFF.md`, `README.md`. Known and not this PR's: D7 ("Go, standard
library only") and `docs/IDEAS.md` ("a tray icon ... later") are settled by D9 in the parallel
"Companion app 0" pull request.

**Verdict:** Yes, with two blocking points. The skeleton serves the owner's 2026-11-14
direction and keeps to the hard rules (no hub address, no network from the app, nothing
persisted beyond a marker, Tauri first-party plugins only, `ASSETS.md`, no npm). But the deep
link gives the app a hub-to-PC channel the contract does not have, and the §5 changes the plan
correctly hands to the coordinator are not in the parallel PR's file list, so they land nowhere.

### 1. The `foreverfriends://connect?code=X` deep link is a new interface, taken silently (blocking)

- **What** — The plan says "no change to `docs/CONTRACT.md` formats", yet registers an OS-wide
  URL scheme whose `code` parameter is meant to be emitted by the hub; `docs/CONTRACT.md` §3.1
  runs the other way (the companion makes the code, shows it, and opens the browser; "the hub
  does not name a page for it to open").
- **Where** — plan §3 (`plugins.deep-link.desktop.schemes`), §4 "Links", §0.5, §8 core tests;
  bears on `docs/CONTRACT.md` §3.1 step 3 and `CLAUDE.md` ("A change to a format changes it
  there first").
- **Why it matters** — A `code` carried *into* the app reads as a reversed linking flow (the
  user code travelling hub to PC), which D8 chose against on security grounds ("No person ever
  sees a secret"; the member approves the code the PC shows). Any web page can fire the scheme,
  so its payload is untrusted input from the open web. Built here with tests and a Verify item,
  the shape `connect?code=[A-Za-z0-9-]{1,32}` becomes the de-facto design the next session
  builds on without anyone having decided it.
- **Suggested resolution** — Keep the scheme plumbing (registration, single-instance
  forwarding, `textContent` display), but record what it is: either D9 (in the parallel PR)
  says the scheme exists, its payload has no meaning yet and the §3.1 flow is unchanged until
  `docs/CONTRACT.md` gains a section for it; or rename the parameter so it cannot be read as
  `user_code` (e.g. `foreverfriends://open?test=X`) and say "placeholder, not a contract" in
  `companion-app/README.md`. If the owner's brief asked for a "connect" link specifically, the
  owner is asked which direction the code travels before it is built.

### 2. The `docs/CONTRACT.md` §5 update has no pull request to land in (blocking)

- **What** — Plan "Changes the other session makes" lists a §5 entry (marker file, HKCU Run
  entry, HKCU `foreverfriends://` registration, install folder, WebView2's `EBWebView` folder),
  but the parallel PR's stated scope is D9, `RESEARCH.md`, `IDEAS.md`, `HANDOFF.md`,
  `CLAUDE.md` and `agents.local.md`: `docs/CONTRACT.md` is not on it.
- **Where** — `docs/CONTRACT.md` §5 last paragraph ("its own config folder ..., its own
  program folder, and its autostart entry"); the party table at `docs/CONTRACT.md:7-11`
  (companion = `companion/`, Go); `.claude/agents.local.md` invariant "and nothing else".
- **Why it matters** — §5 is the one home of what the companion may write on a PC. The app
  writes a URL-scheme key and a WebView2 data folder that §5 does not allow; the architecture
  reviewer will hold it to "nothing else", and a friend reading §5 is told something false.
- **Suggested resolution** — Coordinator adds `docs/CONTRACT.md` (§5 and the party table:
  "the companion app, `companion-app/`, Rust, not yet shipped") to "Companion app 0", and this
  PR's first line says "merge after #N" per `CLAUDE.md`.

### 3. Manual "Start with Windows" before linking departs from §3.1 (non-blocking)

- **What** — The tray toggle can register autostart on an unlinked app; `docs/CONTRACT.md`
  §3.1 step 4 says the companion "registers its autostart only once `linked` is true".
- **Where** — plan §4 "Tray"; `docs/CONTRACT.md` §3.1.
- **Why it matters** — Once the app is the companion, two rules disagree; the next session
  picks one by accident.
- **Suggested resolution** — One line in D9: the rule binds automatic registration; a toggle
  the member presses is their choice. Or grey the toggle until linked (then nothing to record).

### 4. The installer downloads and runs Microsoft's WebView2 bootstrapper (non-blocking)

- **What** — `downloadBootstrapper` makes the installer fetch and run code; D7 says the
  companion never "run[s] anything it downloaded", and the session's hard rule is "no network
  requests".
- **Where** — plan §0.3, §3 `bundle.windows.webviewInstallMode`; `docs/DECISIONS.md` D7.
- **Why it matters** — The choice is sound (offline installer is ~130 MB), but it is an
  exception to a written rule, and plan §0.1's D9 text names WebView2 only as "a runtime
  dependency".
- **Suggested resolution** — D9 says: the installer may download WebView2 from Microsoft when
  absent; the app itself makes no network request. Add the case to the Verify sheet (§6).

### 5. Root `README.md` goes stale and neither PR lists it (non-blocking)

- **What** — The layout table (`README.md:28-40`) lists `companion/` as "the companion program
  (Go, standard library only)" and the two workflows; `companion-app/` and
  `companion-app.yml` are absent, and "For the companion: Go 1.24 or newer, nothing else" no
  longer covers it.
- **Where** — `README.md:28-40`, `README.md:50-57`; the manual is what a reader opens first.
- **Suggested resolution** — Coordinator adds `README.md` to "Companion app 0", or this PR adds
  two table rows and `make rslint rstest` (README.md is not on this PR's forbidden list).

### 6. `license matches repo`: the repository has no licence (non-blocking)

- **What** — Plan §1 sets `[workspace.package] license` "matches repo"; there is no LICENSE
  file and no licence named anywhere in the docs.
- **Why it matters** — The specialist would pick a licence for a public repository, a decision
  only the owner takes; `ASSETS.md`'s "drawn here" icon has the same gap.
- **Suggested resolution** — Omit the `license` field (or `publish = false`); `ASSETS.md`
  states origin only. Ask the owner about a repository licence separately (Follow-ups).

### 7. Dependencies outside "Tauri + first-party plugins" (non-blocking)

- **What** — `serde` (direct dependency) and the CI actions `dtolnay/rust-toolchain`,
  `Swatinem/rust-cache` are third-party; existing workflows use only `actions/*`.
- **Where** — plan §2, §5; `CLAUDE.md` "Ask before a new library dependency".
- **Suggested resolution** — Name all three in D9's approved set or in the PR body as asks.
  The actions can be avoided: GitHub runners ship `rustup`, and `actions/cache` covers caching.

### 8. Windows-only bundle against "we have mac and linux users" (non-blocking)

- **What** — Bundles, autostart labels aside, are Windows-only; Linux release scheme and macOS
  bundle out of scope; D7 records Mac and Linux as the work right after version one.
- **Suggested resolution** — Fine for a skeleton; D9 says Windows-first holds for the app too,
  so the build matrix is not read as a promise.

## Decisions

- Rust + Tauri 2 tray app as the future companion — recorded (D9, parallel PR).
- Go companion stays the shipped one until parity — recorded (D9, per plan §0.1).
- Identifier `io.github.atomtomate.foreverfriends` — flagged to owner (plan §0.2).
- WebView2 `downloadBootstrapper` silent — flagged; its D7 exception silent (finding 4).
- `rslint`/`rstest` outside `make check` and the pre-commit hook — flagged (plan §0.4).
- Deep-link parser in the core crate — flagged (plan §0.5).
- macOS `Accessory` policy — flagged (plan §0.6).
- URL scheme `foreverfriends://` and the `connect?code=` payload — silent (finding 1).
- New on-PC footprint (marker, Run entry, scheme key, EBWebView) — recorded in plan, no
  landing PR (finding 2).
- Autostart toggle available before linking — silent (finding 3).
- NSIS `currentUser`, Windows-only bundle, no updater, no signing — recorded (plan §3, §9).
- Capabilities exclude `core:default` and all plugin permissions from the page — recorded.
- Tray app keeps running after its window closes — recorded (plan §4).
- `serde` and two third-party CI actions — listed, not yet approved (finding 7).
- Crate licence — silent (finding 6).
- App version `0.0.1`, separate from the TOC-driven release number — silent; harmless while
  `release.yml` does not build the app, D9 should say whose number the app follows later.

## Follow-ups

- `README.md` (root): the repository has no licence file; the owner may want one for a public
  repository before anything is distributed.
