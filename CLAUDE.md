# Forever Friends addons

World of Warcraft: Forever addons for a friend group of 5–15. `README.md` is the manual,
`docs/RESEARCH.md` the research with sources, `docs/DECISIONS.md` the decisions,
`docs/IDEAS.md` the backlog, `docs/HANDOFF.md` what the last session left in flight — read it
first. The group's community site is `Atomtomate/wow-forever-hub`; its CLAUDE.md's working
style applies here too (lightweight, decisions written down, the owner merges).

## Facts that do not move until the client proves otherwise

- **Forever is a retail-lineage client** (12.1.5 API, interface `16001`, folder `_forever_`).
  Classic Era API knowledge (`QueryAuctionItems`, `GetItemInfo`) does not apply; use
  `C_AuctionHouse`, `C_Item`, `C_Timer`. Check against Blizzard's Forever UI source
  (`Gethe/wow-ui-source`, branch `forever`, `Blizzard_APIDocumentationGenerated/`) before
  using a function you are not sure of.
- **Secret values exist.** Some API returns cannot be compared, indexed or used in arithmetic
  by addon code; the simulated client in `spec/support/wow.lua` raises where the real one
  would. Code that touches auction data must survive a secret where one may appear.
- **Never call a restricted function.** `PlaceBid`, `PostItem`, `PostCommodity`,
  `ConfirmPost*`, `StartCommoditiesPurchase`, `CancelAuction` and anything that spends or
  moves gold or items. Scanning reads only. This is both Blizzard's rule and ours (D5).
- **SavedVariables is the only output.** No file I/O, no network. Logs go into a capped ring
  table inside the addon's SavedVariables so they come out with the data.
- **A unit's identity can be a secret value too.** `UnitName`, `UnitFullName`, `UnitClass`,
  `UnitRace` and `UnitGUID` are documented as secret when unit identity is restricted
  (`docs/RESEARCH.md` §6). Check before storing or comparing one, and compare inside the
  protected call: `v == nil` on a secret is a comparison too.

## Working agreement

- Each addon is independent: its own TOC, its own `spec/`, no shared Lua between addons unless
  a decision says so. Shared things are tooling only.
- Pure logic (packing, maths, parsing) lives in files with no client globals so busted can load
  them without the simulated client. Client-facing code is thin and tested through
  `spec/support/wow.lua`.
- `make lint test` must pass before a commit (`.githooks/pre-commit`, same as CI), and
  `make golint gotest` as well when a Go file changed. Lint fails on an unlisted global: list it
  in `.luacheckrc` or make it local.
- What a test cannot prove is written into the addon's README as "Verify in client" and into
  `docs/HANDOFF.md`, with the exact `/dump` or `/run` line to try.
- A pull request is a draft only while it is being worked on. Once CI is green and its reviews
  are folded it is marked ready for review, which is what tells the owner to look; a stacked
  one says "merge after #N" in its first line. The owner says merge.
- **One session carries one pull request, from its brief to ready for review, then ends.** Work
  is cut so that each piece finishes inside one session, and the next piece starts from this
  file, `docs/HANDOFF.md` and the contract, not from a carried conversation. A piece that
  cannot finish is committed, what is left goes into the pull request's body, and the session
  stops. `.claude/settings.json` caps a context at 400,000 tokens; that is the backstop, not
  the plan (the owner, 2026-11-13, after one session grew to 900,000). The session that coordinates
  does not do the work itself: it briefs, reviews and decides, and hands edits, pushes and
  bookkeeping to an agent on a cheaper model or to a session of its own.
- Ask before a new library dependency (Ace3 and the like, and any Go module for the companion,
  which uses the standard library only), a SavedVariables format change that loses data, or
  anything that automates play.
- **`docs/CONTRACT.md` is the one home of what the addon, the companion and the hub exchange**,
  and of what the companion may read and write on a PC (its §5). A change to a format changes
  it there first, and the other two parties in the same breath.
- Code copied from another addon carries its licence; prefer writing our own.

## Conventions

- Lua 5.1. Four-space indent, 120 columns, `local` everything, one global per addon: its
  SavedVariables table (`<Name>DB`) plus what the TOC and slash commands require.
- Addon files receive `(addonName, ns)` varargs; share within the addon through `ns`, not
  globals.
- Events through one frame per addon; handlers small and named.
- Everything the user sees is one line of chat or a small frame; nothing nags.
