# Director's check of the three lane plans (impl-director, 2026-11-13)

Direction only, against `docs/CONTRACT.md` at `eec6cb1`, addons D5 to D8, hub D3, D5, D10, D13 to
D17 and both overlays. No plan breaks the order or the ownership table or takes an owner's
decision by accident. **Nothing in them needs the owner.**

## 1. Per plan

**Hub (H): settled once the contract is folded.** Its departures from my sketch stand:
- `companion_version` dropped: right (nothing reads it); the wire follows, see §3.
- `next` after login is an allowlist for `/companion/link?code=` only, not "a path starting with
  one `/`": stands; its reason (`/\evil`, `/%09/evil`) is better than my sketch.
- A limiter on the approval POST, `sync` from a pending device answered 401 before the sequence
  is spent, routes kept in `app.js` though mounted above the session middleware: all stand.
- The one `profilePage` helper for four call sites stands (the card must survive a re-render);
  name it in the PR body for the reviewers.
- The Admin "Reachability" card (receive time, length, no member) is what D17 asked for.

**Addon (A): two changes before code**, both so it meets the hub:
1. Roster caps: "40 characters each" becomes **60**, the hub's per-member cap (its §4 proposal,
   accepted below); with 40, `/ff who` silently misses a member's 41st to 60th character.
2. Roster string bound: "1 to 64 bytes" becomes **1 to 160 bytes** for a member `name`. The
   hub allows a display name of 40 characters (`text(b.display_name, 40)` in `src/app.js`), up
   to 160 bytes in UTF-8; with 64 a member with a long name with umlauts vanishes from `/ff`.
   Character names stay at 64 (the hub cuts them at 24 characters).
3. Not a change, a consequence of the companion's accepted §2 proposal: `Data.lua` will exist
   with `members = {}` from install on. `/ff` must then say "no member list yet ...", not
   "0 members, 0 characters".
Its departures stand (predicate gates, probe decides; `Lines.clean`; coalesced
`SKILL_LINES_CHANGED`; `/ff status`). The coordinator's probe refinement is layer detail with no
direction problem. Question 1 is answered by `docs/RESEARCH.md` §6; 2 and 3 are verify-in-client.

**Companion (C): three changes before code.**
1. **`ValidateRoster` must not fail the whole answer.** As planned, an `ok: true` answer with a
   roster outside the companion's bounds counts as Unreachable. Then a roster entry the
   companion's bounds reject (a 61st character, a 100-byte display name) makes every sync an
   outage for ever, and the owner's Admin list fills with outages nobody had. Skip the entry
   that is out of bounds; refuse the answer only when its shape is wrong (not an object,
   `members` not an array). The bounds are the ones in §2 below, the same as the addon's.
2. **One version source:** drop the release's `-X .../app.version=$V`; the embedded `## Version`
   is the only one (the plan says so itself) and the tag check ties the tag to it.
3. **Release assets: Windows and Linux**, as the plan leans; keep the darwin cross-build in CI
   as a compile check, publish no macOS file (D7: macOS "builds and runs, started by hand").
Its departures stand: install after `linked` (a declined link leaves nothing); `windowsgui` plus
console; `%LOCALAPPDATA%` (a roaming key would give two PCs one key); `advapi32` and
`ShellExecuteW` (no child processes); lock file and `stop.request`; the committed parser sample.
Retrying after a failure every 30 s, then 120 s, stands but goes in the contract (§2).
- **14 packages** is the specialist's grain: the overlay asks for packages testable without a
  PC and free of cycles, which the plan's import rule meets. `pr-parsimony-review` judges the code.
- Copy: setup says "from the site's Game link page"; the address is on `/companion`: align it.

## 2. Rulings on the proposed contract corrections

From the hub:
- `device_name` cut to 40, control, format and line-separator characters dropped, empty becomes
  `PC`: **accept.**
- Race token unknown or not allowed "does not set a race; a new character gets none, a matched one
  keeps its own": **accept.** The literal reading would wipe a Skyborne race on every sync.
- Several entries with one name in one upload: the newest `seen` is used, the rest ignored
  `same name`: **accept.**
- At most 60 characters per member, more ignored `too many characters`: **accept**, and add to
  §3.3's roster paragraph "a member has at most 60 characters" so readers can bound it (§1 above).
- `accepted` and `ignored` present exactly when `characters` was sent, `ignored` `[]` when empty:
  **accept.**

From the addon:
- §1 "in the client's order, the first 40": **accept.**
- An empty read of the skill lines keeps the old list: **accept.**
- The key is opaque; readers use the fields: **accept.**
- A loaded-back table of another `version` is reset: **accept**, without asking the owner:
  version 1 is the first format, so nothing can be lost by it now. A later version bump is
  where the "ask before a format change that loses data" rule applies.
- §2 the addon doubles `|` and drops control characters before it shows a string: **accept** as
  an informational sentence in §2's addon bullet; no other party is affected.

From the companion:
- §2 empty `ForeverFriendsData` written at install: **accept** (one game restart instead of two),
  with "`members = {}`, `written = 0`" spelled out, since the addon must treat it as "no list yet".
- §3.1 a `link/status` answer the table does not name is retried at the same interval until the
  expiry: **reword** to "any answer other than 200, 400 and 401", because 400 (an unknown `v`)
  will not get better by asking again.
- §3.3 a 429 neither starts nor ends an outage: **accept.**
- §3.3 failures in the first minute after a wake from sleep are not counted: **accept**, worded as
  "after the companion resumes from sleep, failures in its first 60 s are not counted", which is
  observable in the outage the hub receives and nothing else.

New from this check, for one text:
- §3.3 timing: add "after a failed request the companion tries again sooner, at most every 30 s",
  because the companion plans 30 s retries and §3.3 says only 60 s and 300 s; the hub must know
  the fastest rate an honest companion sends at. It also means a hub restart longer than about
  30 s appears in the Admin list, which is what the owner asked to learn (D7, point 5).
- §3.3 roster: add "names are at most 160 bytes (display names) and 64 bytes (characters), at most
  60 characters per member; a reader skips an entry outside these bounds and keeps the rest",
  because the companion and the addon each invented bounds, and the hub never stated its own.
- §3.1 unlink: add "401 from `unlink` means the device is already gone: the companion treats it as
  done". The hub's plan answers 401 for a device removed on the profile, and the companion's plan
  already treats 401 as done; the readings agree, but the contract states neither, and a third
  reader would treat 401 as failure.

## 3. Direction questions

- **`companion` field: remove it from both bodies (`link` and `sync`).** Nothing reads it once
  the column is gone, and "no speculative parameters" applies to a wire field as much as to a
  function. Removing it later would be the breaking direction for an old hub that requires it;
  adding an optional one later is free. The test vector's body then changes; regenerated here
  with Go `crypto/ed25519` for body `{"v":1,"game_running":false}`, same seed, device and seq:
  body SHA-256 `637ea0ebc551e160c01667d569224ec68d187f006cb574458cbddef68177bc92`, signature
  `/ugHnkaZUd3KLobU7MuwmJfVZHMDdseFQ0WO5IjfJh0xvHvEWIB8dSWXhpVvPBcxaPCGbtg1srH1wV8UNAsUCw==`
  (public key unchanged). Confirm it in Node before folding, as the first one was.
- **Default hub address in the release build: no, and not an owner question.** The repository
  and its releases are public, the hub is private (the reason the contract lives here); a baked-in
  address publishes the group's server to anyone who downloads the file. The member types the
  address shown on `/companion`.
- Release assets, 14 packages, the addon probe: answered in §1; none is an owner question.

## 4. For the owner

Nothing to decide. For the handoff: antivirus heuristics on an unsigned program with a Run key
(a signing certificate is the real fix, their call), and a crash handler in the game folder may
count as "the game is running".
