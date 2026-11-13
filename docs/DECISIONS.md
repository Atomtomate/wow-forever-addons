# Decisions

Append-only. A decision changes by a new entry, not an edit.

## D1 — One repository for all the group's addons (2026-11-09)

`addons/<Name>/` each with its own TOC and tests; lint, CI, the simulated client and the
feedback-loop scripts are shared. One place for the facts about the client, one junction
script, one CLAUDE.md. An addon moves to its own repository only if it is published on its own.

## D2 — Forever is treated as a retail-lineage client (2026-11-09)

Interface `16001`, API family of retail 12.1.5, modern auction house (`C_AuctionHouse`),
secret values. Verified against Blizzard's Forever UI source and two live Forever addons
(`docs/RESEARCH.md` §0, §1.2). Classic Era knowledge is background only. Revisit if the client
proves otherwise on day one (`docs/HANDOFF.md`).

## D3 — Three testing tiers, honest about their reach (2026-11-09)

1. **busted** on pure-logic files (no client globals), and on client-facing code through a
   small home-grown simulated client (`spec/support/wow.lua`) that models frames, events,
   timers, SavedVariables and secret values. Ours, not a third-party mock: it stays as small as
   what the addons need. Runs in CI and before every commit.
2. **A headless UI simulator** (Osso/wow-ui-sim has a Forever profile) for load errors and
   frame trees, once it proves usable; pinned version, throwaway container, no secrets.
3. **The real client** for behaviour: the auction house exists nowhere else. Every claim a
   test cannot prove is listed as "verify in client" with the exact line to run.

## D4 — The feedback loop is SavedVariables plus Remote Control (2026-11-09)

The game loads the checkout through a directory junction; after a `/reload` a script copies the
addon's SavedVariables file into `data/` (git-ignored) and parses it into a summary; the
assistant reads it through Remote Control on the owner's PC. Lua errors come the same way via
BugSack's SavedVariables. No network from the addon, no desktop daemon, no cloud upload of
game data. Until the client loads SavedVariables back reliably, addons are designed so that a
fresh file each session is acceptable (export-only), and read-back is a later step.

## D5 — Read, never act (2026-11-09)

No addon here calls a protected auction-house function or anything that spends gold, moves
items, casts or moves the character. Market data is read with `ReplicateItems` (one full scan,
rate-limited by the client) or browse queries within the documented 100 calls per minute, and
only while the auction house window is open and the player pressed a button. This keeps the
addons inside Blizzard's addon rules and inside the group's tone (the hub's D10).

## D6 — The first addon the group installs is ForeverFriends, the link to the hub (2026-11-13)

The owner changed the order: "the first addon should be a custom one, that connects the players
with the website", and it "should sync players registered on the website, their ingame chars,
their online status and, maybe, addon list". MarketScan is merged (PR #1) and stays the second
addon; it has not run in the client yet.

ForeverFriends does two things in the game. It records the character being played (name, realm,
class, race, level, skill lines) into SavedVariables at login, on a level-up, when skills change
and at logout. And it shows the hub's member list, who is whose character, behind `/ff`. It
reads its own character and nothing else, calls no protected function and automates nothing
(D5). The addon list of each member comes in the pull request after version one (the owner's
choice); the formats in `docs/CONTRACT.md` leave room for it and version one does not collect it.

## D7 — A companion program on each member's PC carries the data (2026-11-13)

An addon cannot reach the network, and the client writes its SavedVariables file only at logout
and `/reload`. Something outside the game has to carry data both ways, and live online status
needs something that is running while the game is. Three ways were put to the owner:

- a small companion program per member (chosen);
- the addon alone with copy and paste between game and site: no extra program, but no online
  status on the site and a sync only when someone bothers;
- only the owner's PC as a gateway, the addons talking to each other over the guild addon
  channel: nothing to install but the addon, but the site would only be as fresh as the owner's
  last logout.

The companion is written in **Go and shipped as one executable** (the owner's choice over a
PowerShell script: it runs on Windows, macOS and Linux, and Go is a language the owner wants in
use). It:

1. **installs the addon it carries inside itself** into the game's AddOns folder, and replaces
   it when a newer companion is run. No CurseForge, no zip to unpack (the owner, 2026-11-13);
2. uploads the characters when a SavedVariables file changes;
3. tells the hub once a minute that the game is running, which is the online status;
4. writes the hub's member list where the addon can load it;
5. **checks that the hub is reachable** (the owner, 2026-11-13) and, after a stretch in which it
   was not, reports that stretch with the next request that gets through, so the owner learns
   about downtime as members' PCs saw it.

What it never does: read the game's memory, touch any other file under `WTF`, send anything
that `docs/CONTRACT.md` does not list, run anything it downloaded, or update itself. It uses
the Go standard library only; a dependency is asked for first, like a Lua library.

This replaces part of D4 for this addon. D4's "no desktop daemon" no longer holds. Its "no
network from the addon" stands: the addon still does no I/O. Its "no cloud upload of game
data" becomes "game data goes to the group's own hub and nowhere else". D4's loop (junction,
pull script, Remote Control) stays the developer's feedback loop.

Releases are GitHub Releases of this repository, built by CI when the owner pushes a tag; the
hub's profile page links to the latest one. The executable is unsigned, so Windows SmartScreen
warns each member once. Windows is the supported platform for version one: setup, autostart
and the background run are built for it. On macOS and Linux the same program builds and, started
by hand, links the PC, uploads the characters and writes the member list. What those two lack
in version one: no autostart on either; on macOS the program does not look for the game at all,
so nobody shows as in game from a Mac, and no macOS file is published until someone needs one;
on Linux a file is published, and whether the game is found when it runs under Wine is not
known. (The plan's direction review, 2026-11-13, found these narrowings recorded only in review
files; whether anyone in the group plays on a Mac is a question put to the owner.)

## D8 — What crosses the wire, and what does not (2026-11-13)

`docs/CONTRACT.md` is the one home of the three formats: SavedVariables (addon to companion),
the data addon (companion to addon) and the sync request (companion and hub). The hub points at
it. The rules that shape them:

- **To the hub:** characters with class, race, level and skill lines, and whether the game is
  running. **To the game:** members' display names and their characters. Nothing else: no gold,
  play time, location, gear or chat, and nothing about players who are not members.
- **Online status means "the game is running", without a character name.** The client writes
  the addon's file only at logout, so while someone plays, the newest name the companion can
  know is the one from their previous session. A wrong name is worse than none.
- **No times are shown.** The hub shows nobody's last sync or last session (its D10 and D15).
- **A sync never deletes.** A character missing from an upload is left alone, because the
  client may not load SavedVariables back at login (`docs/RESEARCH.md` §1.3) and a file can
  hold only the characters of one session.
- **One companion links one member.** Every WoW account on that PC counts as theirs. Two
  members sharing a PC is a case version one does not handle (`docs/IDEAS.md`).
- **The member list is typed by people and executed by the game**, so the companion writes it
  under the escaping rule in `docs/CONTRACT.md` §2 and the addon treats it as untrusted.
- **A PC is linked by a logged-in member in the browser, and every request it sends is
  signed.** The first draft had the member paste a secret line from their profile into the
  companion. The owner, 2026-11-13: "Why can't we authenticate this? The companion tool
  provides the login, get some token, that token is used to authenticate? ... Think about
  secure alternatives as well", and of the four ways then put to them (browser approval;
  browser approval plus signed requests; a password typed into the companion; the pasted line)
  chose browser approval plus signed requests. So: the companion makes a key pair and shows a
  short code; the member approves that code on the hub, where they are logged in; the hub keeps
  only the public key; each request carries a signature and a rising sequence number
  (`docs/CONTRACT.md` §3.1, §3.2). No person ever sees a secret, the hub's database holds none,
  a recorded request cannot be sent twice, and a PC is cut off from the member's profile. The
  member list goes only to PCs linked this way. Linking adds two things to what reaches the
  hub: the PC's name, so the member can tell their PCs apart, and its public key.
