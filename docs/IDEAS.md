# Ideas and future plans

The durable backlog; entries are updated, never deleted. Status words: **next**, **later**,
**parked** (needs something external), **dropped** (with the reason).

- **MarketScan** — next. One-button full scan of the auction house, per-item minimum buyout,
  median, quantity and auction count, stored compactly in SavedVariables with a timestamp.
- **Price history and tooltips** — later. Needs the client to load SavedVariables back
  (`docs/RESEARCH.md` §1.3); until then every session starts empty.
- **Hub integration** — later. Upload a scan summary to the Forever Friends hub so the group
  sees prices on the site. Needs a decision about what the hub stores and a transport that is
  not the addon (it cannot do network): the pull script posts the JSON.
- **Headless UI simulator in CI** — later. Osso/wow-ui-sim with the Forever profile, for load
  errors; D3 tier 2.
- **Lua Language Server annotations** — later. Ketho's vscode-wow-api for the editor; no
  Forever set yet, retail's is the closest.
- **Publishing on CurseForge or Wago** — later, only if an addon is useful beyond the group.
  The BigWigs packager handles it; whether it knows Forever as a flavour is unverified.
