# The link contract: addon, companion, hub

Version 1 (2026-11-13). This file is the only home of the formats below; the hub
(`Atomtomate/wow-forever-hub`, private) points here instead of restating them. Three parties
build against it:

| Party | Where it runs | Code |
|---|---|---|
| **ForeverFriends**, the addon | inside the game | `addons/ForeverFriends/` (Lua) |
| **the companion** | on each member's PC, beside the game | `companion/` (Go) |
| **the hub** | the group's server | `wow-forever-hub`, `src/companion.js` and its routes |

```
 game                        member's PC                      hub
 ForeverFriends  --§1 file-->  companion  --§3 signed POST /api/companion/sync-->  characters, "in game"
 ForeverFriendsData <--§2 file--  companion  <--§3 response-------------------------  member list
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
            realm = "Everlook",             -- GetRealmName()
            class = "SHAMAN",               -- classFilename from UnitClass: the same in every locale
            race = "Orc",                   -- englishRaceName from UnitRace: the same in every locale
            level = 23,
            skills = {                      -- non-header lines of C_SkillInfo in the client's order, the first 40
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
  three parties cannot disagree about it. The localized `name` of a skill stays in this file
  for a person reading it; it is not sent (§3).
- A value the client returns as `nil` or as a secret value is not written; the field keeps what
  it had. An entry without `name`, `realm` and `class` is not created. An empty read of the
  skill lines (a client that is not ready yet) keeps the old list.
- The key is opaque: readers use the fields, never split the key.
- A table the client loads back whose `version` the addon does not know is replaced by an empty
  one; the characters come back as they are played.
- The companion reads `version`, and `characters` with the fields above. It ignores `log` and
  any key it does not know. When the same key is in the files of two accounts, the entry with
  the newer `seen` wins. It must not assume a file lists every character: the client may not
  load the file back at login (`docs/RESEARCH.md` §1.3), so a session can start from an empty
  table and write only the characters played in it.
- A file that does not parse (the client is in the middle of writing it) is read again at the
  next tick and never uploaded in part.

## 2. Companion to addon: the data addon

The companion creates and owns the folder `<game>/Interface/AddOns/ForeverFriendsData/` and
nothing in this repository lives there. ForeverFriends names it under `## OptionalDeps`, so the
client loads it first when it exists, and the addon works without it.

The companion writes the folder when it installs the addon, with a `Data.lua` of
`members = {}` and `written = 0`, so that one restart of the game is enough for both folders;
the addon treats an empty list as no list yet.

`ForeverFriendsData.toc`, whose `## Interface` line is copied from the `ForeverFriends.toc` the
companion carries, so that a client patch never marks one of the two out of date alone:

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
  escaped and every byte below 32, and byte 127, is written as `\ddd` with exactly three
  decimal digits (Lua reads up to three, so `\1` followed by a digit would become another
  byte); bytes from 128 up pass through unchanged (UTF-8 names). Numbers are written by the
  companion's own integer formatter. Nothing received from the hub is ever written outside
  such a literal.
- The file is replaced atomically (written beside it, then renamed) and only when its content
  would differ, `written` aside.
- The addon treats the table as untrusted: it checks the type of every field it reads, skips
  what does not fit, and never calls, loads or concatenates-and-runs anything from it. Before
  it shows a string it doubles every `|` and drops control characters, so a name cannot carry
  one of the game's chat codes.
- A folder that did not exist when the game started (the first install of either addon) is seen
  only after the game restarts. A changed `Data.lua` in a folder the game already knows is seen
  at the next login or `/reload` (verify in client).

## 3. Companion and hub

The companion talks to four routes under `<hub>/api/companion/`. All take and answer JSON. The
hub's address is an origin, `https://host[:port]`, with no path, query or fragment: the path a
request is signed with (§3.2) must be the path the hub sees. The companion refuses any other
scheme except `http` on a loopback host, which is how the hub is tested. The companion never
follows a redirect. Binary values (`public_key`, `X-FF-Signature`) are standard base64 with
padding (RFC 4648 §4).

A failure answer is always `{ "ok": false, "error": "<one sentence>" }` with a status. Any
status this section does not name, any redirect, and a 200 whose body is not JSON with
`"ok": true` count as "the hub is not reachable".

### 3.1 Linking a PC: the member approves it in the browser

The companion never sees a password and no person ever sees a secret. It makes a key pair, asks
the hub for a short code, and a member who is logged in on the hub approves that code.

1. The companion creates a new Ed25519 key pair for every link it starts and keeps the private
   key in its own config folder, readable by that user only. The private key never leaves the
   PC.
2. `POST /api/companion/link`, unsigned, body at most 4 KiB (else 413):

   ```json
   { "v": 1, "device_name": "JULIAN-PC", "public_key": "<base64 of the 32-byte public key>" }
   ```

   Answer, 200:

   ```json
   { "ok": true, "device": "3f9c2a71d05b88e4", "user_code": "WXYZ-BCDF",
     "expires_in": 600, "interval": 5 }
   ```

   `device` (16 lowercase hex) is this PC's id from now on. `user_code` is eight letters from
   `BCDFGHJKLMNPQRSTVWXZ`, shown as two groups of four. `device_name` is cut to 40 characters;
   the hub drops control, format and line-separator characters from it, and an empty result
   becomes `PC`. 400 for a malformed body, a key that is not 32
   bytes, or a key the hub already holds. 429 when the limit is reached: at most 5 link starts
   from one address in 15 minutes and at most 20 pending links on the hub (an expired pending
   link is deleted, not counted); the companion says to try again later.
3. The companion stores the device id, the key, the code and the expiry in its config, then
   shows the code and opens `<hub address>/companion/link?code=<user_code>` in the browser. It
   builds that address itself; the hub does not name a page for it to open. The member, logged
   in on the hub, sees the PC's name, the code and the line "Press Link only if this code is
   shown by the companion on your own PC", with a Link and a Decline button (a form of the
   hub's own, with its session and CSRF check). The hub links only a pending, unexpired device
   whose code matches, read in either case and with or without the dash or spaces; a code
   already used or expired gets a plain message; Decline deletes the pending device. A member
   who is not logged in comes back to the same code after logging in. A link that is not
   approved within `expires_in` seconds is forgotten.
4. Meanwhile the companion asks every `interval` seconds, signed as in §3.2:
   `POST /api/companion/link/status` with body `{ "v": 1 }`. Answer:
   `{ "ok": true, "linked": false }`, or `{ "ok": true, "linked": true, "member": "Anna" }`.
   401 means the link expired or was declined; the companion starts over when the member asks.
   Any answer other than 200, 400 and 401 is asked again at the same interval until the
   expiry. Closed and reopened before the expiry, the companion shows the same code and goes on asking;
   after it, it starts over. It registers its autostart only once `linked` is true.

A member can link several PCs. Each appears on their hub profile by name and can be removed
there, which makes the hub answer that device with 401 from then on.

`POST /api/companion/unlink` (signed, body `{ "v": 1 }`) removes the calling device, linked or
still pending; the companion's own "remove" uses it before it deletes its key. Answer
`{ "ok": true }`; 401 means the device is already gone, and the companion treats that as
done. `link/status` and `unlink` answer 400 `unsupported version` for another `v`,
as `sync` does.

### 3.2 Every other request is signed

Three headers:

| Header | Value |
|---|---|
| `X-FF-Device` | the device id from §3.1: 16 lowercase hex |
| `X-FF-Seq` | a positive integer, greater than every one the hub has accepted from this device, written in decimal digits only: no sign, no leading zero, at most 15 digits |
| `X-FF-Signature` | base64 of the 64-byte Ed25519 signature |

A missing or malformed header is answered 401.

The signature is over these bytes (UTF-8, `\n` is one line feed, no trailing one), where the
path is the request path without a query and the hash is over the body exactly as sent:

```
FF1\n<METHOD>\n<PATH>\n<device id>\n<seq>\n<lowercase hex SHA-256 of the body>
```

The hub reads the raw body (within the route's size limit, else 413 first), hashes exactly
those bytes, and checks in this order, parsing the JSON only afterwards: the device exists
(else 401; a pending device past its expiry does not exist), the signature verifies against
that device's public key (else 401), the sequence number is greater than the last one accepted
(else 409, `{ "ok": false, "error": "stale sequence", "seq": <last accepted> }`). The hub
accepts a number with one atomic update, so two requests with the same number cannot both
pass. A request that passes all three uses up its sequence number whatever the body turns out
to hold.

The companion sends one request at a time, stores the next number before it sends, and uses a
new number for every request, answered or not. On 409 it continues from the returned number
plus one and sends again once; a second 409 in a row counts as a failed request. There is no
clock in this: a PC whose time is wrong still links and syncs.

What this buys: the hub stores public keys only, so a copy of its database cannot act as any
device; a recorded request cannot be sent twice; and the key works for these routes only, never
for the site's pages.

Test vector, which the hub's and the companion's test suites both assert:

```
private key seed   bytes 0x00 to 0x1f
public key         A6EHv/POEL4dcN0Y50vAmWfk1jCbpQ1fHdyGZBJVMbg=
device, seq        0123456789abcdef, 7
method, path       POST, /api/companion/sync
body               {"v":1,"game_running":false}
body SHA-256       637ea0ebc551e160c01667d569224ec68d187f006cb574458cbddef68177bc92
signature          /ugHnkaZUd3KLobU7MuwmJfVZHMDdseFQ0WO5IjfJh0xvHvEWIB8dSWXhpVvPBcxaPCGbtg1srH1wV8UNAsUCw==
```

### 3.3 The sync request

`POST /api/companion/sync`, signed, from a linked device. A device that is not linked yet gets
401.

```json
{
  "v": 1,
  "game_running": true,
  "characters": [
    { "name": "Thrallina", "realm": "Everlook", "class": "SHAMAN", "race": "Orc", "level": 23,
      "skills": [ { "id": 171, "rank": 145, "max": 150 } ],
      "seen": 1763000000 }
  ],
  "outage": { "from": "2026-11-13T19:14:00Z", "to": "2026-11-13T19:31:00Z" },
  "roster_etag": "9f2c41d07a3b55e1"
}
```

| Field | Required | Meaning |
|---|---|---|
| `v` | yes | `1`. The hub answers any other value with 400 `unsupported version`. |
| `game_running` | yes | whether a process of this game installation is running right now |
| `characters` | no | sent when a SavedVariables file changed since the last sync the hub answered with 200, and always on the first sync after linking: the merged entries of every account on this PC, as §1 has them, without the skills' names |
| `outage` | no | sent once, on the first success after at least two consecutive failed requests: `from` is the first failure, `to` this request, both the companion's clock in UTC |
| `roster_etag` | no | the `roster_etag` of the member list the companion last wrote, sent only while that `Data.lua` still exists |

Answer, 200:

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
| `member` | the display name the device is linked to |
| `accepted` | how many of the uploaded characters the hub stored |
| `ignored` | at most 10 uploaded characters the hub did not store, each with a reason; `[]` when there are none |
| `roster_etag` | 16 lowercase hex naming the current member list; it changes exactly when the list does |
| `roster` | the member list, omitted when the request's `roster_etag` already names it |

`accepted` and `ignored` are present exactly when the request carried `characters`.

The member list holds every account on the hub, in order of name. `name` is the display name,
or the username when that is empty. A member's characters come main first, then by name.
`class` is the hub's name (`Shaman`); `level` is left out when the hub does not know it and
`main` when it is false. A member's name is at most 160 bytes, a character's at most 64, and a
member has at most 60 characters; a reader skips an entry outside these bounds and keeps the
rest, and refuses the list only when its shape is wrong. Real names, BattleTags and everything
else on a profile stay on the hub.

Other answers:

| Status | When | The companion |
|---|---|---|
| 400 | the body is not JSON, a required field is missing or of the wrong type, `characters` is not an array or holds more than 50 | marks the files it sent as uploaded, logs the error, keeps its schedule |
| 401 | unknown or not yet linked device, bad signature or header, or the device was removed on the profile | stops syncing and says this PC is no longer linked; linking again starts it |
| 409 | stale sequence (§3.2) | continues from the returned number and sends again |
| 413 | body over 128 KiB | as 400 |
| 429 | something in front of the hub asks for less; the hub itself sets no limit on a signed request | waits 300 s; neither a failed request nor a success, so it does not start or end an outage |
| 5xx, no answer, TLS failure, anything else | the hub is not reachable | counts a failed request, keeps the upload for the next try |

What does not make a request fail: a character that breaks a limit (a string over 64 bytes, a
name over 24 characters, a level outside 1 to 60, more than 40 skills) goes to `ignored` with a
reason; an ill-formed `outage` or `roster_etag` is dropped and the rest is processed.

The companion's side of the limits: it sends at most 50 characters, those with the newest
`seen` first, at most 40 skills each, and leaves out a character whose `name` or `realm` is
longer than 64 bytes. The hub stores an outage with its own receive time and drops one whose
`from` is after its `to` or that is longer than 31 days: the companion's clock is not trusted.

Timing: one sync every 60 s while the game runs, every 300 s while it does not, and one at once
when the game starts or exits; the companion looks for the game's process at least every 15 s.
After a failed request it tries again sooner, at most every 30 s. After the companion resumes
from sleep, failures in its first 60 s are not counted: a laptop's own network coming back is
not the hub being down.
The hub shows a member as in game while any of their devices' last sync with
`game_running: true` is younger than 150 s, and stops at once on a sync from that device that
says `false`.

Times: `seen` and `written` are unix seconds; `server_time` and `outage` are UTC as
`YYYY-MM-DDTHH:MM:SSZ`, the hub's one stored format.

## 4. What the hub does with an upload

Observable rules, so the addon's and the companion's tests can state them; how they are
implemented is the hub's (`wow-forever-hub`, `docs/DECISIONS.md` D17).

- A character is matched to the member's existing characters by name, ignoring case. Found:
  class, race, level and professions are replaced by the game's; when two of the member's
  characters match, the one created first is updated. Not found: it is added. The role, the
  note and which character is the main stay what the member chose on the site; when the
  matched character is the main, the profile's main class and race follow it, as they do on
  the site.
- `realm` is read and not stored, so two characters of one name on two realms are one
  character on the hub: of several entries with one name in one upload, the one with the
  newest `seen` is used and the others are ignored with the reason `same name`.
- A member holds at most 60 characters; a new one beyond that is ignored with the reason
  `too many characters`.
- A character missing from an upload is left alone. Nothing is ever deleted by a sync.
- `class` is mapped from the client's token; an unknown token means the character is ignored
  with a reason. `race` is mapped likewise; an unknown token, or one whose class the hub's
  race and class table does not allow, does not set a race: a new character gets none, a
  matched one keeps its own.
- A skill line is a profession when its `id` is in the table below; other lines are dropped. A
  `rank` outside 1 to 300 is stored without a rank.

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

Gold, play time, location, gear, guild notes, chat, anything about other players. Beside the
characters and the running flag, the hub learns the PC's name and its public key when it is
linked, and nothing else about the PC. The hub shows no "last seen" and no time of a sync (its
D10 and D15). Inside the game folder the companion
reads one file per account, `SavedVariables/ForeverFriends.lua`, and writes only inside
`Interface/AddOns/ForeverFriends` and `Interface/AddOns/ForeverFriendsData`. Outside it, it
keeps its own config folder (the key, the hub's address, its state and a small log), its own
program folder, and its autostart entry.
