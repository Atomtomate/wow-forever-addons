**Verdict:** Yes, with records to fix first. The plans stay inside addons D6 to D8 and hub D17. They build nothing that either `docs/IDEAS.md` marks later or parked, and they bend neither of the hub's two rules nor D10 beyond what D17 states. Five findings: one rule that contradicts the contract and D17, one narrowing of D7 for macOS (and likely Linux), and three records left stale.

Path roots: `addons:` is `/home/user/wt-ff` (`lane/foreverfriends`, contract at `970ae98`), `companion:` is
`/home/user/wt-companion`, `hub:` is `/home/user/wt-hub-companion`. "Check" is
`addons:docs/reviews/lane-foreverfriends/impl-director-check.md`.

### 1. macOS, and likely Linux, lose part of what D7 says the program does, and only a review file records it

- **What**: The settled plan publishes no macOS file. On macOS its companion never reports the game as running, and on
  Linux the same process-path method likely fails too, because the game runs under Wine there. D7, however, says "On macOS and
  Linux the same program builds and runs, started by hand". The owner chose Go over PowerShell because "it runs on Windows,
  macOS and Linux".
- **Where**: check `:42-43` (publish no macOS file); `companion:docs/reviews/lane-companion/impl-backend-plan.md:125-126`
  (macOS lists no process and sends `game_running:false`), `:154` and `:197` (the specialist built darwin and asked the owner);
  against `addons:docs/DECISIONS.md:73-75` and `:97-99`, and D6's "their online status" (`:49-50`).
- **Why it matters**: A member on a Mac cannot get the program without a Go toolchain. If they do build it, they never show
  "in game", which is D7's third duty and the thing the owner asked for in D6. The check cites D7 for dropping macOS, yet D7
  says the same words of Linux, which is published. Nobody reading `DECISIONS.md` would learn either narrowing, and the
  companion README the plan writes is not where the overlay keeps decisions.
- **Suggested resolution**: Record it in a new entry, because DECISIONS is append-only: say what version one does on macOS
  and Linux (which files are published, game not detected). Add one line in the PR body asking the owner whether any member
  plays on a Mac. If one does, publishing the darwin build the plan already makes is one line of `release.yml`. My only
  disagreement with the check is that it calls this "not an owner question". With no Mac player that is true, but the record
  has to say so either way.

### 2. A synced character can lose its role, against contract §4 and hub D17

- **What**: The hub plan clears a character's role when the game's class does not allow it. The contract and D17 both say
  the role stays what the member chose.
- **Where**: `hub:docs/reviews/lane-m-companion/impl-backend-plan.md:96-97` ("Class change that makes the kept role illegal
  clears the role"). Also `:102`: `applyMain` then copies the emptied role into the profile's `main_role`, as `setMain` does at
  `hub:src/characters.js:72-73`. Against `addons:docs/CONTRACT.md:336-338` and `hub:docs/DECISIONS.md:321`.
- **Why it matters**: It sits under the plan's heading "Rules where the contract is silent", but the contract is not silent
  on roles. The check accepted the hub's five contract proposals and does not mention this rule. The first sync of a
  character typed on the site with the wrong class empties a choice the record promises to keep. The contract calls §4
  "observable rules", so it would be wrong from day one.
- **Suggested resolution**: Either keep the rule and write it into §4 first ("...stay what the member chose, except a role the
  game's class does not allow, which is cleared, as the site's form does": `hub:src/characters.js:29`), with a dated note under
  D17, or drop it and keep the illegal role. The session's call; neither needs the owner.

### 3. The line the reviewers enforce says the companion writes only into the two addon folders

- **What**: `CLAUDE.md` and the overlay say the companion "writes only into the two ForeverFriends addon folders (D7)". The
  plans also write a config folder, a copy of the program and a Run key. Contract §5 has allowed those since `4b746f7`, and D7
  never says the narrow version.
- **Where**: `addons:CLAUDE.md:44-45`; `/home/user/wt-addons-fleet/.claude/agents.local.md:77-78` (under "Invariants the
  reviewers enforce"); against `addons:docs/CONTRACT.md:382-386` and the companion plan's `:43-49`, `:166-167`.
- **Why it matters**: The plan's first pass widened §5 (`git log -S"program folder"` finds `4b746f7`) and stopped one file
  short. Reviewers read that line as the rule they hold code to. `pr-architecture-review` will then flag `%LOCALAPPDATA%`, the
  copied program and the registry entry, or a later change will take the narrow line at its word and cite D7 for it.
- **Suggested resolution**: Give both lines §5's shape: "inside the game folder, writes only into the two ForeverFriends
  addon folders; outside it, only its config folder, program folder and autostart entry (`docs/CONTRACT.md` §5)". This is
  the session's edit; the overlay line is on the fleet branch.

### 4. The documents readers open first are not updated by any plan

- **What**: After lane C the addons repository has a Go module, a release workflow and a Go check before each commit. The
  documents a newcomer and the owner read first still describe Lua only, and no lane plan lists them.
- **Where**:
  - `addons:README.md:34`: `ci.yml` is described as "luacheck and busted".
  - `addons:README.md:22-35`: no `go.mod`, `embed.go`, `Makefile` or `release.yml`.
  - `addons:README.md:39-40`: requirements list Lua only.
  - `addons:CLAUDE.md:35`: `make lint test` is the commit check, but the plan's pre-commit also runs the Go checks
    (companion plan `:151-152`).
  - `addons:docs/IDEAS.md:6`: still says "in progress".
  - `addons:docs/CONTRACT.md:11`: the hub's code is `src/companion.js` only; the plan adds `src/devices.js`.
  - The release rule appears in no document the owner reads, and the owner is the one who tags
    (`addons:...impl-director.md:102-103`). The rule: the tag must be `v` plus the `## Version` line of `ForeverFriends.toc`,
    CI refuses any other, and addon and companion share that one number (companion plan `:103`, `:153`; check `:40-41`).
  - Hub: `hub:README.md:20-23`, `:49-53`, `:62-65` (the Characters, Right now and Admin bullets), `:68` (the line count) and
    `:185-204` (Layout), plus `hub:CLAUDE.md:52` (the `db.js` row, which gains `CHARACTER_COLUMNS_ADDED_LATER`). The hub plan's
    README step (`:17-18`) covers none of these.
- **Why it matters**: The owner's first tag fails with no text explaining why. A companion-only fix needs an edit to the
  `.toc`, which belongs to `impl-frontend`; the overlay says the companion only reads it. The hub's `CLAUDE.md`: "A change that
  makes `README.md` ... wrong is not finished."
- **Suggested resolution**: Name these in the build. Addons side, from lane C or the session: a "Releasing" paragraph and the
  Go lines in the root README, the commit-check line in `CLAUDE.md`, and IDEAS and CONTRACT `:11` at merge. Hub side: the README
  bullets and Layout, in H's README step.

### 5. The hub's D5 says when to switch stacks, the build meets that condition, and the record does not say why it stays

- **What**: D5 says "the hub has one [client] ... Switch when something needs client-side state or a second client, not
  before". The companion is a second client calling a JSON API, and neither D17 nor any plan addresses that condition.
- **Where**: `hub:docs/DECISIONS.md:37-40`. The director reads D5 only as "one dependency" (`addons:...impl-director.md:14-15`).
  The nearest answer is the hub overlay's "Everything else the hub serves is HTML ... no outside reader", which is outside
  the decision record.
- **Why it matters**: "The hub has one" becomes false at merge. The next reader holding the hub to D5 either argues the stack
  again or reads the API as a decision nobody took.
- **Suggested resolution**: Add a dated note under D5, which is the hub's rule for a decision that changes. It would say the
  companion's four routes are not the second client D5 meant: their format lives in the addons' `docs/CONTRACT.md`, where
  OpenAPI would sit, so the stack stays. If the session would rather not interpret the owner's condition itself, ask the
  owner in one line.

**Decisions**

- A PC is linked by browser approval and signs every request: recorded (addons D8, hub D17, contract §3.1 and §3.2).
- The PC's name and public key reach the hub: recorded (addons D8's last bullet, contract §5).
- `/api/companion/*` is exempt from rule 1, and the approval page and `/companion` are member pages: recorded (hub
  `CLAUDE.md`, D17).
- One `devices` table, an `outages` table and `characters.from_game`, all additions: recorded (D17 "Additions only").
- "In game" with a 150 s window, no character name and no poll, hiding "Quiet right now": recorded (D17). The hiding follows
  Steam's precedent in `hub:src/views.js:173`.
- An Admin reachability card showing twenty outages without the member: recorded (D17).
- The `companion` field removed from the wire; the empty data addon written at install; 60 characters per member; the roster's
  bounds; a matched race kept; duplicate names resolved by the newest `seen`; reset on an unknown version; 30 s retries;
  failures forgiven after sleep; 429 counted as neither failure nor success: recorded (contract at `970ae98`).
- A role cleared when the game's class forbids it: silent, and it contradicts contract §4 and D17 (finding 2).
- No macOS release file; macOS, and likely Linux, never report the game running: silent, recorded only in review files
  (finding 1).
- One version number for addon and companion, and the tag must match it: silent (finding 4).
- No hub address built into the release: silent, recorded only in the check (`:110-113`). Cheap to reverse; no finding.
- The addon is installed only after the link is approved: silent, recorded only in the check. Within D7; no finding.
- After login, `next` is accepted only for `/companion/link?code=`: silent. It implements contract §3.1; no finding.
- Config in `%LOCALAPPDATA%` and the program copy in `%LOCALAPPDATA%\Programs`: silent on the location. §5 names the folders
  in general terms; no finding.
- `/ff status`, and ASCII-only case folding in `/ff who`: silent. Within D6's "behind `/ff`"; no finding.
- The hub keeps D5's stack while gaining a second client: silent (finding 5).

**Follow-ups**

- Two lane plans still carry the `companion` wire field, which the check's §3 removed and the contract at `970ae98` no longer
  has: the hub plan validates it at `:60` and `:63`, and the companion plan sends it at `:103`. The hub specialist works in
  another repository, and the check's hub section names only the dropped column. The message that returns its plan should say
  so.
- On Linux, `/proc/*/exe` for a game run under Wine or Proton points at the Wine loader, not at a file under `_forever_`, so
  game detection likely never matches (companion plan `:125`). That is `pr-tech-review`'s to check when the code exists.
- `hub:docs/DECISIONS.md:196`: D14 still calls the feed "the one public route besides ...". It has been wrong since D17 and has
  no dated note; it was stale before this change.

**What carried it**: The hub plan's heading "Rules where the contract is silent, decided here"
(`hub:...impl-backend-plan.md:92`). Reading each rule under it against contract §4 is how the one rule where the contract
is not silent, the role, surfaced.
