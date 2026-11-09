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
