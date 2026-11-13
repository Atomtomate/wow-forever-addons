**Verdict:** The cut is sound. The three programs meet only at `docs/CONTRACT.md` (`970ae98`), each one's pure core takes its clock and inputs from the caller, and each entry point is thin. The addon plan is right as it stands. Six placements need fixing before code: five in the companion (one is a Go test import cycle, one gives a contract file a second writer) and one in the hub (character writes placed in a devices module).

Plan files (the `file:line` below): **C** = `/home/user/wt-companion/docs/reviews/lane-companion/impl-backend-plan.md`, **H** = `/home/user/wt-hub-companion/docs/reviews/lane-m-companion/impl-backend-plan.md`, **A** = `/home/user/wt-ff/docs/reviews/lane-foreverfriends/impl-frontend-plan.md`. Held against both overlays' "The standard the architecture reviewer holds code to" and against the existing `src/*.js`, `addons/MarketScan/`, `spec/support/wow.lua`. Neither repository has a layering script, so I read the imports. I checked the two Go resolution rules below in a scratch module on Go 1.24.7, the version the plan names.

### 1. `hubfake` has no stated import direction, and the obvious one is a test cycle that also makes it check `hubapi` with `hubapi`'s own code

- **C:21, C:23–24, C:130.** The fake hub "verifies signatures and sequence per §3.2" and is "used by `hubapi` ... tests". The import rule lists every package except `hubfake`.
- **The crossing:** to verify a signature, `hubfake` needs the signing string. The only one in the plan is `hubapi`'s `Sign`, so the natural code is `hubfake → hubapi`. But `hubapi`'s own tests are `hubapi → hubfake`. Go refuses that pair for an in-package test: `imports .../hubfake from hubapi_test.go; imports .../hubapi from hubfake.go: import cycle not allowed in test` (reproduced). An external `hubapi_test` package would compile, but then the fake checks `hubapi` with `hubapi`'s own string. A wrong `<PATH>` or a wrong hash case would pass every `httptest` row, `linkflow`'s and `syncer`'s included. Only `TestSignVector` would still catch it.
- **The rule:** test support depends on the contract, not on the package it exists to check. A fake that imports the code under test points the wrong way. The hub plan already does it the right way: its fake companion (H:125, `test/companion-client.js`) "signs raw strings" itself.
- **The fix:** `hubfake` builds the §3.2 string and verifies with `crypto/ed25519` itself, importing only the standard library and `wire`, and the import rule at C:23 names it. That adds about fifteen lines.

### 2. The data addon folder gets two writers: `dataaddon` owns it, and `install` writes its first files

- **C:10, C:14, C:37, C:112–113.** `dataaddon` owns "`Render`, the §2 escaper, `Write` ..., the `.toc`". `install` also owns "the empty `ForeverFriendsData`" and writes a `.toc` (Interface copied from the embedded toc) and a `Data.lua` of `members = {}`, `written = 0`. C:37 says it outright: "`dataaddon` (+ `install` for the empty first file)".
- **The crossing:** `install` can only render the §2 `.toc` and `Data.lua` in one of two ways:
  - it imports `dataaddon`, a leaf-to-leaf edge that C:23's rule forbids ("import only the standard library (and `wire`)");
  - it carries a second renderer of the §2 format, its `## Title`/`## Notes` lines and its `version = 1` table.

  Separately, `dataaddon` "owns the `.toc`" but has no input that carries the Interface value. That value comes from the embedded `ForeverFriends.toc`, which only `install`'s toc reader parses.
- **The rule:** contract §2: "The companion creates and owns the folder". Inside the companion, the plan's own table makes `dataaddon` that owner. A version bump or a Title change would otherwise have to land in two packages, and `Write`'s "only when content differs" check would be comparing against a file that some other code rendered.
- **The fix:** `install` creates no data file. Right after `install.Install`, `app` calls `dataaddon.Write` with an empty roster and `written = 0`, and passes in the Interface value it read once through `install`'s toc reader. This removes a renderer and adds one parameter.

### 3. The hub's sync writes characters from `devices.js`, which must import a transaction-less piece of `characters.js` to do it

- **H:11–14, H:65, H:102.** `src/devices.js` ("queries") gets `applyUpload`, which inserts and updates `characters` rows with `from_game=1`, and `loadRoster`, which reads `users` and `characters`. To copy a main character to the profile, `applyUpload` calls `applyMain(db, userId, c)`, extracted from `setMain` "without its own transaction".
- **The crossing:** `devices.js` → `characters.js` `applyMain`. `applyMain` has to be exported for that call, and it is safe only inside a caller's `BEGIN`. That puts the `characters` table's write path in two modules: `saveCharacter`/`setMain` in `src/characters.js:64–93`, and `applyUpload` in `devices.js`.
- **The rule:** hub `CLAUDE.md`, "Where things are": `src/characters.js` is "characters: parsing / validation, main-character copy". Its header says it holds "the few queries that need a transaction". A module that imports another concern's internals to run that concern's use case has taken the use case over. This is the first writer outside the site form, and the next game-link lane (the addon list, D17 "the next pull request") will copy wherever it sits.
- **The fix:** `applyUpload` (applying `planUpload`'s ops to one member's characters) moves beside `saveCharacter` in `characters.js`, and `applyMain` stays private there. `devices.js` keeps the `devices` and `outages` tables. `loadRoster` goes with the other users-and-characters reads (`listMembers` in `src/app.js:116`, or `characters.js`). This is a move, not new code.

### 4. `instance` is given the stop request with only a path as input, but the request is a 45-second wait

- **C:18, C:24, C:98–99.** The row "one-instance lock and stop request (§6) | lock file path". §6: "A setup run creates `stop.request`, polls the lock for up to 45 s ... setup deletes the file after taking the lock". C:24: "none reaches for the network, clock".
- **The crossing:** whichever package holds the 45 s poll needs a clock or a sleep. The table gives the stop request to `instance` and gives `instance` no clock. Written as the row reads, `instance` calls `time.Sleep` on the real clock, and `TestStopRequestReleases` waits in real time.
- **The rule:** addons overlay: packages under `companion/internal/` "do not reach for ... the clock ... except through what the caller passes in". `linkflow` (C:12) already shows the way: it takes `clock, sleep func`.
- **The fix:** `instance` exposes only non-blocking calls (`RequestStop`, `TryAcquire`), and the 45 s loop sits in `app` on `Env`'s clock. Or `instance` takes a `sleep func`. Either way the plan should say which. One parameter or one loop.

### 5. The program self-copy has no owner, and it reaches for two things no injected seam provides

- **C:113–114, C:19–20.** "Companion program copy: skip when `os.Executable()` is already the installed path; otherwise stop the old instance, write `.exe.new`, rename over; if the installed copy answers `version` with a higher number, ask before downgrading". This sits under "Install rules (`install.Install(...)`)", but `install`'s row neither owns it nor takes its inputs.
- **The crossing:**
  - Placed in `install`, it needs `instance` (a leaf-to-leaf import the rule forbids), `os.Executable()`, and `os/exec` to run the installed copy.
  - Placed in `app`, it calls `os.Executable` and `exec.Command` directly, because `Env`'s platform functions (C:19) offer only `SpawnDetached` (detached, no output) and no executable path.

  Either way, a package spawns a process and reads its own identity outside what its caller passed in, and `TestSetupNonInteractiveLinksInstalls` cannot fake "already installed" or "installed copy is newer".
- **The rule:** as in finding 4, plus C:20's own design: `app` reaches the OS only through `Env`, whose funcs `platform` provides.
- **The fix:** the step belongs to `app`, which already imports `instance`. Add `Executable` and an installed-version query to `Env`, backed by `platform`. The version can also come from a value recorded in `config` at copy time, which avoids running a second binary. That is two func fields.

### 6. The root `embed.go`'s test cannot reach the toc reader it needs

- **C:25–26.** "Root `embed.go` (package `foreverfriends`) ...; its test asserts every file the `.toc` lists is embedded."
- **The crossing:** the root package → `companion/internal/install` (the only `.toc` reader, C:14). Go refuses it: `use of internal package .../companion/internal/install not allowed` (reproduced). The module root is outside `companion/`, the tree that may import `companion/internal/...`. As placed, the test needs a second `.toc` parser.
- **The rule:** Go's `internal` resolution rule, plus C:14's own choice of `install` as the one toc reader.
- **The fix:** move the test into `install`'s (or `app`'s) tests. Those may import the root package's `Addon` (reproduced: compiles). No code is added.

**Where the plans are right.**
- **Hub:**
  - `src/companion.js` is pure: no Express, no db, `now` and `randomInt` passed in, `node:crypto` used only for computation. It follows `presence.js` and `characters.js`.
  - It imports only `gamedata.js`, `characters.js` (`formatProfessions`) and `presence.js`, all of them pure.
  - The token tables sit in the module that speaks the contract. That is right as long as their values are tested against `src/gamedata.js`'s sets, which H:129 does.
  - `devices.js` → `companion.js` points downward.
  - The schema stays in `db.js` with an additions loop.
  - The download URL is a setting.
  - The router lives in `app.js`, mounted above the session middleware in the public position, with `signed` as route-layer glue like `requireAuth`.
- **Addon:** it follows the house exactly (`addons/MarketScan/Pack.lua` + `Core.lua`).
  - `Snapshot`, `Roster` and `Lines` take values in (`buildEntry(old, read, now)`, `collectSkills(count, getLine)`, `readRoster(data)`).
  - `Core` alone touches client globals and `_G.ForeverFriendsData`.
  - The TOC loads the pure files first.
  - `Lines` is the single door to chat. It is fine as a table on `ns`, which the `.luacheckrc` list implies, since that list names no `Lines` global.
  - The shared stub grows only as tooling.
- **Companion:**
  - Leaves import only the standard library and `wire`. The limits have one home, in `wire`.
  - One package sends.
  - `savedvars` reads through `fs.FS`.
  - `gamefind` is pure on injected lists.
  - `platform` is the one home of the registry, process and browser calls, wrapped by `app` in `Env`.
  - `main` is a single line.
  - `instance`'s own `_windows.go`/`_other.go` pair meets the overlay's "behind one small function each".
  - Using a root path as the seam for disk writes is a fair reading of the standard, because Go's `io/fs` is read-only.

**Follow-ups**
- `/home/user/wt-addons-fleet/.claude/agents.local.md` (Go bullet): say whether a root path counts as "what the caller passes in" for the file system. The plan reads it that way (C:24–25) and I accepted that, but the next reviewer should not have to make the same call again.

**What carried it:** the definition's "what a module had to import", applied by reading each row of C's "Takes in" column against the prose that uses that package. The moment was the `install` row (C:14) read against C:112–113's empty data addon. That gap pointed to findings 2, 4 and 5, which share its shape.
