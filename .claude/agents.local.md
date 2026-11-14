# `agents.local.md`: this repository's answers to the generic fleet

The agents in `.claude/agents/` are the owner's shared fleet (`Atomtomate/agents`, a git
submodule) and know nothing about this repository. A definition that says "read
`../agents.local.md`" means this file. It points at the file that owns a fact instead of
copying it, and it is short on purpose: every agent reads it cold, every run.

Mounted on 2026-11-13 at the owner's direction ("we have this agent fleet ... that is used
for software products"), when the repository grew a Go program beside the Lua addons. What is
taken from the fleet: its agents, the plan before code, the four-reviewer pass with reports on
disk, friction notes. What is not: the tracker's scripts, boards, hooks and close block. This
repository stays lightweight (`CLAUDE.md`).

## File ownership

| Agent | Owns | Never touches |
|-------|------|---------------|
| `impl-frontend` | `addons/` (each addon, its README and its `spec/`), `spec/support/wow.lua`, `.luacheckrc`, `.busted` | `companion/`, `go.mod`, `embed.go` |
| `impl-backend` | `companion/`, `go.mod`, `embed.go`, `companion-app/` (D9), `tools/`, `Makefile`, `.github/`, `.githooks/` | Lua under `addons/` and `spec/` |
| `impl-database` | nothing: this repository has no schema | |

"Frontend" here is what runs inside the game; "backend" is what runs beside it. The one file
both need is an addon's `.toc`: `impl-frontend` owns it, and the companion only reads it (the
files it lists are what gets installed). `docs/CONTRACT.md` and `docs/DECISIONS.md` are
direction: the session changes them, a specialist proposes a change in its plan.

## The contract, and the order

`docs/CONTRACT.md` is the contract: what the addon, the companion and the hub exchange. The
order is **contract, then hub, then companion and addon**. The hub is another repository
(`Atomtomate/wow-forever-hub`, private); its side is planned there. A format changes in
`docs/CONTRACT.md` before any party's code does, and a removed, renamed or re-typed field is
breaking for the other two parties: say so.

## The standard the architecture reviewer holds code to

- Lua: `CLAUDE.md`, "Working agreement" and "Conventions". Pure logic sits in files without
  client globals; client-facing code is thin; one addon never loads another's Lua.
- Rust (the companion app, `companion-app/`, D9): Tauri 2, tray-first. The rules the reviewer
  holds it to are written by piece 1 and piece 2 of the plan in `docs/HANDOFF.md`; until then the
  Go rules below apply by analogy (logic in a library crate with no Tauri dependency, tested
  without a PC; code for one operating system behind one small function).
- Go (the companion, `companion/`): the reference implementation and test oracle until the
  companion app reaches parity, then removed (D9). Standard library only, as written.
  `companion/` holds `package main` and stays thin; logic lives in packages under
  `companion/internal/` that do not import each other in a cycle and do not reach for the
  network, the clock or the file system except through what the caller passes in, so they are
  tested without a PC. For writing files, a root directory passed in counts as that (Go's
  `io/fs` is read-only), and a test runs in a temporary one. Code for one operating system sits
  in `_windows.go` / `_other.go` files behind one small function each.
- There is no layering script. Read the imports.

## Where direction and the record live

- What this is and why: `README.md`, `docs/DECISIONS.md` (append-only; a decision changes by a
  new entry), `docs/RESEARCH.md` (facts marked by how they were checked).
- What is out of scope: `docs/IDEAS.md`. An entry marked later or parked is not to be built
  early; say so if a change does.
- What is in flight: `docs/HANDOFF.md`, one file, replaced by the session that closes. It is
  this repository's handoff; there is no `docs/handoffs/` directory, no roadmap, no board.
  Open pull requests: `gh api repos/Atomtomate/wow-forever-addons/pulls`.
- Friction notes: `docs/friction/`, rules in its README.

## Review reports and modes

A reviewer's brief is written by the session and names the head commit, the base, the command
that prints the diff, what is already known to pass, and the report path:
`docs/reviews/<branch with "/" as "-">/<agent>.md`, or `<agent>-plan.md` for a plan read.
Reports are at most 150 lines, name files in backticks, and are committed with the fold that
answers them.

| Mode | When | Plan | Who builds | Pass |
|------|------|------|-----------|------|
| small | one addon or one Go package, no change to `docs/CONTRACT.md` | a paragraph from `impl-director` | the session or one specialist | four reviewers |
| large | a format changes, or more than one party is touched | `impl-director`'s plan, read by `pr-direction-review` and `pr-architecture-review` before code | specialists, each in its own worktree | four reviewers |

A question only the owner can answer is asked, in either mode.

## Invariants the reviewers enforce

The authority is `CLAUDE.md` ("Facts that do not move") and `docs/DECISIONS.md` D5, D7, D8.

- Read, never act: no protected call, nothing that spends, moves or automates (D5).
- SavedVariables is an addon's only output. Inside the game folder the companion reads one
  SavedVariables file per account and writes only inside the two ForeverFriends addon folders;
  outside it, it keeps its own config folder, program copy and autostart entry and nothing
  else (D7, `docs/CONTRACT.md` §5; the app's installer folder and data folder are
  added to §5 before a release reaches members, D9).
- Nothing non-free in this repository or a release, no hub address and no instance data (D9).
- Only what `docs/CONTRACT.md` lists crosses the wire; its §5 says what never does.
- Text typed on the hub reaches the game only under the escaping rule of `docs/CONTRACT.md`
  §2, and the addon treats that data as untrusted.
- Code survives a secret value wherever the client may return one.
- What no test can prove is written down as "verify in client" or "verify on the PC" with the
  exact line to run, never claimed as working.

## Toolchain

- Lua: `make lint test` (luacheck and busted, the same as CI and the pre-commit hook).
- Rust (`companion-app/`): the `make` targets are to be added by piece 1 (D9).
- Go: `go vet ./... && go test ./...` from the repository root, and
  `GOOS=windows GOARCH=amd64 go build ./companion` to prove the Windows build compiles.
- CI is `.github/workflows/ci.yml`. There is no running stack for `ops` and no rig for `qa`:
  the game runs only on the owner's PC.
