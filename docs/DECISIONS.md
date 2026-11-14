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
so nobody shows as in game from a Mac; on Linux whether the game is found when it runs under
Wine is not known. Files are published for Windows, Linux and macOS. (The plan's direction review, 2026-11-13, found these narrowings recorded only in review
files.) Asked whether anyone plays on a Mac, the owner answered the same day: "we have mac and
linux users". So these two are not a footnote: finding the game on macOS and under Wine or
Proton on Linux, a macOS download and autostart on both are the work right after version one
(`docs/IDEAS.md`), and until then a Mac or Linux player links, uploads and gets the member
list, but does not show as in game.

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
- **The hub keeps the game's time of a character's last read and never shows it.** Two
  reviewers of the hub's side found that a member with two PCs could have last week's file
  from one overwrite today's from the other. Asked whether the hub may keep that time, hidden,
  only to refuse an older upload, the owner chose "Store it, never show it" (2026-11-13). So
  "nothing else" in the first rule above reads with `docs/CONTRACT.md` §5, which lists every
  field that crosses, `seen` and the outage stretch included.

## D9 — The companion becomes a desktop app in Rust and Tauri 2, tray-first (2026-11-14)

The owner, in the companion-app coordinator session (2026-11-13 and 14), asked for "a shippable
executable that gives a companion app (in windows I expect this to be optionally in the startup
and with a tray icon, extendable with linking discord)", and chose Rust with Tauri 2: "I think
tauri is a good option"; "I wanted to get a bit into rust anyway. It feels like the correct
lightweight choice for a small scale project like this". The priorities behind the choice:
"language irrelevant, one person project, should feel native and reactive", and above all
painless adoption: "All users of this will be in my friend group and I want to provide them an
experience that is as easy as possible, without solutions that feel like 'custom'/'shaky' or
'home built, we need to work on it ourselves'. They do not come to this project voluntarily
(like for an open source project somebody discovers) so having an absolutely painless adoption
cycle is key."

**The app is tray-first.** No window exists until the member asks for one: a web-view window
costs memory beside the game. It lives in `companion-app/`. Optional autostart on Windows, a
tray icon, a window for setup and status on demand, and room for a Discord item later.

**Alternatives that were weighed**, with why they lost (the ranking, the case against Tauri and
what would flip it are `docs/RESEARCH.md` §7):

- **C# with Avalonia 12** came second: a window that looks like Windows without styling work and
  the best headless tests, but a resident runtime of 60 to 120 MB, a tray menu that Avalonia
  draws itself on Windows, and hand-made release steps for macOS and Linux.
- **A tray library with no framework** (the Go program with `fyne.io/systray`, or the same in
  Rust) came third: the smallest and nothing can look foreign, but no window to grow into (and a
  first-run window is needed, because Windows 11 hides a new tray icon in the overflow), and the
  installer, updater and macOS and Linux packaging are assembled by hand.
- **Go with Wails 3** is Tauri's twin in Go and would keep the existing language, but 3.0.0 was
  tagged on 2026-10-26 after years of alpha: three weeks of stability is not "nothing shaky". It
  is worth a second look after six quiet months.
- **Electron** is what every WoW companion uses and the most trodden path, but ships a browser
  (about 373 MB on disk, 110 to 280 MB for an empty window) that must be re-released several
  times a year: it fails "quiet beside a game".
- **A page in the browser served on loopback** needs no window toolkit, but a browser tab as the
  settings is the home-built signal the owner ruled out, and it needs a hardened local server.
- **Java or Python toolkits** need a resident runtime (a JVM of 150 MB and up) or a frozen Python
  build (the worst case for antivirus), and their trays are weak on Linux.

The research's strongest case against Tauri is recorded with it: the existing Go program behind
an ordinary installer and a small updater would give friends nearly the same experience sooner,
and on Windows a Tauri window costs more memory while open than Electron's in the one benchmark
found. Tray-first is the answer to the second point; whether the web view's processes exit when
the window closes is checked on the owner's PC in piece 1.

**What it replaces in D7.** These sentences no longer hold: that the companion "is written in
**Go and shipped as one executable**", and "It uses the Go standard library only; a dependency
is asked for first, like a Lua library" (the owner: "This may be an old rule from another
project. We do not have this 'no dependency' rule here"); and the console setup D7's companion
does. What stands: D7's five jobs (install the carried addon, upload characters, report that
the game runs, write the member list, check that the hub is reachable) and its "what it never
does" list, including "run anything it downloaded, or update itself", which stands until the
owner decides open decision 2 below. D7's paragraph on releases stands until piece 5 of the plan
(`docs/HANDOFF.md`) replaces it; its platform paragraph is carried by piece 6. D7's text is not
edited (this file is append-only); read it with this entry, as D7 is read with D4. One fact in D7's
release paragraph was too kind: an unsigned program does not warn "each member once" but at every
new version, because reputation is per file hash (`docs/RESEARCH.md` §7.1).

**The Go companion** in `companion/` (v0.1.0, never run on a real PC, never tagged) stays in the
tree as the reference implementation and test oracle until the Rust app reaches parity. The
core of the app is ported to a Rust library crate with no Tauri dependency and proven against
the Go program byte for byte with golden files generated from it.

**What is public and what is private.** The owner: "The repository is just the code! Careful!
When talking about private, I mean the actual instance that is being hosted, where all the data
lives." and "I like the semi-public route". So this repository and its releases are public, and
the hosted hub instance and its data are private, invite-only. It follows that GitHub Releases
is an acceptable download channel, and that neither the app nor this repository may contain the
hub's address or any instance data: the address reaches the app when a PC is linked. (The idea
on the table, not decided and needing the hub's side: a "Connect this PC" link on the hub page
that opens the app through a custom URL scheme with the address and a one-time code, so setup
needs no typing; `docs/IDEAS.md`.)

**Assets.** The owner: "we need to take care of the non-free assets we are using. Those are still
possible for our private instance, but need to be replaced with license free ones for the
releases (the custom skin package can then be installed through a private hub feature)". So
nothing non-free is in this repository or in a release. Every bundled asset (app icon, tray
icon, installer art, fonts) is original or freely licensed, with its source and licence recorded
next to it. Blizzard's art stays on the private hub. The skin package as a hub feature is
deferred ("Not really necessary" for now), and the hub's code stays private "for the
foreseeable future". Today this repository holds no image, font or sound file; in the game, an
addon may refer to the client's own textures by path without shipping them.

**Open decisions**, each the owner's. A recommendation marked as such is the coordinator
session's or its research's, not the owner's word.

1. **Signing, and the Windows trust prompts.** Unsigned, an installer shows a browser warning and
   a SmartScreen warning, and is blocked outright where Smart App Control is on. A certificate in
   the owner's own name (roughly EUR 65 to 290 a year in the EU) shows that name to friends but
   likely still warns, since reputation needs hundreds of installs; whether it passes Smart App
   Control is not verified. Free signing for open source needs an OSI licence, and this
   repository has no `LICENSE` file. The Microsoft Store (private audience) is the only route
   with no warning, at the price of Microsoft's review with a demo account. Apple notarization
   costs USD 99 a year. *Recommendation (the research):* a certificate in the owner's name if
   there is any budget; first, let the owner see the unsigned prompts on the PC (piece 1). The
   research ruled out open-source signing and the Store's public listings for privacy reasons
   that the owner's public/private line no longer supports: those routes are open and unexamined.
2. **Self-update.** D7 says the companion never runs anything it downloaded and never updates
   itself. Painless adoption argues for silent updates. *Recommendation (the research):* updates
   verified against an Ed25519 key whose public half is built into the app and whose private half
   signs in CI or offline and is never on the hub; fail closed, refuse a lower version. Needs the
   owner's word, a new D-entry that changes D7 on purpose, and a change to `docs/CONTRACT.md`
   before any code (piece 7). Where the update manifest comes from (the hub, as the research
   assumed, or the public releases) is part of that decision.
3. **Discord.** Account linking and any bot belong on the hub; the app needs only a menu item that
   opens the hub. Only Rich Presence would live in the app. *Recommendation (the research):* the
   menu item once the hub has a page for it; Rich Presence only after the owner looks at what it
   would show (D8).
4. **The Go companion's end, and the first release.** *Recommendation (the coordinator):* remove
   `companion/`, its workflow and `go.mod` once the app reaches parity (piece 5), never tag
   v0.1.0, and make the Tauri app the first release.
5. **A licence file.** Whether this repository gets one; it bears on open decision 1 and on what
   anyone may do with public code. No recommendation recorded.
6. **How the hub's address reaches the app.** The "Connect this PC" link (a custom URL scheme
   carrying the address and a one-time code, confirmed in the app) is the idea on the table; it
   changes `docs/CONTRACT.md` §3 and needs the hub's side (piece 8). Until then the member types
   or pastes the address once, as in the Go companion.
