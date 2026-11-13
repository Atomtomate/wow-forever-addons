**Verdict:** Heavier than its work in one place: the secret-value and untrusted-list defences are stacked two to four deep on the same values, and the extra layers are exercised by no spec (about 160 lines of code, tests and stub). The rest of the code (file split, handler table, `fit`/`REASONS` as data) is about right; about 15 lines of prose are told in two or three documents. Nine findings, ordered by lines saved times recurrence; the last two are small.

Method: every removal below was made in a scratch copy of `c890b92` and run with `busted` and `luacheck`; the result is stated per finding. Nothing in the checkout was changed.

### 1. A second secret defence for identity: the `C_Secrets` predicate gate
- **What** — `addons/ForeverFriends/Core.lua:31-43` (`identityRestricted`) and its branch at `Core.lua:90-94` decide, from a documentation flag, whether to call `UnitName`, `UnitClass` and `UnitRace` at all. `ns.usable` (`Snapshot.lua:13`) already rejects every secret those calls return, and the comment at `Core.lua:31-33` says so ("ns.usable is the defence either way"). `docs/RESEARCH.md` §6 says a documentation flag is "not proof of behaviour" and that whether the predicate is ever true for `"player"` is unknown, so the gate is never more accurate than the probe.
- **Why it recurs** — the tooltip idea in `docs/IDEAS.md` also reads unit names and will copy whichever pattern is here. The gate carries its own stub (`C_Secrets` in `spec/support/wow.lua`), lint name, `told` kind and "verify in client" item.
- **The smaller version** — call `readIdentity(read)` unconditionally. With the branch removed, 90 of 91 specs pass; the one failure is the assertion counting the "identity is secret" log line (`core_spec.lua:106-112`). Behaviour (nothing stored, old entry kept, level still updated) is identical, because the probe does it. If the owner wants the diagnostic, one `logOnce` in the fallback branch (`Core.lua:102-107`) says "name unreadable" without a predicate.
- **Net effect** — about -45 lines: `Core.lua` -17, `spec/support/wow.lua` -9, `core_spec.lua` -18 (`131-141` plus the count), one `.luacheckrc` name, a README item. One fewer state to hold in mind and one fewer unverified API.

### 2. Six `pcall` wrappers around client calls that are documented not to raise
- **What** — `Core.lua:46-58` (`UnitName`, `UnitClass`, `UnitRace`), `:66` (`GetNumSkillLines`), `:86` (`GetRealmName`), `:95` (`UnitLevel`) each repeat `local ok, x = pcall(f, ...) if ok then read.x = x end`. The written rule is about secret values being returned, not about calls raising; `docs/RESEARCH.md` §6 lists `UnitLevel` as not secret, and the value that comes back goes through `ns.usable` anyway, so the chain per field is call, `pcall`, probe under `pcall`, then `type()`. No spec makes any of these calls raise: replacing all six with direct calls leaves 91 of 91 passing.
- **Why it recurs** — each new reader (the planned addon list calls `C_AddOns`, the tooltip line `UnitName`) copies the four-line block; it is the shape that compounds.
- **The smaller version** — if the owner wants a belt for the unverified client (`MarketScan/Core.lua:85,106,137` wraps its documented-throttled calls, so there is precedent), one helper and six one-line calls:

  ```lua
  local function attempt(fn, ...)
      local ok, a, b = pcall(fn, ...)
      if ok then return a, b end
  end
  read.name = attempt(UnitName, "player")
  local _, class = attempt(UnitClass, "player")
  ```
  If not, direct calls and `ns.usable` alone. The `if not C_SkillInfo` guard (`Core.lua:63-65`) is a separate question (an absent namespace would raise on a direct call) and stays.
- **Net effect** — about -19 lines with direct calls, -15 with the helper; the next reader is one line, not four.

### 3. `ns.normalise` repairs a table only this addon writes
- **What** — `Snapshot.lua:108-133` rebuilds `characters` entry by entry and the whole `log`, on top of the version check that `docs/CONTRACT.md` §1 asks for. A version-1 table is this addon's own output (string keys, table entries, string log lines); the repair only matters for a hand-edited file. `MarketScan/Core.lua:165-167` does the same job in three `or {}` lines.
- **Why it recurs** — it does not recur by itself; it sets the size every later addon's load-back check is measured against. The ring cap is also implemented twice (`Snapshot.lua:101` and `:128`), and `pushLog` trims on the next push anyway.
- **The smaller version** — version check only; keep `db.characters = db.characters or {}` and the same for `log` (MarketScan's form) if insurance against a hand edit is wanted:

  ```lua
  function ns.normalise(db)
      if type(db) == "table" and db.version == 1 then return db end
      return { version = 1, characters = {}, log = {} }
  end
  ```
  Scratch copy: 89 of 91 pass; the two failures are exactly the two specs of the repair (`snapshot_spec.lua:157-176`). No spec reaches the repair through `Core`.
- **Net effect** — about -38 lines (-20 code, -18 tests); one rule ("a loaded-back table is checked") fewer to maintain next to the contract's one.

### 4. The companion's footprint and the never-crosses list are told in five places
- **What** — the footprint (reads one file per account, writes only into the two addon folders, keeps its own config, program copy and autostart entry) is in `docs/CONTRACT.md:383-387` (§5, named in `CLAUDE.md` as the one home), again in `CLAUDE.md:44-46` directly after it names §5 as owner, in `docs/DECISIONS.md:86-88` (D7), and the "no gold, play time, location, gear or chat" list is in `docs/DECISIONS.md:112-114` (D8) and §5. The copies already differ: only §5 has "guild notes"; `CLAUDE.md` says "program copy", §5 "program folder". The overlay on the fleet branch (`agents.local.md:78-81`) will be a fifth copy.
- **Why it recurs** — `docs/IDEAS.md` already lists "Companion installs MarketScan too", which changes the footprint; each such change is an edit to every copy. Link rationale has the same shape: D8 (`:126-138`) re-tells the mechanism §3.1 and §3.2 own, and §3.2's "What this buys" (`CONTRACT.md:221-223`) re-tells D8's reasons.
- **The smaller version** — §5 owns the footprint and the never list; `CLAUDE.md:44-46` keeps its first sentence and says "footprint: `docs/CONTRACT.md` §5"; D8's first bullet ends "Nothing else (`docs/CONTRACT.md` §5)"; the rationale lives in D8 only and §3.2 loses its last paragraph; the addon README drops `README.md:27` and `:33-34`, which restate §2's escaping rule and §1's nil/secret rule two lines after "it is not restated here".
- **Net effect** — about -15 lines now; the real gain is one place to edit when the footprint changes, and no copy left to contradict §5.

### 5. `cut` and the 240-byte cap go beyond "one line of chat"
- **What** — `Lines.lua:15-29` (`cut`: UTF-8 boundary scan, lead-byte drop, pipe-parity fix) plus its calls at `:107` and `:125`. `CLAUDE.md` asks for "one line of chat", which one `print` is however it wraps; no document gives a reason for 240. `fit` (`:32-42`) already keeps the 100-member `members` and `who` cases inside the cap.
- **Why it recurs** — it does not; it is one unit, but the pipe-parity logic is the subtlest code in the addon and nothing outside this file explains it.
- **The smaller version** — keep `fit` and `SUFFIX_RESERVE` (they are the visible "and N more" behaviour), delete `cut`. Scratch copy: 90 of 91 pass; the one failing assertion is `lines_spec.lua:83`, the case of a 160-byte member name beside a 64-pipe class, which, by my arithmetic, a 160-byte name without pipes does not reach (`Roster.lua:8` says names are at most 40 characters, and `fit`'s per-member budget caps the rest). A friend naming themselves 40 pipes gets one long wrapped line.
- **Net effect** — about -22 lines (-16 code, -6 test); no byte-level string surgery to maintain.

### 6. `.luacheckrc` lists every addon's globals twice
- **What** — `files["addons/*/spec/**"]` (`.luacheckrc:47-55`) repeats the four names each addon block already declares (`ForeverFriendsDB`, `ForeverFriendsData`, `SLASH_FOREVERFRIENDS1/2`, and MarketScan's), because luacheck applies `files["addons/ForeverFriends/**"]` to the spec files beneath it too.
- **Why it recurs** — every new addon adds its names to both places; this diff grew the list from one line to four.
- **The smaller version** — the spec block keeps `std = "+busted"`, `globals = { "_G" }` and the `122` ignore. Scratch copy: lint clean, 91 of 91 pass, and a stray `Bogus = 1` in a ForeverFriends spec is still reported as W111.
- **Net effect** — -4 lines now; an addon is declared in one place from here on.

### 7. `Roster.lua` goes past the rule the contract states
- **What** — `Roster.lua:81-90` wraps `read` in `pcall` ("Never raises"), but `read` only reads type-checked tables with `rawget`, takes `#` of strings and compares numbers: the `pcall` has nothing to catch, and its "unreadable" duplicates the reason `read` already returns at `Roster.lua:60-62`. Second, `Roster.lua:10,18-22` accepts a level only as an integer from 1 to 120; `docs/CONTRACT.md` §3.3 names no roster level bound (the hub caps uploads at 1 to 60), the companion writes numbers with an integer formatter (§2), and a number cannot carry a colour code, so `type(value) == "number"` meets "checks the type of every field it reads".
- **Why it recurs** — it does not; both are small. The `pcall` is the claim worth removing, because "survives a hostile table" is carried by `rawget` and the type checks, and a reader will credit it to the wrong line.
- **The smaller version** — `return read(data)` after the `nil` test; level is `type(value) == "number"`, and `MAX_LEVEL` goes. Scratch copy without the `pcall`: 91 of 91 pass, including the hostile-table and 10000-member specs. Dropping the range fails only `roster_spec.lua:75-88`, which shrinks to one assertion.
- **Net effect** — about -15 lines (-6 code, -9 test).

### 8. `name` and `realm` are validated three times per refresh
- **What** — `ns.usable(read.name/realm)` runs at `Core.lua:78` (`oldEntry`), `:102`, `Snapshot.lua:42` and again in the `IDENTITY_FIELDS` loop (`:50-52`, where `name` and `realm` are already known usable); `read.class` is probed at `:51` and `:58`. `oldEntry` exists only because the old entry is needed before `buildEntry` validates the key.
- **Why it recurs** — it does not; the next addon with a key built from a client value could copy it.
- **The smaller version** — `buildEntry(characters, read, now)` looks the old entry up itself after its own check, `IDENTITY_FIELDS` becomes `{ "class", "race" }` with `entry.name, entry.realm = name, realm` set directly, and `oldEntry` goes.
- **Net effect** — about -8 lines (`oldEntry` 6, two probes); one place decides what a usable name is. Modest.

### 9. Scaffolding nothing reaches (small)
- `Core.lua:13-18`: `db()` re-creates the table if `ForeverFriendsDB` is nil, but `ADDON_LOADED` for this addon (`onAddonLoaded`) always runs before login, events and `/ff`. Removing the guard passes 91 of 91; use the global directly (-6).
- `Lines.lua:45-49`: `detail` is a table that holds at most one entry before `table.concat`; `local facts = character.class and ns.clean(character.class) or ""` replaces it (-3).
- `spec/support/wow.lua:95,101`: `wow.skillCount` is read and never set by any spec (-1, and a state the stub has to explain).
- `Core.lua:84`: `withSkills` is false for one caller (level-up); always refreshing skills drops a boolean parameter and `(nil, true)` three times (-3). Optional.

**Follow-ups**
- `/home/user/wt-addons-fleet/.claude/agents.local.md:78-81` (not in this diff): the overlay restates the footprint a fifth time; reduce it to the pointer when it lands (see 4).

**What carried it** — the brief's "Already known" paragraph naming which defences the written rules require: it let me ask of each layer "once or several times" instead of "needed or not", and removing the predicate gate in a scratch copy (one assertion about its own log line failed, nothing about behaviour) settled finding 1.
