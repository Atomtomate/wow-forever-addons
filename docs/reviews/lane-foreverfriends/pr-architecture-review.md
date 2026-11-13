**Verdict:** The structure is sound and follows the house. Three pure files take their inputs and their clock from the caller, `Core.lua` alone touches the client, the dependencies point one way, and both departures from the plan are placed well. The flaw is one split concern: the character's identity is half in `Core.lua` and half in `Snapshot.lua`. `Core.lua` forms the §1 key a second time, and it rewrites the pure builder's input, so the builder now answers its own rule through a proxy. That gives two findings, and one signature change fixes both.

Range `d7dc548...c890b92`, held against `CLAUDE.md` ("Working agreement", "Conventions") and the overlay's Lua standard, with `addons/MarketScan/` as the house. There is no layering script, so I measured instead:
- `luacheck --no-config --std lua51` gives 0 warnings on `Snapshot.lua`, `Roster.lua` and `Lines.lua`. On `Core.lua` it names every client global.
- All three pure files load and answer in bare `lua5.1` without `spec/support/wow.lua`.
- Both findings were reproduced with a script in the scratchpad. The checkout was not changed.

### 1. `addons/ForeverFriends/Core.lua:79` forms the §1 key itself to find the old entry, a second home of the key format next to `addons/ForeverFriends/Snapshot.lua:61`

- **The crossing:** `oldEntry` (`Core.lua:77-80`) does its own `ns.usable` check and indexes `db().characters[name .. "-" .. realm]`. `ns.buildEntry` (`Snapshot.lua:41-61`) then makes the same check and returns `name .. "-" .. realm` as the key that `Core.lua:120` writes under. So the read path and the write path each spell the key their own way.
- **Why it matters:** in a scratch copy I changed only `Snapshot.lua:61` to drop spaces from the realm, which the plan names as a real case. The realm was "Test Realm". The lookup in `Core.lua:79` no longer found the old entry. The next `PLAYER_LEVEL_UP` (`refresh(level, false)`, no skills read) wrote the entry with `skills = nil`. A key change made in the file that owns the key silently breaks the merge in the file that does not.
- **The rule:** `docs/CONTRACT.md` §1 gives the key one definition ("key: name, "-", realm"), and the addon should spell it in one place. `CLAUDE.md`, "Working agreement": "Pure logic ... lives in files with no client globals ... Client-facing code is thin". Forming a key and finding an old entry are pure logic, and `Core.lua` holds both.
- **The fix:** `ns.buildEntry` takes the `characters` table instead of `old` and looks up its own old entry under the key it builds. Or `Snapshot.lua` exports one `ns.key(name, realm)` that both callers use. This changes the signature and adds a line in `Snapshot.lua`, and `oldEntry` leaves `Core.lua`.

### 2. `addons/ForeverFriends/Core.lua:102-107` writes the session's name and realm into `read` before the pure builder sees it, so `Snapshot.lua:58` decides "this read named the character" by checking whether the class is usable

- **The crossing:** `refresh` replaces `read.name` and `read.realm` with `session.name` and `session.realm` whenever the client will not give them. It then calls `ns.buildEntry(oldEntry(read), read, GetServerTime())`. Inside, `buildEntry` can no longer tell a name this read supplied from one `Core.lua` remembered.
- **What that does to the builder's rule:** the docstring at `Snapshot.lua:37-40` says "`seen` moves only when this read named the character completely". `Snapshot.lua:58` can only check `read.class`. That proxy is right only because `Core.lua` substitutes name and realm and never class, and neither file says so. Reproduced at this head: after login, `wow.player.name = wow.secret()` with the predicate false and the class readable, then `PLAYER_LOGOUT`. `seen` moved from 1700000000 to 1700000100 on a read that named nobody.
  - Whether `seen` *should* move there is not mine to judge (`pr-tech-review`'s).
  - The structural fault is that the pure part's rule now depends on an unstated habit of its caller. A later change to what `Core.lua` remembers, the class for instance, would move `seen` on reads that name nothing, and no test of `Snapshot.lua` alone could see it.
- **The rule:** "how a fact is obtained". The pure part needs the fact "which fields did this read supply". It cannot get that from its input, because the client-facing caller has overwritten it, so it asks a different question. `CLAUDE.md`'s pure/thin split puts the fallback decision in the pure file.
- **The fix:** keep the `session` table in `Core.lua`, where state across events belongs, and pass it to `ns.buildEntry` as its own argument. The builder then decides both the fallback key and `seen` from facts it can see. `Core.lua` records `session.name, session.realm = entry.name, entry.realm` after a successful build. This adds a parameter and about four lines to `Snapshot.lua`, and removes the substitution and its `ns.usable` calls from `Core.lua`. Together with finding 1 it is one signature change: `buildEntry(characters, read, now, remembered)`.

**Where the structure is right**
- **Dependency direction.**
  - `Core → Snapshot` (`buildEntry`, `collectSkills`, `normalise`, `pushLog`, `usable`, `MAX_SKILLS`), and `Core → Lines → Roster` (`readRoster`, `countCharacters`, `matchMembers`).
  - No pure file uses anything `Core.lua` defines. `Snapshot` and `Roster` do not know each other. There is no cycle.
  - The TOC loads the pure files first, as `MarketScan.toc` does.
  - `lines_spec.lua` loading `Roster` and `Lines` together is the honest statement of the one pure-to-pure edge.
- **The clock and client functions come from the caller.**
  - `GetServerTime()` arrives as `now`. `C_SkillInfo.GetSkillLineInfo` arrives as `getLine`.
  - The `date` prefix of a log line stays in `Core.lua:21`, and `ns.pushLog` takes a finished line. That is cleaner than the house, where `MarketScan/Core.lua:16-25` keeps the ring next to the clock.
  - `Lines.lua` touches no time at all and never reads `written`.
- **`ns.respond` in `Lines.lua` (departure one) is the right home.**
  - It is pure: the player's text, the raw data and `{ count, key }` go in, one line comes out.
  - `Core.lua:190-192` reads the two globals and prints, and does nothing more.
  - `MarketScan` dispatches in `Core.lua`. Its formatting needs `date`, and its dispatch reads `MarketScanDB` directly. This addon has neither need, so pure dispatch is the stricter reading of the same rule.
  - The next addon will copy this placement, and it is a good one to copy.
- **The `session` table in `Core.lua` (departure two)** is in the right file. Only `Core.lua` lives across events, and a pure file holds no state. What sits on the wrong side of the boundary is the substitution it feeds, which is finding 2.
- **One reader of the untrusted table.** Only `Roster.lua` looks inside `ForeverFriendsData`. `Core.lua` passes the global through untouched. All text reaches chat through `ns.respond`, and every string from the member list goes through `ns.clean`.
- **No second home for a contract table.**
  - §4's class, race and profession tables are absent from the addon. `Snapshot.lua` records the raw tokens and every skill line, as §1 asks.
  - §1's 40 and 50 have one home in the addon, `Snapshot.lua:6-7`. `MarketScan`'s own `MAX_LOG` is separate by the independence rule.
- **No addon reaches into another.** The TOC lists only its own files. The one edge between addons is `## OptionalDeps: ForeverFriendsData`, the contract's §2 channel.
- **The shared simulated client grows only as a client.**
  - It gains `Unit*`, `C_SkillInfo` and `C_Secrets` stubs shaped by the client's documentation.
  - `wow.restrictIdentity()` and the raising predicate are client states. `wow.skill()` is a client return shape.
  - It contains no ForeverFriends logic. The resets of the addon's globals follow the existing `MarketScanDB` line.

**Follow-ups**
- `.luacheckrc`:
  - The top-level `read_globals` apply to every file. A pure file could call `UnitName` or `C_AuctionHouse` and still pass the lint. This PR adds five client names to that shared list.
  - Because `.busted` loads `spec/support/wow.lua` for every spec, the pure specs also run with the stubs in `_G`, so "busted loads it alone" (`Snapshot.lua:2`) is proven by nothing.
  - A per-file `std = "lua51"` block for `addons/ForeverFriends/{Snapshot,Roster,Lines}.lua` and `addons/MarketScan/Pack.lua` would make the pure-file rule a lint failure. It would be the Lua side of the layering check the overlay says this repository lacks.
- `docs/CONTRACT.md` §2 (`pr-direction-review`'s):
  - `Roster.lua:6-10` enforces bounds whose only home is §3.3: 160, 64, 60. Its comment says "the hub's own cap".
  - Two of the bounds, 100 members and level 120, have no home anywhere.
  - `Roster.lua:31` bounds `class` with the character-name limit, which the contract does not give.
  - §2, the addon's own interface, states none of these.

**What carried it:** the definition's "how a fact is obtained: trace each dependency, helper or accessor a changed line uses back to what it actually consults". The moment was following `oldEntry`'s `ns.usable` and index at `Core.lua:78-79` into `ns.buildEntry`. The same check and the same key were already there at `Snapshot.lua:42-61`. That trace also exposed the substitution right above it at `Core.lua:102-107`, which is finding 2.
