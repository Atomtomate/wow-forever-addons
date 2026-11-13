# Forever Friends addons

Addons for a small friend group playing **World of Warcraft: Forever**, developed with an AI
assistant in the loop. Each addon lives under `addons/<Name>/`; lint, tests and the feedback
loop are shared.

- **ForeverFriends** links the game to the group's site (the hub): it records the character
  being played and shows who is whose character. A small **companion** program on each
  member's PC installs it and carries the data, because an addon cannot reach the network
  (`docs/DECISIONS.md` D6 to D8, formats in `docs/CONTRACT.md`).
- **MarketScan** records auction-house prices. Merged, not yet run in the client.

## Facts that shape everything (verified 2026-11-09, see `docs/RESEARCH.md`)

- Forever runs the **modern addon API** (retail 12.1.5 lineage), not the Classic one. Interface
  number **16001**. The auction house is `C_AuctionHouse`; a full scan is `ReplicateItems()`.
- SavedVariables are the only thing an addon can write. The beta client wrote them but did
  not load them back; whether the launch client does is checked on day one (`docs/HANDOFF.md`).
- An addon may read anything; buying, bidding, posting and cancelling are protected calls and
  none of ours ever calls them.

## Layout

| Path | What |
|---|---|
| `addons/<Name>/` | one addon: `<Name>.toc`, Lua files, `spec/` with its busted tests |
| `companion/` | the companion program (Go, standard library only): links a PC to the hub, installs ForeverFriends, uploads characters, reports that the game runs. Its own README has setup, the on-disk layout and the "Verify on the PC" list |
| `go.mod`, `embed.go` | the Go module, and the package through which the companion carries the addon's files inside its executable |
| `docs/CONTRACT.md` | the formats the addon, the companion and the hub exchange; their only home |
| `spec/support/wow.lua` | a small simulated client for busted: frames, events, timers, SavedVariables |
| `tools/link.ps1` | Windows: directory junction from the game's AddOns folder into this checkout |
| `tools/pull_sv.ps1` | Windows: copy an addon's SavedVariables file into `data/` after a `/reload` |
| `tools/parse_sv.py` | turn a SavedVariables file into JSON and a short summary |
| `.luacheckrc`, `.busted` | lint and test configuration, shared by every addon |
| `.github/workflows/ci.yml` | luacheck and busted, and the Go checks on Linux and Windows, on every push and pull request |
| `.github/workflows/release.yml` | builds the companion for Windows and Linux when a version tag is pushed |
| `docs/` | research, decisions, handoff, ideas |

## Working on an addon

Requirements: Lua 5.1 (or LuaJIT), luacheck, busted. On Debian/Ubuntu:
`apt-get install lua5.1 liblua5.1-0-dev luarocks && luarocks install luacheck busted`.

```bash
make lint       # luacheck over addons/ and spec/
make test       # busted
```

For the companion: Go 1.24 or newer, nothing else.

```bash
make golint     # gofmt, go vet, and go vet for Windows
make gotest     # go test ./...
make check      # all four
```

In the game, after linking (`tools/link.ps1 MarketScan`), edit, then `/reload`. A new addon
folder needs a full client restart; edits to existing files do not.

## Releasing the companion

A release is a tag, and you push it. The tag must be `v` plus the `## Version` line of
`addons/ForeverFriends/ForeverFriends.toc` (`v0.1.0` for `## Version: 0.1.0`); the release
workflow refuses any other. The addon and the companion share that one number, because the
executable carries the addon: a fix to the companion alone still means raising that line first.
The workflow builds the Windows and Linux files and publishes them as a GitHub release, which
is where the hub's `/companion` page sends members.

## Feedback loop

1. `tools/link.ps1 <Name>` once per addon: the game loads the checkout directly.
2. Play, `/reload` (or log out): the client writes `WTF/Account/<ACCOUNT>/SavedVariables/<Name>.lua`.
3. `tools/pull_sv.ps1 <Name>` copies that file to `data/<Name>.lua` (git-ignored) and runs
   `tools/parse_sv.py`, which prints a summary and writes `data/<Name>.json`.
4. The assistant reads `data/` through Remote Control on the PC, or you paste the summary.
5. Lua errors: enable `/console scriptErrors 1` or install BugSack; its SavedVariables file
   can be pulled the same way.
