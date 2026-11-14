# WoW: Forever addon development: research note

Date of research: 2026-11-09 (five days after launch). Method: WebSearch plus WebFetch. Marking used below:
**[opened]** = I fetched the page and read it; **[snippet]** = the claim rests on a search-result summary only;
**[unverified]** = no source found, or my own inference. Most of this is community material and changes weekly.

## 0. The premise needs correcting first

The brief assumed Forever exposes the Classic Era / Anniversary addon API. The evidence says otherwise:

- Reports of Blizzard's 17 September 2026 Q&A say Forever shares Mainline's UI architecture, "the vast majority of APIs available in 12.1.5", and that Midnight's addon restrictions (secret values) apply ([Warcraft Tavern](https://www.warcrafttavern.com/forever/news/wow-forever-developer-qa-elf-races-character-models-gdkp/), [Icy Veins](https://www.icy-veins.com/wow-forever/news/addons-in-wow-forever-blizzard-devs-just-addressed-the-big-question/), [GamesRadar](https://www.gamesradar.com/games/world-of-warcraft/world-of-warcraft-forever-takes-page-out-of-retails-book-as-blizzard-confirms-access-to-add-ons-will-be-restricted-in-the-throwback-mmo/)). **[snippet]** only, all secondary; I found no Blizzard transcript.
- The auction house is the modern one: an addon-author issue reading the Forever UI source says the client has `Blizzard_AuctionHouseUI` and `C_AuctionHouse`, "not the classic `QueryAuctionItems`" ([forever-armoury-addon #45](https://github.com/mhealy-dev/forever-armoury-addon/issues/45)) **[opened]**. An Auctioneer port agrees that Forever uses Mainline's retail-style auction house ([NoobyGains/Auctioneer-Forever](https://github.com/NoobyGains/Auctioneer-Forever)) **[opened]**.
- Blizzard's `forever` branch of the UI source mirrors this: its `AuctionHouseDocumentation.lua` lists `ReplicateItems`, `SendBrowseQuery` and the other `C_AuctionHouse` functions ([Gethe/wow-ui-source, forever branch](https://raw.githubusercontent.com/Gethe/wow-ui-source/forever/Interface/AddOns/Blizzard_APIDocumentationGenerated/AuctionHouseDocumentation.lua)) **[opened]**.

So the plan below treats Forever as a retail-lineage (12.1.5-like) client with a Classic-style game, and Classic Era API knowledge (section 4) as background only. Classic-era addons need porting, not just a new TOC number ([Auctioneer-Forever](https://github.com/NoobyGains/Auctioneer-Forever) lists removed globals such as `GetItemInfo`, now `C_Item`) **[opened]**.

## 1. Addon basics as they apply to Forever

### 1.1 Anatomy
Standard and unchanged across flavours (general knowledge; the Warcraft Wiki pages for it were not reachable, see the end). **[unverified here]**
- Folder `Interface/AddOns/<Name>/` containing `<Name>.toc` plus Lua (and optionally XML) files listed in the TOC in load order.
- TOC directives: `## Interface:`, `## Title:`, `## SavedVariables:`, `## SavedVariablesPerCharacter:`, `## Dependencies:` / `## OptionalDeps:`. Interface may list several comma-separated values; the BigWigs packager reads that ([packager README](https://github.com/BigWigsMods/packager)) **[opened]**.
- Lua 5.1 with Blizzard's sandbox (no `io`, `os` mostly absent, no `require`). Retail-lineage clients add taint and secret-value rules; WeakAuras-style combat scraping is restricted ([Icy Veins](https://www.icy-veins.com/wow-forever/news/addons-in-wow-forever-blizzard-devs-just-addressed-the-big-question/)) **[snippet]**.
- Event-driven: create a frame, `RegisterEvent`, handle in `OnEvent`. SavedVariables are plain Lua globals written to `WTF/Account/<ACCOUNT>/SavedVariables/<Addon>.lua` on logout or `/reload` ([Warcraft Wiki: Saved variables](https://warcraft.wiki.gg/wiki/Saved_variables)) **[snippet, URL only seen in search results]**.

### 1.2 Interface number and folder
- **16001** is the Forever Interface value. The Warcraft Wiki TOC page lists it next to 11509 (Classic Era) and 120105 (retail) ([TOC format](https://warcraft.wiki.gg/wiki/TOC_format)) **[snippet]**. A real Forever addon repo ships `## Interface: 16001` ([CKozanecki/FriendlyForever](https://github.com/CKozanecki/FriendlyForever)) **[opened]**, and so does the Auctioneer port **[opened]**. Forever game build quoted as client 1.60.1, build 70532, in one tooling README ([RikyZ90/wow-addon-engineer](https://github.com/RikyZ90/wow-addon-engineer)) **[opened]**.
- Confirm in-client with `/dump select(4, GetBuildInfo())` ([TOC format](https://warcraft.wiki.gg/wiki/TOC_format)) **[snippet]**.
- Folder: several guide sites say `World of Warcraft\_forever_\Interface\AddOns`; the beta used `_classic_beta_` ([guide aggregate via search](https://wowforeveraddons.com/articles/wow-forever-addons-folder), [FriendlyForever README](https://github.com/CKozanecki/FriendlyForever)) **[snippet]** / **[opened]** respectively. Blizzard itself was not found naming it. Check on the owner's PC.
- Source mirror: Gethe's `forever` branch, updated 7 November 2026, beside `live`, `ptr`, `classic`, `classic_anniversary` ([branches](https://github.com/Gethe/wow-ui-source/branches)) **[opened]**. I did not find an official Blizzard-hosted repo for Forever.

### 1.3 Known client bug that decides the feedback loop
During the beta the client wrote SavedVariables at logout but did not load them at the next start ([Blizzard forum thread](https://us.forums.blizzard.com/en/wow/t/2356559) **[snippet]**; [Auctioneer-Forever](https://github.com/NoobyGains/Auctioneer-Forever) **[opened]**; workaround [KonigTX/ForeverSVBridge](https://github.com/KonigTX/ForeverSVBridge) **[opened]**). I could not establish whether launch fixed it: the bridge README does not say. Writing is what matters for an export-only scanner, but any addon that reads its own history back (price deltas) is affected. Test it first on day one.

### 1.4 Where the API reference lives
| Source | Covers Forever? |
|---|---|
| In-game `/api` (Blizzard_APIDocumentation) | Yes by construction; it is the client's own. Not independently checked. **[unverified]** |
| `Blizzard_APIDocumentationGenerated` in the `forever` branch | Yes; I read the AuctionHouse file **[opened]** ([link](https://raw.githubusercontent.com/Gethe/wow-ui-source/forever/Interface/AddOns/Blizzard_APIDocumentationGenerated/AuctionHouseDocumentation.lua)). Many entries have no prose Documentation. |
| [Ketho/vscode-wow-api](https://github.com/Ketho/vscode-wow-api) | Badges list retail 12.0.1, MoP Classic 5.5.3, Vanilla 1.15.8; no Forever mentioned **[opened]**. Retail annotations are the closest fit; expect gaps. |
| Warcraft Wiki API pages | Retail-centric; Forever pages not confirmed. Host blocked here. |
| Third-party Forever guides ("API cheat sheet", "patterns") | Exist ([search hit](https://wowforeverguides.com/addons/api-cheatsheet)); host blocked, **[snippet]**, treat as unverified. |

## 2. Tooling outside the game

### 2.1 Linting and formatting
- **luacheck**: upstream README says latest release 0.23.0, config via `.luacheckrc` with custom globals ([mpeterv/luacheck](https://github.com/mpeterv/luacheck)) **[opened]**; the page did not show recent activity, and a maintained fork exists under lunarmodules (**[unverified]**, from memory). A Forever addon repo uses luacheck with Lua 5.1 rules ([FriendlyForever](https://github.com/CKozanecki/FriendlyForever)) **[opened]**; its config is a reasonable template to copy. I did not open it.
- **Lua Language Server** with Ketho's annotations: installs as a VS Code extension, activates when a folder has a `.toc` ([vscode-wow-api](https://github.com/Ketho/vscode-wow-api)) **[opened]**. Annotations are generated with `lua luasrc/init.lua` ([setup README](https://github.com/Ketho/vscode-wow-api/blob/master/setup/README.md)) **[opened]**. For Claude Code in a container, plain luacheck is simpler than LuaLS.
- **StyLua**: used by a Claude-oriented addon toolkit ([wow-addon-engineer](https://github.com/RikyZ90/wow-addon-engineer)) **[opened]**. Config details not checked.
- An existing AI-oriented toolkit offers a TOC/file-chain checker that "does not execute Lua or validate API semantics" ([wow-addon-engineer](https://github.com/RikyZ90/wow-addon-engineer)) **[opened]**. Useful to catch typos in TOC paths, the commonest silent addon failure.

### 2.2 Packaging and release
- **BigWigs packager**: `release.sh`, versions from git tags, optional `.pkgmeta` (`externals`, `ignore`, `package-as`, and so on), GitHub Action `BigWigsMods/packager@v2`, uploads to CurseForge, WoWInterface, Wago and GitHub Releases via tokens; supports per-flavour TOCs ([README](https://github.com/BigWigsMods/packager)) **[opened]**. Its listed flavour set (Classic, BCC, Wrath, Cata, Mists) does not name Forever; whether `-g` accepts it is **[unverified]**. For a private group, plain zip or the junction (3.3) is enough; the FriendlyForever repo has a `make package` zip target and a gated CurseForge release workflow ([FriendlyForever](https://github.com/CKozanecki/FriendlyForever)) **[opened]**.

### 2.3 Libraries
LibStub, Ace3, LibDataBroker: all pure Lua, no flavour-specific parts that I know of **[unverified for Forever]**. Caveats found: an AceDB initialisation-timing workaround did not cure the SavedVariables bug ([Blizzard thread](https://us.forums.blizzard.com/en/wow/t/2356559)) **[snippet]**; an Auctioneer port needed edits to `LibRevision` for `C_AddOns` ([Auctioneer-Forever](https://github.com/NoobyGains/Auctioneer-Forever)) **[opened]**. Recommendation: avoid Ace3 for a small scanner; use a plain frame and a global table.

### 2.4 Testing outside the client
| Tool | What it is | Forever? | Use for this project |
|---|---|---|---|
| **busted** + home-grown stubs | Lua unit tests, run under a normal Lua 5.1 or LuaJIT | Neutral | Pure logic: packing, price maths, dedupe. Best value. |
| [Adirelle/wowmock](https://github.com/Adirelle/wowmock) | Environment mock for tests, built for luaunit 2.1 / mockagne | Neutral, last updated about 12 years ago per search summary | Idea only; do not adopt. **[snippet]** |
| [FriendlyForever](https://github.com/CKozanecki/FriendlyForever) `spec/support/wow.lua` | busted suite on a simulated client, in a live Forever addon | Yes | Best template to read. **[opened]**, file itself not read |
| [wowless/wowless](https://github.com/wowless/wowless) | Headless interpreter for client Lua and FrameXML; README says "still pre-alpha"; Docker-only development; page mentions only retail interface code; run with `bin/run.sh wow --addondir path` | Retail only per README | Not for AH logic. **[opened]** |
| [HogSim](https://github.com/Hogtron-Solutions-Tech-Stack/hogsim) | Fork of wowless tagged `wow-forever`; "runs Blizzard's real Forever UI plus your addon", but README is inherited and documents no Forever specifics; 0 stars, no releases | Claimed | Experimental. **[opened]** |
| [Osso/wow-ui-sim](https://github.com/Osso/wow-ui-sim) | Rust simulator; profiles include WoW Forever (`wowf`); Linux/Windows release packages for Forever; Docker image and a GitHub Action; tests are Lua files under `Interface/AddOns/<Name>/tests/` with `assertEquals`; GPL-3.0; Forever builds need extra spell data | Yes, listed | Most promising headless option. **[opened]** |
| Third-party review of both | Issue #49 in FriendlyForever: adopted wow-ui-sim at "level A" (load errors, frame tree) and "level B" (screenshots); noted both are unreviewed code, pinned versions and a throwaway container; Blizzard art absent | n/a | Follow its safety stance. **[opened]** ([issue](https://github.com/CKozanecki/FriendlyForever/issues/49)) |

Reach for an AH scanner: none of these supply a real auction house. At best a simulator loads the addon and reports Lua errors; the AH events and data exist only in the real client. I found no statement that any of them fakes `C_AuctionHouse` (**[unverified]**). Plan: simulator for load/syntax errors, busted for logic, real client for behaviour.

### 2.5 Testing inside the client
General practice, from memory **[unverified here]**: `/console scriptErrors 1`, BugSack with BugGrabber, `/dump`, `/etrace`, `/fstack`, DevTool, WoWUnit ([CurseForge WoWUnit](https://www.curseforge.com/wow/addons/wowunit) **[snippet]**). A Claude toolkit includes a script that reads BugSack data ([TheMizeGuy/wow-addon-dev](https://github.com/TheMizeGuy/wow-addon-dev)) **[opened]**. An addon cannot write files; SavedVariables is the one persistence path **[unverified here, standard knowledge]**. A cheap log: append to a capped ring table `ScannerDB.log`, so it appears in the same file as the data.

## 3. Feedback loop for an AI assistant

### 3.1 Existing practice
LLM-plus-WoW-addon material does exist, mostly as skills:
- [DennysOliveira/wow-addon-dev](https://github.com/DennysOliveira/wow-addon-dev): Claude Code skills, Midnight 12.0+ focus. **[snippet]**
- [TheMizeGuy/wow-addon-dev](https://github.com/TheMizeGuy/wow-addon-dev): SKILL.md plus 11 references, templates, seven PowerShell scripts (list, validate, install, package, read BugSack); covers Retail and Classic, does not mention Forever or the AH. **[opened]**
- [RikyZ90/wow-addon-engineer](https://github.com/RikyZ90/wow-addon-engineer): seven skills, optional `wow-api-mcp`; tiers structural, static, simulated, real-client; explicitly "does not certify Forever"; mentions no SavedVariables read-back and no junctions. **[opened]**
- [JBurlison/WoWAddonAPIAgents](https://github.com/JBurlison/WoWAddonAPIAgents), [refaim/MasterTradeSkills CLAUDE.md](https://github.com/refaim/MasterTradeSkills/blob/master/CLAUDE.md): **[snippet]** only.
- FriendlyForever contains `CLAUDE.md`, `AGENTS.md` and `.claude/`, contents not shown ([repo](https://github.com/CKozanecki/FriendlyForever)) **[opened]**.
- A Forever-focused guide advertises "AI prompts" ([wowforeverguides.com](https://wowforeverguides.com/addons/developers)) **[snippet]**; host blocked.
None of these documents a closed loop in which the assistant reads in-game output. That part is ours to build.

### 3.2 Getting data out
- SavedVariables on logout or `/reload`; the file is plain Lua, so a script can parse it. Desktop tools (TSM app, Oribos-Exchange-style exporters) work this way **[unverified here; from memory]**.
- File watcher or hook: copy `WTF/Account/*/SavedVariables/<Addon>.lua` into the repo on change **[my design, section 5]**.
- Forever-specific wrinkle: the load bug (1.3) means a bridge addon may use a directory junction into `SavedVariables` ([ForeverSVBridge](https://github.com/KonigTX/ForeverSVBridge)) **[opened]**. That confirms junctions are permitted on Windows installs.

### 3.3 Live edits
Junction: `mklink /J "<WoW>\_forever_\Interface\AddOns\MyAddon" "C:\src\addons\MyAddon"`. Standard Windows command **[unverified here]**; one toolkit warns against copying a repo into `Interface/AddOns` ([wow-addon-engineer](https://github.com/RikyZ90/wow-addon-engineer)) **[opened]**. `/reload` picks up edits to existing files; a new addon folder needs a full restart ([Auctioneer-Forever](https://github.com/NoobyGains/Auctioneer-Forever)) **[opened]**. Folder name `_forever_` per section 1.2.

## 4. Auction house API

### 4.1 What Forever has
From the `forever` branch documentation file **[opened]** ([source](https://raw.githubusercontent.com/Gethe/wow-ui-source/forever/Interface/AddOns/Blizzard_APIDocumentationGenerated/AuctionHouseDocumentation.lua)):
- `C_AuctionHouse.ReplicateItems`, `GetNumReplicateItems`, `GetReplicateItemInfo` exist, none marked `HasRestrictions`; no `HardwareEvent` markers appear anywhere in the file. The doc string says ReplicateItems replaces an "allItem" `QueryAuctionItems` call.
- `SendBrowseQuery`, `SendSearchQuery`, `RequestMoreBrowseResults`, `GetBrowseResults` exist, unrestricted; search is "restricted to 100 calls per minute". `SendBrowseQuery`, `SendSearchQuery` and `GetReplicateItemInfo` carry `SecretArguments = AllowedWhenUntainted`; what that means for an addon I could not determine. **[unverified]**
- Restricted (`HasRestrictions = true`): `PlaceBid`, `PostItem`, `PostCommodity`, `ConfirmPost*`, `StartCommoditiesPurchase`, `CancelAuction`. Read-only scanning stays outside that set.
- Throttle events: `AUCTION_HOUSE_THROTTLED_MESSAGE_QUEUED/SENT/RESPONSE_RECEIVED/DROPPED` and `AUCTION_HOUSE_THROTTLED_SYSTEM_READY`; `IsThrottledMessageSystemReady()`.
- `GetReplicateItemInfo` returns: name, texture, count, qualityID, usable, level, levelType, minBid, minIncrement, buyoutPrice, bidAmount, highBidder, bidderFullName, owner, ownerFullName, saleStatus, itemID, hasAllInfo.
A missing restriction flag is not proof of behaviour; the issue's author says to test with a throwaway script ([#45](https://github.com/mhealy-dev/forever-armoury-addon/issues/45)) **[opened]**. That issue is an open spike with no findings yet.

### 4.2 Retail-lineage behaviour (applies if Forever matches 12.1.5)
Search summary of the Warcraft Wiki **[snippet]** ([ReplicateItems](https://warcraft.wiki.gg/wiki/API:C_AuctionHouse.ReplicateItems), [GetReplicateItemInfo](https://warcraft.wiki.gg/wiki/API:C_AuctionHouse.GetReplicateItemInfo)):
- One successful `ReplicateItems` per 15 minutes, account-wide; result arrives as `REPLICATE_ITEM_LIST_UPDATE`; read indices 0 to `GetNumReplicateItems()-1`.
- `count`, `minBid`, `buyoutPrice`, `itemID` are available at once; other fields depend on item data loading (`hasAllInfo`).
- `owner` is nil except for your own auctions (player names removed in a 9.0.2 hotfix), so per-seller data is unavailable.
- One report of disconnects on a realm with over 80k auctions; per-frame event limit about 2000 as of 2022. Could differ on Forever; a Forever realm may be much smaller. **[unverified for Forever]**
- Time-left has a separate call, `GetReplicateItemTimeLeft` (URL appears in search results only).

### 4.3 Classic Era API (background; not what Forever exposes)
`QueryAuctionItems` with getAll: about 0.3 s between normal queries, about 15 minutes for getAll, `CanSendAuctionQuery()` tells which is allowed, window must be open, risk of disconnects ([Warcraft Wiki: QueryAuctionItems](https://warcraft.wiki.gg/wiki/API:QueryAuctionItems), [Wowpedia](https://wowpedia.fandom.com/wiki/API_QueryAuctionItems)) **[snippet]**. Retail dropped it in 8.3.0 for `C_AuctionHouse`. I did not verify what Anniversary exposes today.

### 4.4 Existing scanners and Forever projects
- Auctionator has a Forever release (Wago file for client 1.60.1; dates differ between sites) ([listing](https://wowforeverbuilds.com/addons/auctionator)) **[snippet]**.
- "AuctionHouse Forever" on CurseForge records price history from your scans into tooltips ([page](https://www.curseforge.com/wow/addons/auctionhouse-forever)) **[snippet]**.
- Auctioneer port exists; stores history in SavedVariables ([Auctioneer-Forever](https://github.com/NoobyGains/Auctioneer-Forever)) **[opened]**; its scan method was not described.
- TSM on Forever: sources disagree ([search result](https://mmonster.co/wow-forever/guides/addons/professions-and-gold)) **[snippet]**.
- Auctionator's scan logic was not opened. Its source is the best reference for a modern `C_AuctionHouse` scan; read it before writing ours.

### 4.5 Terms and volumes
- Policy: Blizzard's published addon policy was not found or opened. Known from the unrestricted flags above: reads are free, spending actions are protected, so a scanner stays on the right side as long as it never automates buying or posting. **[inference]**
- Midnight-style rule that addons must not play the game for the player is reported for Forever ([Icy Veins](https://www.icy-veins.com/wow-forever/news/addons-in-wow-forever-blizzard-devs-just-addressed-the-big-question/)) **[snippet]**.
- Row counts and file size for a Forever realm: no source found. **[unverified]** Measure with one scan: count rows, check the `.lua` size.
- Compression approaches (mine, not sourced): store per item only count, min buyout, median, and a few percentiles, not raw rows; integer copper; one table of `[itemID] = "ts:min:med:qty"` strings.

## 5. Recommendation

### 5.1 Repository
One monorepo, `addons/<Name>/` each with its own TOC, plus shared `tools/`. Reason: small group, shared lint and CI config, one junction script, one CLAUDE.md. Split a repo only when an addon is published on its own.
```
addons/
  MarketScan/ (MarketScan.toc, Core.lua, Pack.lua, spec/)
tools/ (link.ps1, pull_sv.ps1, parse_sv.py)
.luacheckrc  .luarc.json  stylua.toml  .pkgmeta
.github/workflows/ci.yml
CLAUDE.md   (flavour facts from section 0, Interface 16001, folder _forever_)
```
Keep pure logic (`Pack.lua`: no globals from the client) in separate files so busted can load them.

### 5.2 Checks
- luacheck: `std = "lua51"`, a `globals` list seeded from the FriendlyForever config and extended per warning. Run in CI and as a pre-commit hook.
- LuaLS with Ketho's annotations as an editor nicety; optional.
- busted on `Pack.lua` and similar, with a 20-line stub (`C_Item`, `GetServerTime`). Do not adopt wowmock.
- Simulator (wow-ui-sim, pinned, throwaway container, read-only mount, no secrets, per [issue #49](https://github.com/CKozanecki/FriendlyForever/issues/49)): try for load errors only; GPL-3.0 is fine as an external tool.
- CI: lint, busted, TOC path check, `packager` run with `-d` (dry run flag **[unverified]**) or a simple zip. Release upload skipped.

### 5.3 Feedback loop
1. `tools/link.ps1` creates the `mklink /J` from `_forever_\Interface\AddOns\<Name>` into the checkout.
2. After `/reload`, `tools/pull_sv.ps1` (run by hand, a file watcher, or a scheduled task) copies `SavedVariables\<Addon>.lua` to `data/` in the repo; `parse_sv.py` turns it into JSON and prints a short summary.
3. Claude reads the summary and `ScannerDB.log` either via Remote Control on the Windows PC or from a pushed commit (use a separate `data` branch, not `main`).
4. If the load bug persists, nothing changes for export; for read-back, evaluate ForeverSVBridge **[opened]** knowing it is a third-party workaround that edits TOC load order.
Also capture BugSack output, as TheMizeGuy's scripts do **[opened]**.

### 5.4 First addon: MarketScan (sketch)
All calls below are from the Forever-branch documentation unless noted **[opened]**.
- Event frame; on `AUCTION_HOUSE_SHOW` (event name from general knowledge **[unverified]**) offer a button "Scan".
- On click: `C_AuctionHouse.ReplicateItems()`; wait for `REPLICATE_ITEM_LIST_UPDATE`.
- Loop `i = 0 .. GetNumReplicateItems()-1`, `GetReplicateItemInfo(i)`; per row use `itemID`, `count`, `buyoutPrice`, `minBid`; skip `buyoutPrice == 0`. Chunk the loop across frames (`C_Timer.After`) to avoid hitches; re-read rows with `hasAllInfo == false` later or ignore them.
- Aggregate per itemID: min unit buyout, median, total quantity, auction count. Save `MarketScanDB.scans[#+1] = {t = GetServerTime(), realm, items = {[id] = "min:med:qty:n"}}`; cap at N scans.
- Fallback if ReplicateItems is blocked or throttled: `SendBrowseQuery` with `RequestMoreBrowseResults`, honouring `IsThrottledMessageSystemReady` and the 100-per-minute search limit.
- Never call any restricted function.

### 5.5 Confirm in the real Forever client (day-one checklist)
1. `/dump select(4, GetBuildInfo())` returns 16001 and the folder is `_forever_`.
2. `/dump C_AuctionHouse.ReplicateItems` exists; one scan returns rows; row count and `.lua` size on this realm.
3. Whether a second `ReplicateItems` within 15 minutes is refused, and how.
4. Whether any AH call raises a taint or secret-value error, and whether a hardware event is required.
5. Whether SavedVariables load at login on the launch build.
6. Whether `count`/price fields are secret values (cannot be used in arithmetic) when read by an addon. **[unverified, high impact]**

## 6. What an addon that links to the hub can read (2026-11-13)

Read for D6 to D8 from the generated API documentation on the `forever` branch
(`https://raw.githubusercontent.com/Gethe/wow-ui-source/forever/Interface/AddOns/Blizzard_APIDocumentationGenerated/<file>`).
Every item is **[opened]**: the named file was fetched and the entry read. None of it has run
in the client; a flag in a documentation file is not proof of behaviour (§4.1).

- **The character's identity can be a secret value.** In `UnitDocumentation.lua`, `UnitName`,
  `UnitFullName`, `UnitClass`, `UnitRace` and `UnitGUID` carry
  `SecretWhenUnitIdentityRestricted = true`; `UnitLevel`, `UnitFactionGroup` and `UnitIsPlayer`
  do not. `C_Secrets.ShouldUnitIdentityBeSecret(unit)` (`SecretPredicateAPIDocumentation.lua`)
  "returns true if queries for unit identity (such as name or GUID) will produce secret
  values". When identity is restricted, and whether it ever is for `"player"` out in the world,
  the files do not say. Code that records the character must check before it stores, and code
  that compares a unit's name with a list (a tooltip line) must survive a secret.
- **Locale-independent tokens exist.** `UnitClass` returns `className, classFilename, classID`
  and `UnitRace` returns `localizedRaceName, englishRaceName, raceID`. The group plays in
  German, so anything sent to the hub uses `classFilename` and `englishRaceName`. The tokens of
  the two Skyborne races are not in the documentation.
- **Skills come from `C_SkillInfo`** (`SkillInfoDocumentation.lua`): `GetNumSkillLines()` and
  `GetSkillLineInfo(index)`, which returns a `SkillLineAttributes` table with `skillID`, `name`
  (localized), `isHeader`, `rank`, `maxRank`, `parentSkillLineID`, `skillLineCategoryID` and
  more; the event is `SKILL_LINES_CHANGED`. Which `skillID` is which profession is not in the
  file; `docs/CONTRACT.md` §4 uses the ids Classic and retail share, unverified for Forever.
- **The addon list is readable** (`AddOnsDocumentation.lua`, namespace `C_AddOns`):
  `GetNumAddOns`, `GetAddOnName(index)`, `GetAddOnInfo` (name, title, notes, loadable, reason,
  security), `GetAddOnMetadata(name, variable)`, `GetAddOnEnableState(name, character)`,
  `IsAddOnLoaded`.
- **Addons can talk to each other in game.** `C_ChatInfo.SendAddonMessage(prefix, message,
  chatType, target)` and `SendAddonMessageLogged` exist (`ChatInfoDocumentation.lua`), both
  with `SecretArguments = "NotAllowed"`: a secret value cannot be sent. Not used in version
  one; it is the route to in-game presence later (`docs/IDEAS.md`).
- **The client says which platform it is**: `IsWindowsClient`, `IsMacClient`, `IsLinuxClient`
  (`BuildDocumentation.lua`), beside `GetBuildInfo`.
- `C_FriendList`, `C_GuildInfo.GuildRoster` and `C_BattleNet` exist; only their function names
  were read.
- Not in these files, so not checked: the tooltip hook (it is FrameXML Lua, not generated
  documentation), the older global functions for professions and guild rosters, and the name of
  the game's executable, which the companion needs to tell that the game is running.

That an addon has no file or network access is standard knowledge and was not re-derived here
**[unverified here]**; D7 rests on it.

## 7. Companion app: choosing the technology (2026-11-14)

Condensed for D9 from the companion-app coordinator session's report
(`companion-app-open-search.md`, written 2026-11-14, about 860 lines, not in this repository).
Marks as in that report: **[P]** primary source read that day (where the site was blocked, the
copy in the project's own GitHub repository); **[S]** a web-search summary of a page that could
not be opened; **[K]** the author's knowledge, not checked. Every size, start time and memory
figure is **quoted, not measured**. Nothing in it was run on Windows or macOS.

**Read with the owner's correction.** The report's distribution part was written under a stricter
reading of "private" than the owner meant: it took the whole project as private and so ruled out
GitHub Releases, public or link-reachable listings, and free signing for open source. The owner
then said that the repository and its releases are public and only the hosted hub instance and
its data are private (D9). Where the report rules a route out for privacy, that ruling no longer
holds: those routes are open again, and **not examined since**. What it says about the warnings a
friend sees, about reputation and about the stacks is unaffected.

### 7.1 The answer in one page

- **The judge** is a friend on Windows 11 who clicks a link in the group chat and has no patience.
  A "scary moment" is one where the system says the program may be dangerous, or where they must
  do something they could not do without a screenshot.
- **Unsigned** (any stack): the browser's "isn't commonly downloaded", then "Windows protected your
  PC", More info, Run anyway [P][S]. Reputation is per file hash, so every new version starts at
  zero [P]; a file fifteen people download never builds any. Where **Smart App Control** is on, an
  unsigned file without reputation is blocked with no per-app bypass, Mark of the Web or not [P][S];
  since April 2026 it can be switched on without reinstalling Windows [S].
- **Signed in the owner's name**: certificates for a natural person exist in the EU (Certum
  "Standard", SSL.com "Individual Validation", Sectigo through resellers; roughly EUR 65 to 290 a
  year; key on a token or in the CA's cloud, the cloud one being what works from CI; at most about
  460 days) [S]. Friends see the owner's legal name as publisher. SmartScreen most likely still
  warns: reputation "can take several weeks and hundreds of clean installs from a wide audience"
  [P]. Whether Smart App Control passes a new, validly signed file was **not verified**.
- **Free signing for open source**: SignPath Foundation needs an OSI licence (this repository has
  none), MFA, a published signing policy and their approval; the publisher shown is "SignPath
  Foundation" [S]. Certum's "Open Source" certificate asks for the project's website and puts "Open
  Source Developer" in the name [S]. Azure Artifact Signing takes individuals only in the US and
  Canada [P].
- **Microsoft Store**: the only route with no warning at all [P]. Free account for individuals since
  2025-09-10 [S]; certification up to three business days, again for every update [P]. "Private
  audience" hides the listing from everyone but listed Microsoft accounts, "even if they were able to
  type in its Store listing URL" [P]. The policies ask for a working demo account and a reachable
  server (10.3.1, 10.3.2), a privacy policy URL (10.5.1), a disclosed dependency (10.2.4) and forbid
  downloading code that changes what the app does (10.2.2) [P]. Under MSIX autostart is a manifest
  startup task, a `foreverfriends://` protocol is one manifest element, AppData and HKCU writes are
  redirected and removed on uninstall, writes to the game folder are not [P]. **Whether the Store
  certifies this app is the report's largest unknown**; only a submission answers it.
- **macOS**: unsigned means "Apple could not verify...", then System Settings, Privacy & Security,
  Open Anyway, an administrator password; the Control-click bypass is gone since macOS 15 [S].
  Developer ID plus notarization costs USD 99 a year for an individual, no waiver [S]; it is an
  automated scan, not a listing [P]. **Linux**: an AppImage or a binary, no prompt; stock GNOME shows
  no tray icons without an extension [K], so the app must be usable without a tray; Flatpak's
  sandbox hides the host's processes and the Wine prefix [K].
- **One identical file for every member.** A per-member build has a hash nobody else downloads, can
  never gain reputation, and signed would need signing on the hub server [K][P: "Do not modify signed
  files"]. Zero-typing setup comes from beside the file: a "Connect this PC" link on the hub page
  (`foreverfriends://...` with the address and a one-time code, confirmed in the app so that a link
  sent by someone else cannot attach the PC to their account), or a code in the file name (fragile
  when the browser appends " (1)").
- **A first-run window is needed**: Windows 11 puts a new tray icon in the overflow by default [K],
  so a tray-only program looks like nothing happened.
- **Updates**: verified against an Ed25519 key whose public half is compiled into the app, the
  private half signing in CI or offline and never on the hub; fail closed, refuse a lower version.
  Files the app downloads itself carry no Mark of the Web, so SmartScreen does not reappear; Smart
  App Control still judges them. The report fetched the manifest from the hub with the companion's
  signed requests because it thought releases private; with public releases that choice is open.
  Tauri's updater makes the signature mandatory ("This cannot be disabled") and takes custom
  endpoints [P]; whether it can send the companion's signed headers was **not verified**.

### 7.2 How the options were found, and the long list

The space was built from the jobs (watch one file, list processes, speak HTTP, write two folders,
show a tray icon, ask one or two things once), from what shipped programs of this shape use
(WeakAuras Companion is Electron [P]; Tailscale carries `fyne.io/systray` [P]; Ollama's desktop app
is Go with a React UI [P]; Syncthing Tray is Qt [P]; WowUp, CurseForge, Raider.IO and Warcraft Logs
are Electron [K]), and from the registries for liveness (dead ends: `native-windows-gui` 2022,
`lxn/walk` 2021, `getlantern/systray` 2023, Flutter's `system_tray` 2023, `cacao` 2023) [P].

Of 49 options, the ones that matter here:

| Option | Verdict |
|---|---|
| Rust + Tauri 2.12 | **Ranked first** (7.3) |
| C# + Avalonia 12 | **Second.** Own renderer, Fluent look, tray built in, best headless tests (`Avalonia.Headless`) [P]; tray menu on Windows drawn by Avalonia, not Windows (`TrayIconImpl.cs`) [P]; 60 to 120 MB resident [K] |
| No framework: tray library + system dialogs (Go `fyne.io/systray` + `ncruces/zenity`, or Rust `tray-icon` + `muda` + `rfd`) | **Third.** Smallest (10 to 20 MB [K]), nothing can look foreign; no window to grow into, Mac and Linux release plumbing by hand. Go's FAQ calls antivirus false positives "a common occurrence, especially on Windows" [P] |
| Go + Wails 3.1 | Tauri's twin in Go, with MSIX packing and a Windows build without cgo built in [P]; **3.0.0 tagged 2026-10-26** after years of alpha [P], one lead maintainer [K]. Not ranked on three weeks of stability; re-compare after six quiet months |
| Electron 45 | Fourth by a clear distance. What every WoW companion uses; about 373 MB on disk and 110 to 280 MB for an empty window [P], and a browser to re-release several times a year |
| C# WinForms on .NET 10 | The most native on Windows, Windows only; first if Windows is declared the only system with an app |
| wxWidgets / `wxdragon` | The strongest native-widget cross-platform option; C++ and CMake, weak headless tests, the Rust bindings one maintainer's [P] |
| Local web page on loopback (Syncthing's shape) | "A browser tab is the settings" is what the owner called home-built; needs a hardened loopback server. No |
| Python + `pystray`, Java SWT/Swing, Tcl/Tk, Kotlin Compose | `pystray` last released 2023, frozen Python the worst antivirus case [K]; a resident JVM of 150 MB and up, tray blurry or missing on Linux [K]. No |
| Flutter, Fyne, Slint, egui and the like | Community-plugin tray and autostart, a visibly foreign look, or too young. No |
| PowerShell + scheduled task, Overwolf, PWA, shipping the addon through CurseForge | "Paste this into PowerShell" is how malware spreads; a platform with advertising; a page cannot watch a file; two installs. No |

### 7.3 Tauri 2, tray-first

- **Draws with** the system's menus for the tray (`muda`, `tray-icon`: a Win32 menu, `NSMenu`, GTK)
  and the system's dialogs (`rfd`); a window is HTML in the system web view (WebView2, part of the
  OS on Windows 10 from 2018 and 11 [P]; WKWebView; WebKitGTK).
- **What a friend could notice** is only what the page gets wrong in a window: a white flash while
  the web view starts, selectable text, a browser context menu, a non-system font. Fixable, but work
  a native toolkit does not need, to be checked on a real PC.
- **Parts**, all first-party and released within five weeks of 2026-11-14 [P]: tray and menus built
  in, `tauri-plugin-single-instance` 2.5.5, `-autostart` 2.7.0, `-deep-link` 2.6.3, `-dialog` 2.8.4,
  `-notification` 2.8.1, `-updater` 2.14.1. On Linux a left click on the tray icon emits no event;
  the menu still opens [P].
- **Footprint** [P, Elanis benchmark, CI machines, noisy]: an empty release build about 3 MB. With a
  window open on Windows, 187 to 312 MB across its processes and a 650 ms start, **worse than
  Electron's 110 to 279 MB and 195 ms**, because WebView2 is a browser too; on macOS and Linux 76 to
  96 MB. "Light" holds on Windows only while no window exists, which is why the app is tray-first.
  **Whether the web-view processes exit when the last window is destroyed was not found in any
  source.**
- **Packaging**: NSIS and MSI, DMG, AppImage, deb, rpm; not MSIX itself (Microsoft's winapp CLI, in
  public preview, has a Tauri guide for that) [P]. A Windows NSIS build cross-built from Linux works
  "with caveats" [P].
- **Testing by agents on Linux**: `cargo test` for the core; the window is a page a headless Chromium
  shows as WebView2 would (same engine family); `tauri-driver` for the whole app.
- **Maturity**: 2.0 stable since October 2024, 2.12.5 on 2026-11-06, about 112,000 stars, 18 million
  crate downloads in 90 days [P]; a version 3 will cost a migration.

### 7.4 The case against, and what would flip it

**The strongest argument against Tauri**: little of what it adds is needed to fix the friend's
journey. Setup moved to the hub removes the typing; a certificate, or nothing, decides the
warnings; the existing Go program behind an ordinary installer with a 200-line updater gets nearly
the same result sooner, without rewriting 3,500 lines that have never run on a PC. What Tauri then
adds is a window, and on Windows that window has the weakest evidence. If it opens once at setup
and almost never again, the owner pays a second language and toolchain for a screen a system
dialog could show. (Weaker without the Store: Tauri brings the per-user installer, link handler,
autostart and signed updater as maintained parts.)

| Observation | Then |
|---|---|
| The Store's private audience is accepted and Microsoft certifies the app | The package supplies autostart, link handler and updates; no framework, in Go, becomes the cheapest complete answer |
| A friend's PC has Smart App Control on | An unsigned build cannot run there; a certificate stops being optional |
| A destroyed Tauri window leaves web-view processes behind, or the window looks like a page however styled | Avalonia first |
| The app needs no window beyond a confirmation, now or later | No framework first, in Go as it is |
| Windows is declared the only system with an app | WinForms on .NET 10 first |
| Wails 3 passes six quiet months | Re-compare it with Tauri |

**The one-hour experiment on the owner's Windows PC** (CI builds the files first): (1) download an
unsigned build with Edge and Chrome, write down every prompt, repeat with a build one byte
different, and note whether Smart App Control is On, Evaluation or Off; (2) install a per-user NSIS
build, click a `foreverfriends://` link, sign out and in, uninstall from Settings; (3) a Tauri tray
stub: memory with no window, count `msedgewebview2.exe` processes with the window open, close it and
see whether they exit, time the second opening, judge the window beside Windows Settings. Not in
the hour: ask one CA what documents it accepts from the owner's country and whether its cloud
signing works from GitHub Actions. Piece 1's "Verify on the PC" sheet (`docs/HANDOFF.md`) carries
this.

The report also read the two earlier ones (a loopback setup page; Wails 3 first): their constraints
"standard library only, no cgo, one Ubuntu runner" were the project's rules, not the friend's
needs, and on a blank page the second's own text put Tauri first or second.

### 7.5 What could not be verified

Blocked hosts that day: learn.microsoft.com, blogs.windows.com, support.microsoft.com, tauri.app
and v2.tauri.app, docs.avaloniaui.net, signpath.org, shop.certum.eu, go.dev, docs.rs, lib.rs,
brew.sh, textslashplain.com and others; documentation was read from its GitHub copy where one
exists. **Not verified**, in the order the ranking leans on it:

1. Whether the Store would certify this app (one private group, a self-hosted site, `runFullTrust`,
   starts at logon, writes into another program's folder).
2. Whether a packaged program writes into the AddOns folder and sees the game's process as an
   unpackaged one does.
3. Whether a Tauri app with its last window destroyed drops back to the Rust process alone on
   Windows, and its real memory and start time on a PC.
4. Avalonia's memory, size and start time (knowledge only).
5. That Windows 11 hides a new tray icon in the overflow by default.
6. Whether Smart App Control, Edge and Chrome treat a file signed by an unknown individual
   differently from an unsigned one.
7. Prices, accepted documents per EU country and CI-usable cloud signing for individual
   certificates (the quoted prices disagree with each other).
8. Whether Tauri's updater can send the companion's signed headers.
9. What SmartScreen, the browsers and Defender transmit about a downloaded file (knowledge).
10. Game detection under Wine or Proton and on macOS: as open as `companion/README.md` says.

### 7.6 Sources (accessed 2026-11-14)

Read from the project's GitHub copy, the site being blocked:

- Microsoft, "Code signing options for Windows app developers", "SmartScreen reputation for Windows
  app developers", "Choose a distribution path for your Windows app", Store policies, visibility
  options and certification: `https://github.com/MicrosoftDocs/windows-dev-docs/tree/docs/hub/apps`
  (`package-and-deploy/`, `publish/`)
- Microsoft, desktop app extensions and how packaged desktop apps run:
  `https://github.com/MicrosoftDocs/windows-dev-docs/blob/docs/hub/apps/desktop/modernize/desktop-to-uwp-extensions.md`,
  `https://github.com/MicrosoftDocs/msix-docs/blob/main/msix-src/desktop/desktop-to-uwp-behind-the-scenes.md`
- Microsoft winapp CLI and its Tauri guide: `https://github.com/microsoft/winappcli`
- Tauri documentation: `https://github.com/tauri-apps/tauri-docs/tree/v2/src/content/docs`;
  `https://github.com/tauri-apps/tray-icon`, `https://github.com/tauri-apps/muda`
- Wails v3 documentation: `https://github.com/wailsapp/wails/tree/master/docs`
- Avalonia's Windows tray:
  `https://github.com/AvaloniaUI/Avalonia/blob/master/src/Windows/Avalonia.Win32/TrayIconImpl.cs`
- .NET 10 Windows Forms: `https://github.com/dotnet/docs-desktop/blob/main/dotnet-desktop-guide/winforms/whats-new/net100.md`
- Go FAQ on virus scanners: `https://github.com/golang/website/blob/master/_content/doc/faq.md`
- The benchmark: `https://github.com/Elanis/web-to-desktop-framework-comparison`
- Manifests: WeakAuras Companion's `package.json`, Tailscale's `go.mod`, Ollama's `app/`,
  Syncthing Tray

Read directly: Apple's Developer ID and membership pages (`https://developer.apple.com/developer-id/`);
the registries crates.io, proxy.golang.org, api.nuget.org, registry.npmjs.org and pub.dev for every
version and date quoted.

Search summaries only [S]: the Windows Developer Blog on free Store registration (2025-09-10,
2026-05-07); Microsoft Q&A on Azure Artifact Signing for EU individuals; SignPath's terms; Certum's
and SSL.com's certificate pages; Microsoft's Smart App Control FAQ and coverage of enabling it
(2026-04); Edge's download-warning documentation; Apple's help page on unidentified developers;
Homebrew 5.0 and 6.0 on Gatekeeper. The full list of URLs is in the report.

## Open questions (max 8)
1. Does the launch client still fail to load SavedVariables, and is Blizzard's fix scheduled?
2. Are `ReplicateItems` results usable (non-secret) by an untainted addon on Forever?
3. Are Forever AH items commodities or items, and does `SendSearchQuery` need both paths? ([#45](https://github.com/mhealy-dev/forever-armoury-addon/issues/45) leaves it open.)
4. Does the packager accept Forever as a game type, and what Interface/flavour does CurseForge need?
5. Is wow-ui-sim Forever profile stable enough for CI, and does it stub `C_AuctionHouse`?
6. Realm auction count and per-scan file size on a Forever realm.
7. What does Blizzard's current addon policy say about scheduled or automated scans?
8. How does Auctionator's Forever build scan; can we copy its pattern?

## Sources I could not open
Network policy blocked these hosts (name resolution failed); claims attributed to them come from search snippets only:
- warcraft.wiki.gg (TOC format, API pages, Saved variables, ReplicateItems and GetReplicateItemInfo pages)
- www.curseforge.com (WoWUnit, AuctionHouse Forever, Auctionator, ForeverSVBridge listing)
- us.forums.blizzard.com (SavedVariables thread)
- wowforever.ninja, wowforeverguides.com, wowforeverinfo.com, wowforeveraddons.com, mmonster.co, wowforeverbuilds.com (guide sites)
- Wowpedia/fandom, wowinterface.com, addonstudio.org (old API references)
- Warcraft Tavern, Icy Veins, GamesRadar (Q&A coverage)
- Not tried: Auctionator's GitHub source, Blizzard's addon policy page, the FriendlyForever `.luacheckrc` and `spec/support/wow.lua` files, wowlua.dev.
Other pages (github.com repos, raw.githubusercontent.com) opened fine.
