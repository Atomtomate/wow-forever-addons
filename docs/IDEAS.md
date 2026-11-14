# Ideas and future plans

The durable backlog; entries are updated, never deleted. Status words: **next**, **later**,
**parked** (needs something external), **dropped** (with the reason).

- **ForeverFriends, the link to the hub** — merged 2026-11-13 (D6 to D8), not yet run on a real PC
  or in the game. The addon records the character being played and shows the hub's member list; a
  companion program installs it, uploads the characters, reports that the game is running and checks
  that the hub is reachable. The wire formats are `docs/CONTRACT.md`.
- **MarketScan** — merged (PR #1), not yet run in the client. One-button full scan of the
  auction house, per-item minimum buyout, median, quantity and auction count, stored compactly
  in SavedVariables with a timestamp. Its "Verify in client" list is still open.
- **Addon list per member** — next, right after ForeverFriends version one (the owner,
  2026-11-13). Name, version and enabled state of each installed addon, shown on the member's
  hub profile. `C_AddOns` has the calls (`docs/RESEARCH.md` §6). Waits only so version one
  stays small.
- **Whose character is this, in the tooltip** — next. One line on a player's tooltip when the
  name is a member's character. Waits for a look at the tooltip hook in the real client, where
  unit names can be secret values (`docs/RESEARCH.md` §6); a tooltip that errors nags on every
  mouse-over.
- **Who of us is online, shown in the game** — later. From the guild roster or the friends
  list, or from the addons greeting each other over an addon channel. Waits for the client
  checks of those calls.
- **The character's name next to "in game"** — parked. The client writes SavedVariables only at
  logout, so the name is not knowable while someone plays (D8). Unparks if the client turns out
  to flush the file earlier, or if addons relay it to a member whose companion uploads.
- **Companion on macOS and Linux: find the game, a macOS download, autostart** — next, right
  after version one. The owner, 2026-11-13: "we have mac and linux users". In version one a Mac
  or Linux player links, uploads and gets the member list, started by hand, but never shows as
  in game: the program does not look for the game on macOS, and on Linux a game under Wine or
  Proton runs as the loader, not as a file in the game folder (matching the process's command
  line is the likely way; nobody has tried it). Needs someone with a Mac and someone on Linux
  to run the checks. Which comes first, this or the addon list, is the owner's to say. Since D9
  (2026-11-14) this is done in the companion app, not the Go program: piece 6 of the plan in
  `docs/HANDOFF.md`.
- **Companion: a tray icon** — taken into D9 (2026-11-14): the companion app is tray-first, built
  in piece 1 of the plan in `docs/HANDOFF.md`.
- **Companion app: updates itself** — parked on the owner's decision (D9, open decision 2). D7
  says the companion never runs anything it downloaded and never updates itself; painless
  adoption argues for silent updates verified against a key pinned in the app. Needs the owner's
  word and a change to `docs/CONTRACT.md` §5 before any code; then it is piece 7.
- **Companion app: a signing route for Windows and macOS** — parked on the owner's decision (D9,
  open decision 1). Unsigned means a browser warning, a SmartScreen warning, and a block where
  Smart App Control is on. Routes: a certificate in the owner's name (likely still warns for a
  group this small), free signing for open source (needs an OSI licence; there is no `LICENSE`
  file), the Microsoft Store. macOS notarization costs USD 99 a year. `docs/RESEARCH.md`
  "Companion app: choosing the technology".
- **"Connect this PC" link** — later, piece 8, needs the hub's side
  (`Atomtomate/wow-forever-hub`). A button on the hub page opens the app through a custom URL
  scheme with the hub's address and a one-time code, so setup needs no typing and the address
  never sits in this public repository or a release (D9). Not yet decided; the URL scheme itself
  is registered by piece 1.
- **Companion app: a Discord item** — later. Account linking and any bot belong on the hub; the
  app needs only a menu item that opens the hub's page for it. Waits for the hub to have one.
- **Companion app: Discord Rich Presence** — later. The only Discord feature that would live in
  the app itself; it would show what the member is doing to their Discord friends, so it needs
  the owner's look at D8's "what crosses" first.
- **Companion app: hub pages inside the app** — later. Showing a hub page in the app's window
  instead of the browser. Waits for the app's window to exist (piece 4) and for someone who
  misses it.
- **Hub skin package** — deferred by the owner ("Not really necessary"). Blizzard's art stays on
  the private hub and never enters this repository or a release (D9); a skin package installed
  through a hub feature belongs to `Atomtomate/wow-forever-hub`'s backlog, not here.
- **Companion: two members on one PC** — later. One companion links one member (D8); a mapping
  from WoW account to member would lift that. Unparks when a couple asks.
- **Companion installs MarketScan too** — later, once MarketScan has run in the client.
- **Price history and tooltips** — later. Needs the client to load SavedVariables back
  (`docs/RESEARCH.md` §1.3); until then every session starts empty.
- **Prices on the hub** — later. Upload a scan summary to the Forever Friends hub so the group
  sees prices on the site. The transport would be the companion (D7); needs a decision about
  what the hub stores.
- **Headless UI simulator in CI** — later. Osso/wow-ui-sim with the Forever profile, for load
  errors; D3 tier 2.
- **Lua Language Server annotations** — later. Ketho's vscode-wow-api for the editor; no
  Forever set yet, retail's is the closest.
- **Publishing on CurseForge or Wago** — later, only if an addon is useful beyond the group.
  The BigWigs packager handles it; whether it knows Forever as a flavour is unverified.
