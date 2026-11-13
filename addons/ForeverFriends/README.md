# ForeverFriends

Records the character you play into SavedVariables, for the group's hub, and shows the group's
member list with `/ff`. It reads only: it never acts for you (D5), has no file or network access,
prints nothing at login, and shows no time about anyone.

## Install

The companion installs it (`docs/CONTRACT.md` §2). By hand for development: run
`tools/link.ps1 ForeverFriends` from the checkout (a directory junction into
`_forever_\Interface\AddOns`) and restart the client once. Without the companion's second addon
folder, `ForeverFriendsData`, the addon still records the character and `/ff` says there is no list yet.

## Use

`/ff` (or `/foreverfriends`) answers with one line of chat:

- `/ff`: how many members and characters the list holds.
- `/ff members`: the members' names.
- `/ff who <name>`: the member whose name, or whose character's name, starts with `<name>`, with
  their characters (class, level, `main`). The query is plain text; case is folded for ASCII only.
- `/ff status`: how many characters this account's file holds, and this session's character.

The addon records at login, on a level-up, a moment after the skill lines change, and at logout.
The client writes `WTF/Account/<ACCOUNT>/SavedVariables/ForeverFriends.lua` at logout and `/reload`.

## The data

The format of `ForeverFriendsDB` (what is written) and of `ForeverFriendsData` (what is read) is
`docs/CONTRACT.md` §1 and §2; it is not restated here. `ForeverFriendsDB.log` holds the last 50
lines (`/dump ForeverFriendsDB.log`).

## Verify in client

Tests cover the logic and a simulated client (`spec/support/wow.lua`); nothing here is known to
work in the real game until these are answered.

1. `/dump select(4, GetBuildInfo())` is 16001; `/dump GetRealmName()` (spelling, spaces).
2. `/dump UnitClass("player")` and `/dump UnitRace("player")`: the second values are the tokens;
   what are the two Skyborne race tokens? (The hub's table lacks them.)
3. `/dump C_Secrets and C_Secrets.ShouldUnitIdentityBeSecret("player")` in town, in combat and in
   an instance; `/run local n=UnitName("player") print(pcall(function() return (n.."")==n end))`;
   `/dump issecretvalue`.
4. `/run for i=1,C_SkillInfo.GetNumSkillLines() do local s=C_SkillInfo.GetSkillLineInfo(i) print(i,s.skillID,s.name,s.isHeader,s.rank,s.maxRank) end`:
   are the profession ids those of the hub's table, are there more than 40 lines, do professions
   come before weapon skills, and are the lines readable at login?
5. After `/reload`: `/dump ForeverFriendsDB.characters` has the §1 fields; the file changes at
   `/reload` and logout only. After a full relog: `/dump ForeverFriendsDB and #ForeverFriendsDB.log`
   (does the client load the file back, `docs/RESEARCH.md` §1.3?).
6. `/dump ForeverFriendsData and #ForeverFriendsData.members` after the companion wrote the list
   and the game was restarted, and after a change and a `/reload`.
7. `/ff`, `/ff members`, `/ff who <name>`, `/ff status`: one readable line each; does another
   addon own `/ff` (`/dump SlashCmdList.FOREVERFRIENDS` shows ours)?
8. After a level-up: does the log show the new level (is the event's first argument the new
   level)? How often `SKILL_LINES_CHANGED` fires at login and in combat. No Lua error or taint at
   login (`/console scriptErrors 1`).
