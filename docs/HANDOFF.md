# Handoff — 2026-11-09, first session

## State

- Repository scaffolded locally and (once the owner created it) pushed: docs, lint and test
  configuration, the simulated client, the feedback-loop scripts, CI. No addon yet beyond the
  MarketScan lane in flight.
- Research: `docs/RESEARCH.md`. Its decisive findings were verified against GitHub sources
  (FriendlyForever's TOC and luacheck config, Blizzard's Forever UI source, Auctioneer-Forever's
  README) on 2026-11-09. The wiki and guide sites were blocked from the cloud session when the
  note was written; the "Forever Friends" cloud environment now allows warcraft.wiki.gg.
- Toolchain in the cloud environment: Lua 5.1 and luacheck install from apt and LuaRocks;
  busted is blocked on its `lua-term` dependency, whose source download goes through
  codeload.github.com, which the proxy still refused at the end of the session. Retry; if it
  works, save the install lines as the environment's setup script.

## Day-one checks in the client (owner, five minutes, before any addon code is trusted)

1. `/dump select(4, GetBuildInfo())` → expect `16001`.
2. The addons folder: expect `World of Warcraft\_forever_\Interface\AddOns`.
3. Do any addon's settings survive a logout and login? (The beta did not load SavedVariables.)
4. `/dump C_AuctionHouse.ReplicateItems` at the auction house → expect `function`, not `nil`.
5. After one scan with MarketScan: the row count and the size of `MarketScan.lua` in
   `WTF/Account/<ACCOUNT>/SavedVariables/`.

## Open

- Create the GitHub repository `Atomtomate/wow-forever-addons` (private) and push `main`; the
  session could not create it (permission classifier).
- Add the checkout folder on the PC under Remote Control so a session can read `data/`.
- MarketScan draft PR: review, then the day-one checks above with it installed.
