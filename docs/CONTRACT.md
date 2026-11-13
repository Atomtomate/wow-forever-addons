# The link contract: addon, companion, hub

Version 1 (2026-11-13). This file is the only home of the three formats below; the hub
(`Atomtomate/wow-forever-hub`, private) points here instead of restating them. Three parties
build against it:

| Party | Where it runs | Code |
|---|---|---|
| **ForeverFriends**, the addon | inside the game | `addons/ForeverFriends/` (Lua) |
| **the companion** | on each member's PC, beside the game | `companion/` (Go) |
| **the hub** | the group's server | `wow-forever-hub`, `src/companion.js` and its routes |

```
 game                        member's PC                      hub
 ForeverFriends  --§1 file-->  companion  --§3 POST /api/companion/sync-->  characters, "in game"
 ForeverFriendsData <--§2 file--  companion  <--§3 response------------------  member list
```

A change to a format is a change to all three parties. Adding an optional field keeps the
version at 1 and old readers ignore it; removing, renaming or re-typing a field bumps the
version in that section, and the reader refuses a version it does not know instead of guessing.
Why the link exists and what it must never carry: `docs/DECISIONS.md` D6 to D8.

## 1. Addon to companion: SavedVariables

The client writes `<game>/WTF/Account/<ACCOUNT>/SavedVariables/ForeverFriends.lua` at logout and
at `/reload`, never in between. One file per WoW account; it holds every character of that
account that logged in with the addon enabled.

```lua
ForeverFriendsDB = {
    version = 1,
    characters = {
        ["Thrallina-Everlook"] = {          -- key: name, "-", realm
            name = "Thrallina",
            realm = "Everlook",
            class = "SHAMAN",               -- classFilename from UnitClass: the same in every locale
            race = "Orc",                   -- englishRaceName from UnitRace: the same in every locale
            level = 23,
            skills = {                      -- every non-header line of C_SkillInfo, at most 40
                { id = 171, name = "Alchimie", rank = 145, max = 150 },
            },
            seen = 1763000000,              -- GetServerTime() when this entry was last refreshed
        },
    },
    log = { "2026-11-13 20:15:01 login: Thrallina-Everlook level 23" },   -- ring, at most 50 lines
}
```

- The addon records what the client returns and interprets nothing: `skills` carries weapon
  skills and languages too, and which lines are professions is the hub's table (§4), so the
  three parties cannot disagree about it.
- A value the client returns as `nil` or as a secret value is not written; the field keeps what
  it had. An entry without `name`, `realm` and `class` is not created.
- The companion reads `version`, and `characters` with the fields above. It ignores `log` and
  any key it does not know. It must not assume the file lists every character: the client may
  not load the file back at login (`docs/RESEARCH.md` §1.3), so a session can start from an
  empty table and write only the characters played in it.

## 2. Companion to addon: the data addon

The companion creates and owns the folder `<game>/Interface/AddOns/ForeverFriendsData/` and
nothing in this repository lives there. ForeverFriends names it under `## OptionalDeps`, so the
client loads it first when it exists, and the addon works without it.

`ForeverFriendsData.toc`:

```
## Interface: 16001
## Title: Forever Friends (member list)
## Notes: Written by the Forever Friends companion. Changes here are overwritten.
Data.lua
```

`Data.lua`:

```lua
ForeverFriendsData = {
    version = 1,
    written = 1763064000,                   -- unix seconds, the companion's clock
    members = {
        { name = "Anna", characters = {
            { name = "Thrallina", class = "Shaman", level = 23, main = true },
            { name = "Bankalt", class = "Mage" },
        } },
    },
}
```

- **Every string comes from people typing on the hub, and this file is executed by the game.**
  The companion writes a string only as a double-quoted Lua literal in which `\` and `"` are
  escaped and every byte below 32, and byte 127, is written as `\ddd`; bytes from 128 up pass
  through unchanged (UTF-8 names). Numbers are written by the companion's own integer
  formatter. Nothing received from the hub is ever written outside such a literal.
- The file is replaced atomically (written beside it, then renamed) and only when its content
  would differ, `written` aside.
- The addon treats the table as untrusted: it checks the type of every field it reads, skips
  what does not fit, and never calls, loads or concatenates-and-runs anything from it.
- The game reads the file at login and at `/reload`; a list written while the game runs is seen
  at the next of those.

## 3. Companion and hub: one request

```
POST <hub>/api/companion/sync
Authorization: Bearer <token>
Content-Type: application/json
```

The token is the member's, minted on their hub profile and shown there as one **link line**,
`<hub base URL>/companion#<token>`, which the member pastes into the companion once. The part
before `/companion#` is the hub's base URL, the part after `#` is the token (32 lowercase hex
characters). The companion refuses a base URL that is not `https://`, except for a loopback
host, which is how the hub is tested.

Request body:

```json
{
  "v": 1,
  "companion": "0.1.0",
  "game_running": true,
  "characters": [
    { "name": "Thrallina", "realm": "Everlook", "class": "SHAMAN", "race": "Orc", "level": 23,
      "skills": [ { "id": 171, "name": "Alchimie", "rank": 145, "max": 150 } ],
      "seen": 1763000000 }
  ],
  "outage": { "from": "2026-11-13T19:14:00Z", "to": "2026-11-13T19:31:00Z" },
  "roster_etag": "9f2c41d07a3b55e1"
}
```

| Field | Required | Meaning |
|---|---|---|
| `v` | yes | `1`. The hub answers any other value with 400 `unsupported version`. |
| `companion` | yes | the companion's version, at most 20 characters |
| `game_running` | yes | whether a process of this game installation is running right now |
| `characters` | no | sent only when a SavedVariables file changed since the last upload the hub accepted; the merged entries of every account on this PC, as §1 has them |
| `outage` | no | sent once, on the first success after at least two consecutive failed requests: `from` is the first failure, `to` this request, both the companion's clock in UTC |
| `roster_etag` | no | the `roster_etag` of the member list the companion last wrote |

Response, 200:

```json
{
  "ok": true,
  "member": "Anna",
  "server_time": "2026-11-13T20:00:00Z",
  "accepted": 1,
  "ignored": [ { "name": "Xx", "reason": "unknown class" } ],
  "roster_etag": "d41a07c2e9b3f610",
  "roster": { "members": [ { "name": "Anna", "characters": [
    { "name": "Thrallina", "class": "Shaman", "level": 23, "main": true } ] } ] }
}
```

| Field | Meaning |
|---|---|
| `member` | the display name the token belongs to, so the companion can say who it is linked as |
| `accepted` | how many of the uploaded characters the hub stored; absent when none were sent |
| `ignored` | at most 10 uploaded characters the hub did not store, each with a reason |
| `roster_etag` | names the current member list |
| `roster` | the member list of §2, omitted when the request's `roster_etag` already names it |

Other answers, each `{ "ok": false, "error": "<one sentence>" }`:

| Status | When | The companion |
|---|---|---|
| 400 | malformed body, unsupported `v` | logs it and keeps its schedule; it does not resend that upload |
| 401 | missing or unknown token | stops syncing and says the link was removed; a new link line starts it again |
| 413 | body over 64 KiB | as 400 |
| 429 | too many requests | waits 300 s |
| 5xx, no answer, TLS failure | the hub is not reachable | counts a failed request, keeps the upload for the next try |

Limits the hub enforces and the companion respects: body 64 KiB; 50 characters; 40 skills per
character; 64 bytes per string; level 1 to 60.

Timing: one request every 60 s while the game runs, every 300 s while it does not, and one at
once when the game starts or exits. The hub shows a member as in game while their last request
with `game_running: true` is younger than 150 s, and stops at once on a request that says
`false`.

Times: `seen` and `written` are unix seconds; `server_time` and `outage` are UTC as
`YYYY-MM-DDTHH:MM:SSZ`, the hub's one stored format.

## 4. What the hub does with an upload

Observable rules, so the addon's and the companion's tests can state them; how they are
implemented is the hub's (`wow-forever-hub`, `docs/DECISIONS.md` D17).

- A character is matched to the member's existing characters by name, ignoring case. Found:
  class, race, level and professions are replaced by the game's. Not found: it is added. The
  role, the note and which character is the main stay what the member chose on the site.
- A character missing from an upload is left alone. Nothing is ever deleted by a sync.
- `class` is mapped from the client's token; an unknown token means the character is ignored
  with a reason. `race` is mapped likewise; an unknown token leaves the race empty.
- A skill line is a profession when its `id` is in the table below; other lines are dropped.

| `class` token | hub | | `race` token | hub |
|---|---|---|---|---|
| `WARRIOR` | Warrior | | `Human` | Human |
| `PALADIN` | Paladin | | `Dwarf` | Dwarf |
| `HUNTER` | Hunter | | `NightElf` | Night Elf |
| `ROGUE` | Rogue | | `Gnome` | Gnome |
| `PRIEST` | Priest | | `Orc` | Orc |
| `SHAMAN` | Shaman | | `Scourge` | Undead |
| `MAGE` | Mage | | `Tauren` | Tauren |
| `WARLOCK` | Warlock | | `Troll` | Troll |
| `DRUID` | Druid | | the two Skyborne races | not known yet: verify in client |

| Skill line `id` | Profession | | Skill line `id` | Profession |
|---|---|---|---|---|
| 171 | Alchemy | | 165 | Leatherworking |
| 164 | Blacksmithing | | 186 | Mining |
| 333 | Enchanting | | 393 | Skinning |
| 202 | Engineering | | 197 | Tailoring |
| 182 | Herbalism | | | |

The class and race tokens and the skill line ids are the ones Classic and retail clients use;
that Forever uses the same is **not verified** until the first run in the client
(`addons/ForeverFriends/README.md`, "Verify in client").

## 5. What never crosses

Gold, play time, location, gear, guild notes, chat, anything about other players. The hub shows
no "last seen" and no time of a sync (its D10 and D15). The companion reads one file per
account, `SavedVariables/ForeverFriends.lua`, and writes only inside
`Interface/AddOns/ForeverFriends` and `Interface/AddOns/ForeverFriendsData`.
