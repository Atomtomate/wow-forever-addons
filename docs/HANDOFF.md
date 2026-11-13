# Handoff — 2026-11-13, second session (closing)

Read this after `CLAUDE.md`. Replace it when the next session closes.

## State

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

## What has to happen before anyone can use it

1. The hub on the server is updated to `main` by the owner's ops session, so that a companion
   has something to link to. `PUBLIC_URL` must be `https://` and `TRUST_PROXY=1` behind Caddy.
   Until then the companion stops with `could not reach the site (status 404)`.
2. The owner pushes the tag `v0.1.0` (it must be `v` plus the `## Version` line of
   `addons/ForeverFriends/ForeverFriends.toc`). The release workflow, which has never run,
   builds the files the hub's `/companion` page links to.

## First steps for the next session

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

## Decided by the owner this session

In D6 to D8 with their words: ForeverFriends first; a companion per member, written in Go, one
executable that carries and installs the addon and checks that the hub is reachable; the addon
list right after version one; a PC is linked by browser approval and signs every request (no
pasted token); the hub keeps the game's time of a character's last read and never shows it; the
group has Mac and Linux players.

## How this session ran, for the next one

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
