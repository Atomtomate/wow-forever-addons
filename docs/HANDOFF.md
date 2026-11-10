# Handoff — 2026-11-10, first session (closing)

Read this after `CLAUDE.md`. Replace it when the next session closes.

## State

- `main` at https://github.com/Atomtomate/wow-forever-addons holds the framework: research with
  sources (`docs/RESEARCH.md`, written 2026-11-09, decisive claims verified against GitHub
  primary sources), decisions D1–D5, luacheck and busted configuration, CI (`.github/workflows/
  ci.yml`: luacheck + busted on ubuntu), pre-commit hook, and the feedback-loop scripts in
  `tools/` (junction, SavedVariables pull, parser). The parser is tested against a sample in the
  client's own format.
- **Draft PR #1, MarketScan** (`lane/marketscan`, 3 commits): one-button `ReplicateItems` scan,
  per-item min/median/quantity/count packed into SavedVariables, 33 specs on our own simulated
  client (`spec/support/wow.lua`). Reviewed by the coordinator and one independent review;
  findings folded, the dismissed ones with reasons in the PR body. **CI's busted run is the
  first thing to check**: busted could not run in the cloud container (its `term` dependency is
  a GitHub source archive the session's GitHub gate refuses to download; cloning and building it
  needs the owner's say because it runs external code), so the specs were only run through a
  plain-Lua shim.
- Nothing has run in the real client yet. The addon's README lists twelve "verify in client"
  items; the first scan on the owner's realm is the real test.

## In flight

- Nothing is running. The lane worktree in the cloud container was removed after the push.
- Cloud environment "Forever Friends" (personal, for the tracker and hub repositories; add this
  one in its settings) allows luarocks.org, warcraft.wiki.gg, the Blizzard and Steam developer
  hosts. Lua 5.1 and luacheck install from apt and LuaRocks; no setup script saved yet because
  busted's install is incomplete (see above). Lines that worked: `apt-get install -y lua5.1
  liblua5.1-0-dev luarocks && luarocks install luacheck`.

## First steps for the next session

1. Read PR #1's CI result. Green: ask the owner whether it may merge. Red: fix on the lane.
2. Owner's side, once merged and pulled on the PC: `tools\link.ps1 MarketScan`, restart the
   client, open the auction house, press "Scan prices", `/reload`, `tools\pull_sv.ps1
   MarketScan`. Go through the README's verify-in-client list with them; record the answers in
   `docs/RESEARCH.md` (a dated "Observed in client" section) and change `CLAUDE.md`'s facts if
   the client disagreed.
3. Remote Control: ask the owner to add the PC checkout under Remote Control, then read
   `data/MarketScan.json` from there. That closes the loop (D4).
4. Then pick from `docs/IDEAS.md` with the owner (price history waits on the SavedVariables
   load-back answer; hub integration waits on a decision about transport).
