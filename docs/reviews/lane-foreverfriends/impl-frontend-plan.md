# ForeverFriends, lane A: the addon plan (impl-frontend, 2026-11-13)

Plan only, against `docs/CONTRACT.md` §1, §2, §5 and the director's plan §3, §4 "A", §5, §6.
## 1. Files, in build order

Each commit passes `make lint test`. Pure = no client globals, loaded with `loadfile` as `pack_spec.lua` does.

1. `addons/ForeverFriends/ForeverFriends.toc`, alone: `## Interface: 16001`, `## Title`, `## Notes`,
   `## Author: Forever Friends`, `## Version: 0.1.0`, `## OptionalDeps: ForeverFriendsData`,
   `## SavedVariables: ForeverFriendsDB`, then `Snapshot.lua`, `Roster.lua`, `Lines.lua`, `Core.lua`.
2. `Snapshot.lua` (pure), `spec/snapshot_spec.lua`, the stub's player and skill parts, `.luacheckrc` read_globals.
3. `Roster.lua` (pure) and `spec/roster_spec.lua`: reads and searches the untrusted member list.
4. `Lines.lua` (pure) and `spec/lines_spec.lua`: the only door from untrusted text to chat.
5. `Core.lua` (client-facing, thin), `spec/core_spec.lua`, the stub's `C_Secrets`; `.luacheckrc` globals
   `ForeverFriendsDB`, `SLASH_FOREVERFRIENDS1/2`, `SlashCmdList.FOREVERFRIENDS`, read-only `ForeverFriendsData`.
6. `README.md` with "Verify in client".

## 2. Client API calls

`UnitName("player")`, `UnitClass("player")` (2nd return), `UnitRace("player")` (2nd return), all secret when
identity is restricted (RESEARCH §6, opened); `UnitLevel("player")` (not secret per §6, probed anyway);
`GetRealmName()`, `GetServerTime()`; `C_Secrets.ShouldUnitIdentityBeSecret("player")` (§6);
`C_SkillInfo.GetNumSkillLines()` and `.GetSkillLineInfo(i)` (§6; **unverified:** index versus skill line id,
retail's takes an id: Verify 4); plus `date`, `CreateFrame`, `C_Timer.After`, `SlashCmdList`, `print`.
Not called: `issecretvalue` (the probe covers it without a global of unproven existence), `C_AddOns`,
`UnitGUID`, anything protected (D5).

## 3. Nil and secret, per field

`ns.usable(v, kind)` returns `v` or `nil`. It first runs a probe under `pcall`: `v .. ""` for
`"string"`, `v + 0` for `"number"`; only then checks `type` (non-empty string; finite number of at least 1).
The real secret string has `type` "string", the stub's is a table, and both raise on the probe, so `type()`
never decides first (the director's constraint). Every field below goes through it and a miss keeps
what the entry had: `name`, `class`, `race` (via `pcall(UnitName/UnitClass/UnitRace)`), `realm`,
`level` (`UnitLevel` or the level-up argument), `skills` (§4). `seen` is not bumped on a miss.

Identity gate (`Core.lua`): if `pcall(C_Secrets.ShouldUnitIdentityBeSecret, "player")` is true, the
three identity calls are not made; level and skills still are. If `C_Secrets` is nil or the call errors,
go on; the probe is the defence. It is asked at every refresh, not once (identity may be restricted
only in some situations), and "false" never replaces the probe.

`ns.buildEntry(old, read, now)`: the key is `name-realm` of this read, so a read without usable name
and realm changes nothing. An entry is created only when name, realm and class are known (this read or
the old entry). `seen` is bumped only on such a read. The key is opaque.
## 4. Skills

`ns.collectSkills(count, getLine)`, injected getter, pure. `count` is probed; each line is read under
`pcall`; a line is dropped when it is not a table, `isHeader` is true, or `skillID`, `rank` or `maxRank`
is unusable (never a guessed zero). A secret or missing `name` is stored `""`. Output
`{ id, name, rank, max }` in the client's order, **first 40**, a truncation logged once per session.
Returns `nil` (keep the old list) when the count is unusable or 0 (skills not loaded at login) or every
line was dropped.

## 5. Events (one frame, one handler table, as MarketScan)

- `ADDON_LOADED` ("ForeverFriends" only): `ns.normalise(ForeverFriendsDB)`. A loaded-back table is checked,
  not trusted: not a table or `version ~= 1` gives a fresh `{ version = 1, characters = {}, log = {} }`;
  `characters` not a table becomes `{}`, non-table entries are dropped, `log` becomes at most 50 strings.
  Works from nil (the client may not load the file back) and from a full table; other characters untouched.
- `PLAYER_LOGIN`: refresh all, log `login: Name-Realm level N`, nothing in chat.
- `PLAYER_LEVEL_UP`: refresh with `arg1` as level (unusable: `UnitLevel`), log `level up: ...`.
- `SKILL_LINES_CHANGED`: dirty flag and one pending `C_Timer.After(2, ...)` refreshing skills only. Not logged.
- `PLAYER_LOGOUT`: refresh all, log `logout`. The file is written after this event, so this read counts.
Failures (identity secret, skills unreadable, truncated) are logged once per session per kind, so they do not
flush the 50-line ring (`ns.pushLog(log, line, max)`).

## 6. The slash command

`/ff` and `/foreverfriends`; command lowercased, rest is the argument. Each call reads `_G.ForeverFriendsData`
afresh. Every output is one line of at most 240 bytes, prefixed `ForeverFriends:`; no time is ever shown
(`written` never is). `/ff`: `9 members, 14 characters. /ff members, /ff who <name>, /ff status.`
`/ff members`: `Anna, Ben, Chris and 3 more.` (cut at a member boundary). `/ff who <q>`:
`Anna: Thrallina (Shaman 23, main), Bankalt (Mage)`, several matches joined by `; ` then `and N more`;
no match: `nobody matches "<q>".` `/ff status`: `3 characters recorded, this one is Thrallina-Everlook.`
(second half omitted if unknown). `/ff who` without a name and an unknown subcommand: the usage line
`/ff, /ff members, /ff who <name>, /ff status.` No data: `no member list yet. The companion writes it once
your PC is linked.` Version not 1: `the member list is in a format this addon does not know.` Unreadable:
`the member list could not be read.` No `/ff log` (the ring is in `ForeverFriendsDB.log`; `/dump` shows it).
## 7. Reading `ForeverFriendsData` safely

`ns.readRoster(data)` runs under `pcall`, returns `members` or `nil, "missing"|"version"|"unreadable"`.
It never calls `load`, `loadstring` or `setmetatable`, never concatenates into code. Every access is
`rawget`; loops are `for i = 1, cap` stopping at the first nil (no `pairs`, which a hostile table could
make endless). Caps: 100 members, 40 characters each. A string field is accepted only as `type ==
"string"` of 1 to 64 bytes, else the member or character is skipped; `level` an integer 1 to 120 else
omitted; `main` true only when exactly `true`; unknown keys ignored; a member without characters stays.
`ns.matchMembers(members, q)`: the query is plain text, never a pattern; both sides are lowercased (ASCII
only, umlauts do not fold: accepted, documented) and compared by `string.sub` prefix; a match is a member
name or any of their character names. `Lines.clean(s)` is the one door to chat: bytes below 32 and 127
removed, every `|` doubled, so a hub member cannot print a colour code or hyperlink into the chat of
others. The contract's escaping protects the Lua file; this is a separate defence.
## 8. The simulated client (`spec/support/wow.lua`)

In `wow.reset()`: `wow.player = { name, class, race, level }` behind `_G.UnitName/UnitClass/UnitRace/
UnitLevel` (multi-returns as documented); `wow.skills` (`{ skillID, name, isHeader, rank, maxRank }`) behind
`_G.C_SkillInfo`; `wow.identityRestricted` behind `_G.C_Secrets.ShouldUnitIdentityBeSecret`; the addon's
globals cleared. **A test makes identity secret** two ways: `wow.restrictIdentity()` (predicate true, and
`name/class/race` become `wow.secret()`: the gate path), or `wow.player.class = wow.secret()` with the
predicate false (the probe path the real client may take). `wow.noSecrets = true` removes `C_Secrets`; any
skill field may hold `wow.secret()`. The stub's secret, a bare table with a raising metatable, raises on
concatenation by plain Lua rules.
## 9. Specs, by name

`snapshot_spec`: usable accepts string and number, rejects nil, secret, empty string, NaN, infinity, level
below 1, wrong types; buildEntry writes the §1 shape, keeps old fields on nil and secret, refuses an entry
without name, realm or class, no `seen` bump on a partial read; collectSkills drops headers, caps 40, drops
a line with a secret id or rank, returns nil for count 0, secret count, erroring getter, all dropped;
pushLog caps 50; normalise survives nil, wrong version, `characters` a string, an entry a number, a mixed `log`.
`roster_spec`: the contract §2 sample; version 2 or missing; data a number or `wow.secret()`; a hole, a string
member, a number name, 65-byte name; level 0, 1e9, "5", NaN; `main = "yes"`; 101 members; 41 characters; a
table whose `__index` raises; matching exact, prefix, upper case, queries `%`, `(`, `[a-`, via a character
name, no match. `lines_spec`: each §6 line; at most 240 bytes with 100 long members; `|cffff0000x|r` becomes
`||cffff0000x||r`; control characters gone; "and N more".
`core_spec`: login records the §1 entry exactly, from nil and from an existing DB (others kept); wrong-version
DB reset; nothing printed at login; level-up argument and fallback; 20 `SKILL_LINES_CHANGED` give one read
after one tick; logout refreshes skills; restricted identity records nothing, keeps old, logs once over three
events; secret class alone keeps the old class, updates the level; `C_Secrets` missing; a raising predicate;
skill count 0 at login keeps the list; 60 lines give 40; ring stays 50 after 100 events; every §6 row prints
one line; garbage `ForeverFriendsData` (number, string, `{}`, hostile table, secret, 10000 members) never
raises; the toc has `Interface: 16001`, `OptionalDeps`, `SavedVariables` and lists exactly the files that exist.
## 10. "Verify in client" (README; the session carries the lines into `docs/HANDOFF.md`)

1. `/dump select(4, GetBuildInfo())` is 16001; `/dump GetRealmName()` (spelling, spaces).
2. `/dump UnitClass("player")`, `/dump UnitRace("player")`: the tokens, and the two Skyborne tokens.
3. `/dump C_Secrets and C_Secrets.ShouldUnitIdentityBeSecret("player")` in town, in combat, in an instance;
   `/run local n=UnitName("player") print(pcall(function() return n.."" end))`; `/dump issecretvalue`.
4. `/run for i=1,C_SkillInfo.GetNumSkillLines() do local s=C_SkillInfo.GetSkillLineInfo(i) print(i,s.skillID,s.name,s.isHeader,s.rank,s.maxRank) end`;
   if the index form errors, `/dump C_SkillInfo.GetSkillLineInfo(171)`. Note the profession ids, the line
   count (over 40? professions before weapons?), readable at login?
5. After `/reload`: `/dump ForeverFriendsDB.characters` has the §1 fields; the file changes at `/reload`
   and logout only. After a full relog: `/dump ForeverFriendsDB and #ForeverFriendsDB.log` (loads back?).
6. `/dump ForeverFriendsData and #ForeverFriendsData.members` after a companion write and `/reload`.
7. `/ff`, `/ff members`, `/ff who <name>`, `/ff status`: one readable line each; `/dump SlashCmdList.FOREVERFRIENDS`
   and whether another addon owns `/ff`.
8. A level-up: the log shows the new level (is `arg1` the new level?). How often `SKILL_LINES_CHANGED`
   fires at login and in combat. No Lua error or taint at login (`/console scriptErrors 1`).
## 11. What I will not do

The "not in scope" list; also no `/ff log`, no pruning of characters, no folding beyond ASCII, no edit
outside `addons/`, `spec/support/wow.lua`, `.luacheckrc`, `.busted`.
## 12. Where the sketch and the contract did not survive

- Director §4: the predicate alone is not the secret check. It gates identity only; the probe stays for every
  field (`C_SkillInfo` is not documented as non-secret); asked at every refresh.
- Director "pure line formatting" gains `|` doubling and control-character removal (chat injection).
  `SKILL_LINES_CHANGED` is coalesced (it fires per weapon skill-up); the rest of the event list stands.
- §1: change "every non-header line of C_SkillInfo, at most 40" to "... in the client's order, the first
  40", because the cut was undefined and two parties could differ.
- §1: add "an empty read of the skill lines (a client not yet ready) keeps the old list", because erasing a
  good list at login would make the next sync replace professions with nothing.
- §1: add "the key is opaque; readers use the fields", because a realm can hold spaces.
- §2, addon bullet: add "and doubles `|` and drops control characters before it shows a string"
  (informational; no format changes, no other party affected).
- §1: a loaded-back DB of another `version` is reset, not merged; say so there if the owner agrees.
## 13. Open questions

1. `C_SkillInfo.GetSkillLineInfo`: index or id (Verify 4); if an id, I report and wait, not guess.
2. `/ff` as the name (the director's); another addon may own it (Verify 7).
3. Whether identity is ever restricted for "player": the gate path is built and tested but may never run.
