# Handoff — 2026-11-14 (the companion app record), over 2026-11-13's second session

Read this after `CLAUDE.md`. Replace it when the next session closes.

## The companion app: the plan of pieces (2026-11-14)

The owner decided that the companion becomes a desktop app in Rust and Tauri 2, tray-first,
with the Go program in `companion/` kept as the reference implementation until the app reaches
parity (`docs/DECISIONS.md` D9, which also lists the decisions still open). The research is
`docs/RESEARCH.md`, "Companion app: choosing the technology". The coordinator session cut the
work into the pieces below, one session and one pull request each. **This is the plan, not
decided scope**: the owner can reorder or cut it.

0. **This record** (D9, the research, the backlog, this plan). Documentation only.
1. **App skeleton** in `companion-app/`, run in parallel with piece 0; touches only
   `companion-app/`, the `Makefile` and `.github/workflows/`. Tauri 2, tray-first, a window
   only on demand, an autostart toggle, single instance, a folder picker, a
   notification, an original placeholder icon, a per-user Windows installer, CI on Windows,
   macOS and Linux, and a "Verify on the PC" sheet the owner runs to see the unsigned install
   and the native feel. It adds the `make` targets a Rust change must pass and names them in its
   pull request; writing them into `CLAUDE.md` and `.claude/agents.local.md` falls to piece 2.
   It is a skeleton for the owner's PC, not a release to members, so it may install and
   register autostart ahead of the §5 change that piece 4 or 5 makes. No URL scheme (D9,
   decision 6: the member types the hub's address).
2. **Core port A**: a Rust library crate with no Tauri dependency: the wire formats, request
   signing, the hub client, config, atomic file writes and the log, proven byte-for-byte against
   the Go program with golden files generated from it.
3. **Core port B**: reading SavedVariables, finding the game folder, the embedded addon and its
   installation, the member list file, the sync loop, telling that the game runs.
4. **First-run setup and linking** in the app's window; status in the tray. It or piece 5
   changes `docs/CONTRACT.md` §5 to name the installer folder, the autostart entry, the app's data
   folder and the web view's cache (proposed in the body of the piece 0 pull request), before
   any release to members. The member types or pastes the hub's address here (D9, decision 6).
5. **Windows release**: the installer, autostart, the release workflow rebuilt for builds per
   operating system, signed with the owner's certificate (D9, decision 1; the certificate
   authority is chosen before this piece); the Go companion, its workflow and `go.mod` removed
   (D9, decision 4).
6. **macOS and Linux** (the `docs/IDEAS.md` entry that was next after version one).
7. **Updates**: the owner said yes (D9, decision 2); a decision entry with the design and the
   `docs/CONTRACT.md` change come first, in this piece.

The former piece 8, the "Connect this PC" link, is dropped: the member provides the hub's
address (D9, decision 6); the link is parked in `docs/IDEAS.md`. Discord is backlog, not part of
the first release.

What this changes in the list below: the tag `v0.1.0` is never pushed and the Go companion's
"Verify on the PC" run is not done (D9, decision 4, the owner: "yes"); the first release is the
Tauri app, and the "Verify on the PC" run is piece 1's sheet. Everything about the addon and the
hub stands. Still open in D9: the licence file (deferred), WebView2 in the installer, macOS
notarization, and the open points for the pieces.

## State on 2026-11-13, still current

- **The order changed.** PR #1 (MarketScan) is merged on the owner's word; it has not run in the
  game. The owner then made the first addon the group installs **ForeverFriends**, the link
  between the game and the hub: members, their characters, online status (D6 to D8). Three
  programs build against one file, `docs/CONTRACT.md`.
- **Merged on 2026-11-13, on the owner's word, all six:**

  | Pull request | What | Review pass |
  |---|---|---|
  | #3 `lane/foreverfriends` | D6 to D8, `docs/CONTRACT.md`, the addon (96 specs) | four reviewers, folded |
  | #4 `lane/companion`, stacked on #3 | the companion, Go (13 packages), release workflow | four reviewers, folded |
  | #2 `tooling/agent-fleet` | the agent fleet as a submodule, the overlay, friction notes | not run: docs and tooling |
  | `Atomtomate/wow-forever-hub#12` | the hub's side: linking, signed sync, "in game" (290 tests) | five reports, folded |
  | `Atomtomate/wow-forever-hub#11` | the fleet mounted in the hub | not run |
  | `Atomtomate/agents#28` | an overlay skeleton and notes from this session | not run |

  The reports are under `docs/reviews/<branch>/`; each pull request's body says what was folded
  and what was declined, with the reason.
- **Proved here:** an end-to-end run in the cloud container with the real hub on loopback, the
  real companion binary and the real addon in the simulated client. Link through the hub's
  approval form; upload; a character without a skill list keeps the professions typed on the
  site; an older file changes nothing; the member list is written, loaded by `lua5.1` and
  answered from by `/ff`; a display name built to break out of the Lua file stays a string;
  Remove on the profile makes the next sync "unlinked". CI runs the Go tests on Ubuntu and on
  Windows; the Windows runner found one real parser defect (CRLF), fixed.
- **Not proved anywhere:** anything in the real game, and anything on a real PC. Two lists say
  exactly what to check: "Verify in client" in `addons/ForeverFriends/README.md` (8 items) and
  "Verify on the PC" in `companion/README.md` (15 items).

### What has to happen before anyone can use it

Since D9 (2026-11-14), item 2 is dropped: v0.1.0 is never tagged, and members get the Tauri app
(decision 4). Item 1 stands for whatever companion links first.

1. The hub on the server is updated to `main` by the owner's ops session, so that a companion
   has something to link to. `PUBLIC_URL` must be `https://` and `TRUST_PROXY=1` behind Caddy.
   Until then the companion stops with `could not reach the site (status 404)`.
2. The owner pushes the tag `v0.1.0` (it must be `v` plus the `## Version` line of
   `addons/ForeverFriends/ForeverFriends.toc`). The release workflow, which has never run,
   builds the files the hub's `/companion` page links to.

### First steps for the next session

These were written before D9. The tag and the Go companion's "Verify on the PC" run in steps 1
and 2 are replaced by the plan at the top; the addon's "Verify in client" list, steps 3 to 5 and
the hub update still stand.

1. Ask the owner whether the ops session has updated the hub and whether the tag `v0.1.0` is
   pushed; look at the release workflow's first run.
2. With the owner at the PC, after the release: the companion's "Verify on the PC" list first
   (SmartScreen and Defender on an unsigned program that copies itself and adds a Run key is
   item 12: the owner sees it before any friend does), then the addon's "Verify in client" list.
   Record what the client and the PC actually did in `docs/RESEARCH.md` as a dated "Observed"
   section, and change `CLAUDE.md`'s facts if the client disagreed.
3. From that run, fix the tables in `docs/CONTRACT.md` §4 and the hub's `src/companion.js`: the
   two Skyborne race tokens are unknown, and the profession skill ids are the ones Classic and
   retail share, unverified for Forever.
4. Then the next work, both marked next in `docs/IDEAS.md`; the owner says which first:
   **macOS and Linux** (the group has Mac and Linux players; in version one they link and upload
   but never show as in game) and **the addon list per member**.
5. MarketScan's own "Verify in client" list is still open.

### Decided by the owner on 2026-11-13

In D6 to D8 with their words: ForeverFriends first; a companion per member, written in Go, one
executable that carries and installs the addon and checks that the hub is reachable; the addon
list right after version one; a PC is linked by browser approval and signs every request (no
pasted token); the hub keeps the game's time of a character's last read and never shows it; the
group has Mac and Linux players.

### How the 2026-11-13 session ran, for the next one

- **Two corrections from the owner at the close; both are rules in `CLAUDE.md` now.** The
  finished pull requests had all been left as drafts, so nothing told the owner what to review.
  And one session ran every lane and every review in a single context that grew from 90,000 to
  900,000 tokens: of the 150 million tokens it read, 89% were read above 400,000. One session
  now carries one pull request, and `.claude/settings.json` caps a context at 400,000. For
  several pieces, open one session per piece (a cloud session can open them itself).
- The owner's agent fleet did the work: `impl-director` cut the plan, the direction and
  architecture reviewers read the plans before code, three specialists built in three
  worktrees, four reviewers read each pull request. `.claude/agents.local.md` (pull request #2)
  holds this repository's answers to the fleet.
- A cloud session that starts outside a checkout does not list the fleet's agents; they run
  through the fleet manual's fallback. Add `Atomtomate/agents` to the session's repositories.
- **The pass for three pull requests used up the owner's five-hour limit once.** Two runs died
  mid-work and were resumed from where they stopped once the limit reset. Resume an agent
  (a few thousand tokens) instead of starting a new one (150k or more). The numbers are in
  `Atomtomate/agents#28`.
- busted in a fresh cloud container: `luarocks install busted` cannot fetch two of its
  dependencies there (GitHub source archives). What worked: a three-line stand-in
  `/usr/local/share/lua/5.1/term.lua` returning `{ isatty = function() return false end }`, and
  `mediator.lua` (one pure-Lua file) fetched from `raw.githubusercontent.com/Olivine-Labs/mediator_lua/v1.1.2-0/src/mediator.lua`
  into the same folder. CI no longer has the problem: it installs from Ubuntu packages.
- The end-to-end run was a throwaway script outside the repository (it needs both
  repositories checked out). Keeping one under `tools/` is worth a thought; it found nothing
  the reviews did not, but it is the only thing that runs all three programs together.
