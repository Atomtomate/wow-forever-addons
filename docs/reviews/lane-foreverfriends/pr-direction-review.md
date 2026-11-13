**Verdict:** Yes. The addon stays inside D6 to D8 and the contract. It builds nothing that `docs/IDEAS.md` marks next, later or parked: no addon list, no tooltip, no in-game presence, no character name beside "in game". Nothing a player sees nags, ranks, or shows a time about a member. Three small findings, all about documents: one contract line that the addon's `seen` rule no longer matches, one inventory row ahead of #4, and one "fact that does not move" that lists four of its source's five functions.

Paths are relative to `/home/user/wt-ff` at `c890b92` (diff `d7dc548...c890b92`).

### 1. The contract says `seen` moves when the entry is refreshed; the addon moves it only on a read that names the character in full

- **What**: §1 defines `seen` as "GetServerTime() when this entry was last refreshed". The addon keeps `seen` unchanged when a refresh writes level and skills through the session's remembered name while the client will not name the character (the reported departure in `Core.lua`). So an entry can be refreshed while its `seen` stays where it was.
- **Where**: `docs/CONTRACT.md:43`, against `addons/ForeverFriends/Snapshot.lua:40` and `:58-60` (`seen` only when `read.class` is usable) and `addons/ForeverFriends/Core.lua:102-107` (name and realm taken from `session`). `addons/ForeverFriends/spec/core_spec.lua:105-106` asserts it: the level changes to 24 and `seen` equals the value before. The three places where another party orders entries by `seen`: `docs/CONTRACT.md:61-62` (the companion's merge across accounts), `:312-313` (the companion's newest-50 cut) and `:342-343` (the hub's choice among entries with the same name, `src/companion.js:209-220` in the hub).
- **Why it matters**: `docs/CONTRACT.md` is the only home of what each field means, and the overlay's rule is that a format changes there before any party's code does. Today no reader reaches a different result, because every recorded session moves `seen` at least once, at the read that first named the character. But the line is now false. The next party to read `seen` (the companion in #4, the addon list PR) will build against "last refreshed".
- **Suggested resolution**: Change the doc. One comment on `:43`: "GetServerTime() at the last read that named the character in full (name, realm, class)". The other way is to move `seen` on every write, which is a code change in `Snapshot.lua`. The rule the code has is reasoned and tested, so the doc is the cheaper fix.

### 2. The root README lists `companion/` as part of the repository at a head that does not contain it

- **What**: The Layout table gains a row for `companion/` ("installs ForeverFriends, uploads characters, reports that the game runs"). No such directory exists at `c890b92`. The program and its other lines (`go.mod`, `embed.go`, the release workflow, the Go commit check) arrive with #4. `docs/IDEAS.md` likewise says "The transport now exists (the companion, D7)".
- **Where**: `README.md:27`; `docs/IDEAS.md:36`. Against the plan review's finding 4 as folded: the README's Go lines were assigned to #4. `CLAUDE.md:44` and `docs/CONTRACT.md:10` also name `companion/`. Those two state a rule and a party of the contract, not an inventory, so they can stand.
- **Why it matters**: #3 can sensibly merge alone. The addon installs by junction (`addons/ForeverFriends/README.md:9-11`), and its verify-in-client list does not need the companion, while #4 waits on its own checks on the PC. In that window, `main`'s README describes a directory that is not there, and the Go side of the README is split between two PRs. The plan's fold meant to avoid exactly that split.
- **Suggested resolution**: Change the doc. Move the `README.md:27` row into #4 with the other Go lines, and word `docs/IDEAS.md:36` as "the transport is decided (the companion, D7)" until #4 merges. The other option is to keep both and say in #3's body that #3 and #4 merge together.

### 3. The new fact about secret unit identity names four of the five functions its source names

- **What**: `CLAUDE.md` now says `UnitName`, `UnitClass`, `UnitRace` and `UnitGUID` are secret when unit identity is restricted. `docs/RESEARCH.md` §6, which it cites, also lists `UnitFullName`. `UnitFullName` stays allowed in `.luacheckrc`.
- **Where**: `CLAUDE.md:24-26`, against `docs/RESEARCH.md:181-183` and `.luacheckrc:15`.
- **Why it matters**: "Facts that do not move" is the list every later session holds code to. `UnitFullName` gives name and realm in one call, so it is the natural alternative to `UnitName` plus `GetRealmName`. The tooltip idea, marked next in `docs/IDEAS.md`, compares unit names. A session that takes the list as complete would store or compare that function's result unchecked.
- **Suggested resolution**: Change the doc: add `UnitFullName` to `CLAUDE.md:25`.

**Decisions**

- ForeverFriends first; a Go companion carries the data, installs the addon and checks the hub is reachable; the addon list comes in the next PR: recorded (D6, D7; the owner's).
- A PC is linked by browser approval and signs every request; its name and public key reach the hub: recorded (D8, contract §3.1, §3.2, §5).
- What crosses and what never does; no times shown; a sync never deletes; one companion links one member: recorded (D8, contract §5).
- Windows is the supported platform for version one, and D7 says what macOS and Linux lack: recorded (D7, last paragraph).
- The data addon `ForeverFriendsData` under `## OptionalDeps`, written empty at install; the escaping rule; the addon reads the list as untrusted, doubles `|` and drops control characters before showing it: recorded (contract §2).
- Skills: non-header lines in the client's order, the first 40; an empty read keeps the old list; the key is opaque; a table of an unknown version is reset: recorded (contract §1).
- Roster bounds of 160 bytes, 64 bytes and 60 characters, with an out-of-bounds entry skipped: recorded (contract §3.3).
- Unit identity may be secret, so the addon gates on `C_Secrets` and probes every value: recorded (`CLAUDE.md`, `docs/RESEARCH.md` §6, `Snapshot.lua:9-12`); see finding 3.
- `/ff status`, ASCII-only case folding, nothing printed at login, one line per command and no time in any line: recorded (`addons/ForeverFriends/README.md`).
- `seen` moves only on a read that names the character in full: silent in the contract, which says otherwise (finding 1).
- Level and skills are written to this session's character while the client will not name it: recorded in code (`Core.lua:105`) and in the author's report. Within §1, so no finding beyond finding 1.
- No skill read on a level-up: recorded only in the author's report. Within D6, because `SKILL_LINES_CHANGED` re-reads skills; no finding.
- A skill line with any unusable id, rank or max is dropped, and the shorter list replaces the old one: silent. §1 covers only the empty read (see Follow-ups).
- At most 100 members read, and a roster level of 1 to 120 accepted (`Roster.lua:6`, `:10`): silent. No group of 5 to 15 reaches either bound; no finding.
- A chat line of at most 240 bytes, ending "and N more" (`Lines.lua:6-7`): silent beyond the README's "one line". Cheap to change; no finding.
- `## Version: 0.1.0` in the toc: the rule that ties the tag to it is #4's, as already assigned.

**Follow-ups**

- `CLAUDE.md:36-37` sends the verify-in-client lines to "the addon's `docs/` note". Both addons keep them in their README (`addons/MarketScan/README.md:40`), so the line was stale before this change.
- `docs/HANDOFF.md` still says draft PR #1 is in flight (stale before this change). It carries none of ForeverFriends' eight verify-in-client items, which `CLAUDE.md:36-37` sends there. The close writes it, and should do so on #3's branch if #3 may merge alone.
- For `pr-tech-review`: `Snapshot.lua:68-95` together with `docs/CONTRACT.md:334-335` ("professions are replaced by the game's"). On a read where only a profession line has a secret rank, that line is dropped, the shorter list replaces the old one, and the next sync clears that profession on the hub. Whether skill fields are ever secret is verify item 4 in `addons/ForeverFriends/README.md`.

**What carried it**: The brief's list of the author's reported departures. Holding the session-memory departure against contract §1 field by field led to the `seen` comment, and `core_spec.lua:106` confirmed that the code means it.
