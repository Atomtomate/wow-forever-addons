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
