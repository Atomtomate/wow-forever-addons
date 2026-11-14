# Tooling research: a code graph and a decision knowledge base

Written 2026-11-14 at the owner's request: *"have another session track the code via graphify
or a similar tool (quick research what works best for LLMs these days) and also find a graph
based knowledgebase software we will use for tracking decisions and information about this
project."* Quick research on purpose: enough to decide, not a survey.

How each claim was checked:

- **[ran]**: run in this session's cloud container against this repository (or a scratch clone
  of it).
- **[read]**: read in a primary source (the project's repository, documentation or package
  registry page) fetched in this session.
- **[snippet]**: seen only in a search result, page not fetched. Treat as unverified.
- **[blocked]**: the network policy refused the page (dev.to and arxiv.org were blocked).

## Summary

| Job | Recommendation | Decision needed from the owner |
|-----|----------------|--------------------------------|
| 1. Code graph | **Adopt nothing yet.** Revisit when the Rust app or the hub grows, using a three-question A/B with `codebase-memory-mcp`, the best of the tools tried. | **Answered: accepted** (below). |
| 2. Knowledge base | **Keep git Markdown as the source of truth; split decisions into one file each with typed front-matter links, and view them in Obsidian.** Basic Memory is the later option for agents. | **Answered: accepted as the start, not the end state** (below). Still open: reading on a phone, and front matter beyond decisions. |

### The owner's answers (2026-11-14)

The owner answered in the coordinator session (`session_01BoZz39J5gTi1eky3ESzWYZ`), and that
session relayed both answers here.

1. **Code graph.** Asked "Is skipping the code graph for now acceptable, given that you asked
   for one?", the owner answered: **"yes"**. Nothing is adopted now, and the revisit trigger in
   job 1 stands. This answer is as relayed; the transcript page read here did not reach it.
2. **Knowledge base.** Asked "is Markdown in git with one file per decision, viewed in
   Obsidian, the one you want?", the owner answered: **"For a start maybe, I want a more
   systematic solution that interacts optimally with LLMs i[n] the future"**. This one was
   checked word for word in that session's transcript. So the first choice is the starting
   point, not the end state. [Where this is meant to go](#where-this-is-meant-to-go) says what
   the end state needs and what to keep true now. The split of `docs/DECISIONS.md` is its own
   small piece, after pull request #5 (which edits that file) has merged; it follows
   [the schema below](#the-schema-the-split-follows).

---

## Job 1: a code graph the agents can use

### How the candidates were found

Searches: "graphify code knowledge graph Claude Code github", "code knowledge graph MCP
server tree-sitter local Claude Code 2026", "Claude Code LSP plugin code intelligence
go-to-definition find references", "aider repomap MCP server standalone RepoMapper tree-sitter
ctags", and "independent benchmark code graph MCP vs grep". Versions were then checked on PyPI
and npm. The approaches covered were code knowledge graphs, LSP symbol indexes, repository
maps, MCP servers that expose any of these, and what Claude Code already has. Embedding-only
search indexes were left out: all of them either send code to an embeddings service or bundle
a local model inside a graph tool. `codebase-memory-mcp` bundles one.

### What "graphify" is

The tool built for coding agents is **`safishamsi/graphify`** (also published as
Graphify-Labs). It is not the same thing as graphify.com. [read]

- Released on PyPI as `graphifyy`, version 0.9.108, uploaded 2026-11-12; Apache-2.0 and MIT.
  [read] Its GitHub star count (about 147k) is implausible, so it is not used as a signal.
- It parses code locally with tree-sitter and needs no model to do so. An optional "semantic
  pass" over documents, PDFs and images does need an LLM key: Gemini, Anthropic, OpenAI,
  Ollama, or the `claude` CLI. [read]
- Agents reach it through a `/graphify` skill, a PreToolUse hook on Grep and Glob, a CLI, and an
  optional MCP server. It writes `graphify-out/graph.json` and a report into the repository.
  [read]

### The candidates

| Tool | What it is | Measured savings | Lua · Go · Rust · JS · Markdown | Local, no key | In a cloud container | Agents reach it by |
|------|-----------|------------------|-------------------------------|---------------|----------------------|--------------------|
| Built-in Grep, Glob, Explore agent | text search | the baseline | all | yes | nothing to install | built in; subagents too |
| Official LSP plugins (`gopls-lsp`, `lua-lsp`, `rust-analyzer-lsp`, `typescript-lsp`) | `LSP` tool: definitions and references | none published | all but Markdown | yes | **cloud sessions do not start plugin language servers** [read] | terminal sessions only |
| `codebase-memory-mcp` (DeusData), MIT, 0.12.1 (2026-10-16) | tree-sitter + LSP graph in SQLite; 17 tools including `trace_path` and `detect_changes` | vendor preprint: about 10× fewer tokens and 2.1× fewer tool calls over 31 repositories [snippet] [blocked]; no independent check | all; Lua confirmed [ran] | yes; embeddings model bundled | pip/npm wrapper downloads a **330 MB** binary from GitHub releases, which worked here; indexing took 2.3 s [ran] | MCP or CLI |
| graphify (`graphifyy` 0.9.108) | AST graph as a JSON file plus a report | vendor "71.5×" on a mixed corpus; a small n=6 test scored 82% against 70.8% of key facts [snippet] | all; Markdown headings without a model | code yes; the documents pass needs a key | `uv tool install`, 164 MB, built in 2.7 s [ran] | skill/hook, CLI, MCP |
| Serena (`serena-agent` 2.1.0), GPL-3.0 | LSP-backed symbol tools | none published | all five | yes | uv, plus each language server separately (lua-language-server comes from GitHub) [read] | MCP |
| RepoMapper / `repomap-mcp` | aider-style ranked repository map | none published | the tree-sitter set | yes | pip or `npx` [snippet] | MCP or CLI |
| CodeGraph (`@colbymchenry/codegraph` 1.6.2) | SQLite graph | vendor claims; a third party reproduced the cut in tool calls but not the cut in cost [snippet] [blocked] | not checked | yes | npm | MCP |

Neither graph tool that was run commits its index. `codebase-memory-mcp` keeps its index in
`~/.cache/codebase-memory-mcp/` and a file watcher keeps it current. graphify writes
`graphify-out/` in the repository and is kept current by a git hook or `graphify watch`. Both
rebuild in about 3 seconds at this size, so a rebuild each session costs nothing. The download
is what costs time. [ran]

### Measured on this repository

Three real questions, answered once with grep and once with each graph tool. [ran]

| Question | grep | graphify | codebase-memory-mcp |
|----------|------|----------|---------------------|
| Who calls the fake hub's `Approve()`? | 1 command, **4 call sites, exact** (`app_test.go`, `hubapi_test.go`, `linkflow_test.go` ×2) | 1 command, **found 1 of the 4**; the Go call edges are marked "INFERRED" | 1 command with `--include-tests`, **27 callers to depth 2**: a superset, not checked one by one, and every name carries the full project path |
| Who calls `ns.pushLog`? | 1 command, **2 calls, exact** (`Core.lua:23`, `snapshot_spec.lua:186`) | **missed the spec caller** | **4 inbound edges**; the two beyond grep's were not traced |
| Which Go files mention SavedVariables? | 1 command (`grep -rli savedvariables --include=*.go`), 5 files including tests | not tried | not tried |

At this size grep plus one file read is cheap and exact. The graph tools cost the same number
of tool calls and answered less precisely. They also add context of their own: 17 tool schemas
for `codebase-memory-mcp` in every session that loads it, or a hook that steers every Grep for
graphify. The one independent benchmark found (60 tasks) has tuned grep beating a graph server
by 9 F1 points; graphs won only on structural lookups. [snippet] [blocked]

### Recommendation: adopt nothing yet

The repository holds about 5,700 lines of Go, 2,100 of Lua and 3,000 of Markdown. That is
small enough that no tool showed a saving, and the one that recalled best,
`codebase-memory-mcp`, adds a 330 MB download to every fresh container plus a third-party
binary to trust.

**When to revisit:** run a three-question A/B like the one above with `codebase-memory-mcp`
when either of these happens:

- the Rust + Tauri companion that replaces the Go one (being added in a parallel session as
  `companion-app/`, per the brief for this work; not on `main` yet) or the hub passes about
  20,000 lines;
- session transcripts show agents re-reading the same files to trace calls.

Adopt it only if it then saves tool calls *and* answers at least as well as grep.

**Strongest argument against waiting:** `trace_path` and `detect_changes` map exactly onto
"what calls this" and "what does this change touch". That question grows with the Rust app
from day one, and installing early would build the habit.

**What adopting it would take, here or on the private hub.** Both tools work offline once
installed and upload nothing. The one exception is graphify's optional semantic pass over
documents, which sends them to an LLM provider. `graphify update .` on code does not use it.
Never switch it on for the private hub without asking.

- A setup script on the cloud environment: `uv tool install codebase-memory-mcp &&
  codebase-memory-mcp --version`. The `--version` call fetches the binary.
- `github.com` release downloads on the network allowlist. That host was reachable from this
  session; check it on the hub's environment.
- `.mcp.json` in the repository, declaring the server as `codebase-memory-mcp`.
- Index at session start: `codebase-memory-mcp cli index_repository --repo-path "$PWD"`, or let
  the watcher do it.
- The `mcp__codebase-memory-mcp__*` names added to the `tools:` list of any fleet agent
  definition that restricts its tools. Subagents otherwise inherit MCP tools.
- The hub needs the same, plus a check that its Node/JS parses well, which was not tested here.

Commands that worked in this session, for the revisit:

```sh
uv tool install codebase-memory-mcp && codebase-memory-mcp --version
codebase-memory-mcp cli index_repository --repo-path "$PWD"
codebase-memory-mcp cli search_graph --project <name printed by index> --name-pattern '.*pushLog.*'
codebase-memory-mcp cli trace_path --project <name> --function-name Approve --direction inbound --include-tests true

uv tool install graphifyy && graphify update .   # writes graphify-out/; gitignore it
graphify explain "ns.pushLog()"
```

---

## Job 2: a graph-based knowledge base for decisions

### How the candidates were found

Searches covered linked-notes tools over Markdown, self-hosted wiki and knowledge-graph
servers, LLM memory graphs, ADR tooling and Logseq's state. Queries included "Basic Memory MCP
markdown knowledge graph", "Obsidian MCP server Claude Code vault", "TriliumNext MCP server
relations", "Graphiti MCP server FalkorDB", "SilverBullet MCP server", "log4brains ADR
supersedes front matter", "Logseq DB version 2026", "Obsidian Bases frontmatter mobile git
sync", "Obsidian Git plugin mobile limitations" and "Cognee knowledge graph MCP self-hosted".

Outline and Wiki.js were not examined further: they have no typed links, and their export is
no better than TriliumNext's. An issue tracker used as the graph was dropped too: it adds a
system without adding typed links. Release dates in directory listings disagreed with each
other, so treat versions as approximate.

### The candidates

| Candidate | Where data lives | Typed links | For agents in cloud containers | History and review | Export | Upkeep, five-year odds |
|-----------|------------------|-------------|--------------------------------|--------------------|--------|------------------------|
| **Git Markdown, one file per decision, front matter** | this repository | by convention: `supersedes:`, `depends_on:` | nothing to set up; already readable | pull requests | it is the format | none; certain |
| **Obsidian** (+ core Bases, Breadcrumbs plugin) as a viewer | a local vault, which can be the repository | Breadcrumbs reads typed edges from front matter (v4.15, maintained May 2026) [read] | not needed: agents edit the files | through git | Markdown | free but proprietary; large community; high odds |
| **Basic Memory** (MCP over a Markdown folder) | `.md` files plus a rebuildable local SQLite index; an optional cloud plan at $15/month | yes, as `- supersedes [[D3]]` lines under a `## Relations` heading in the body; plain links count as `links_to` [read] | runs locally on the clone via `uvx` from PyPI; the vendor's own MCP server; AGPL, about 4.6k stars, v0.2x [read] | through git | Markdown | low upkeep; small company with an open-source core; moderate odds |
| **TriliumNext** | self-hosted Docker server, SQLite | native relations and relation maps | server on the allowlist plus an ETAPI token; MCP servers are community prototypes (about 65 stars, "back up first") [snippet] | note revisions, not PRs | Markdown/HTML export | AGPL, about 39k stars; a server and backups to run [read] |
| **SilverBullet** | self-hosted server over a folder of `.md` files | front-matter attributes and queries | server on the allowlist; community MCP servers around v0.6 [snippet] | only if the folder is in git | Markdown | MIT; small team; medium odds |
| **Logseq** | the file-based version is in maintenance; DB 2.0 (beta, mid-2026) makes SQLite canonical, with one-way Markdown export [snippet] | properties | a poor fit | none | one way | in the middle of a migration; risky |
| **Graphiti / Cognee / the reference MCP memory server** | a graph database (FalkorDB, Neo4j) or a JSONL file | yes, but extracted by an LLM (OpenAI by default) | a server plus an LLM key behind the allowlist; Graphiti's MCP server is labelled experimental [snippet] | no PR-style review | database dump, JSONL | high upkeep; the reference server is a toy [read] |
| **log4brains, adr-tools, AdrMcp** | Markdown in git | supersession written in the ADR body; AdrMcp returns typed edges [snippet] | files | pull requests | Markdown | log4brains is quiet; niche |

### Ranked recommendation

1. **Git Markdown stays the source of truth, one file per decision, typed links in front
   matter.** Add a short generated index file, and a check in `make lint` that every link
   resolves and that every `supersedes` has its matching `superseded_by`. A cold session reads
   the index and then opens only the neighbouring decisions. That needs no server, no MCP and
   no new dependency for any session.
2. **Option 1, with Obsidian as the owner's window** on desktop and phone: open the repository
   as a vault, use Bases tables for status and open questions, Breadcrumbs for the typed edges,
   and the graph view. Agents never touch Obsidian; it only reads the files.
3. **Option 1 plus Basic Memory**, added only if sessions turn out to need graph traversal or
   search over a much larger record. One catch: it reads relations from a `## Relations`
   section in the body, not from front matter, so links would be written twice unless the lint
   keeps them in step.

Not recommended as the primary store: TriliumNext, SilverBullet and the Graphiti-style systems.
Each moves the truth into a server database outside pull-request review, and adds a host and a
token to every cloud environment's allowlist. Graphiti and Cognee also need an LLM API to build
the graph at all.

**Strongest argument against the first choice:** it is not really a graph tool. Typed links
exist only by convention, and git alone gives no pleasant phone interface. On a phone,
Obsidian needs the files synced. The Obsidian Git plugin's mobile support is officially
experimental: no SSH, memory caps, crashes on large repositories. [snippet] In practice the
phone is read-mostly, through GitHub's mobile view, or through Obsidian Sync or Syncthing
keeping a copy current. Obsidian Sync costs about $4–5 a month; that figure was not verified.

**What it adds over today's files:** decisions that can be addressed one by one, with links
that a machine can check and follow. Supersession becomes an edge instead of a dated sentence
inside another entry. "Everything that depends on D7" becomes a lookup. A session reads one
decision and its neighbours instead of the whole file. **What it costs:** one split of
`docs/DECISIONS.md` and a small lint script. A new decision also needs a filename instead of an
appended heading.

**Your other projects:** the same scheme scales to `boardgame-tracker`. Its larger record is
where Basic Memory's search and context tools would pay off first, so pilot option 3 there if
at all. The fleet repository `Atomtomate/agents` is consumed as a submodule, so files with front
matter suit it, and a server-based knowledge base would not.

### The schema the split follows

This is what the piece that splits `docs/DECISIONS.md` should build, after pull request #5 has
merged. It is fixed here so that every later tool reads the same thing.

**Layout**

```
docs/decisions/
  README.md                     # this schema, short; how to add a decision
  INDEX.md                      # generated, never edited by hand
  D1-one-repository.md
  D2-retail-lineage-client.md
  D3-three-testing-tiers.md
  D4-savedvariables-and-remote-control.md
  D5-read-never-act.md
  D6-foreverfriends-first.md
  D7-companion-program.md
  D8-what-crosses-the-wire.md
  D9-<slug>.md                  # whatever pull request #5 records as D9
```

- The file name is `D<n>-<slug>.md`. `<n>` is the decision's number for good: it is never
  reused or renumbered. `<slug>` is lowercase ASCII words joined by hyphens, chosen once and
  never renamed, so links and URLs stay stable.
- A new decision takes the next free number. A decision changes only through a new decision
  that supersedes it, in whole or in part. The record stays append-only, as it is today.
- `docs/DECISIONS.md` becomes a one-paragraph pointer to `docs/decisions/`. Links elsewhere
  that read "D7" keep working, because the identifier does not change.

**Front matter: a closed set of keys**

```yaml
---
id: D7                      # required; equals the file name's D<n>
title: A companion program on each member's PC carries the data   # required; the heading text
date: 2026-11-13            # required; the day it was decided, YYYY-MM-DD
status: accepted            # required; one of: proposed | accepted | partly-superseded | superseded | withdrawn
supersedes: []              # decision ids this one replaces whole
supersedes_part: [D4]       # decision ids this one replaces in part; the body says which part
superseded_by: []           # mirror of supersedes, set on the older decision
superseded_by_part: []      # mirror of supersedes_part
depends_on: [D6]            # decision ids this one assumes
relates_to: []              # decision ids related without dependence
code: [companion/]          # repository paths (files or directories) this decision governs
docs: [docs/CONTRACT.md#5]  # document paths, with an optional #anchor
prs: [4]                    # pull request numbers in this repository
open_questions: []          # one string each, short
---
```

- The body is the decision's existing text, moved unchanged, under `# D7 — <title>`. The
  supersession sentences stay in the body as prose, and the front matter carries the edge.
- After the body comes a `## Relations` block, generated from the front matter by the same
  script that writes `INDEX.md`. It has one line per edge, such as `- supersedes_part [[D4]]`
  and `- depends_on [[D6]]`, and is never edited by hand. That line format is what Basic Memory
  reads as typed relations [read], so moving to it later needs no rewrite.

**Edges of D1–D8 as far as their text states them today:** D7 `supersedes_part: [D4]`, and so
D4 `superseded_by_part: [D7]` with `status: partly-superseded`. Every other edge (D7 on D6, D8
on D7, client-facing entries on D2 and D5) is an inference. The split proposes those for review
in its pull request and does not slip them in as fact.

**The check** (in `make lint`, so the pre-commit hook and CI run it). It is a small script with
no dependency beyond the toolchain the repository already has.

- Every file in `docs/decisions/` other than `README.md` and `INDEX.md` has front matter with
  exactly the required keys, and only keys from the list above.
- `id` matches the file name; ids are unique; `date` is a valid date; `status` is from the list.
- Every decision id in a link key names an existing file, and no decision links to itself.
- The mirrors agree: A has `supersedes: [B]` exactly when B has `superseded_by: [A]`, and the
  same for `_part`.
- Status agrees with the edges: `superseded` exactly when `superseded_by` is non-empty;
  `partly-superseded` exactly when only `superseded_by_part` is non-empty.
- Every `code` and `docs` path exists, without the anchor. Every `prs` entry is a positive
  integer; whether the pull request exists is not checked offline.
- `INDEX.md` and every `## Relations` block equal what the generator writes now.

### Where this is meant to go

The owner wants "a more systematic solution that interacts optimally with LLMs". Per-decision
files are the start. From the research above, the end state has to provide five things that
files alone do not.

1. **Retrieval of a neighbourhood.** "Give me D7 and everything one or two links away", in one
   call, without reading the folder.
2. **Typed links that agents query and write**, through a tool interface (MCP), not by
   editing YAML by hand.
3. **Validation.** Links resolve, mirrors agree and statuses match the edges, whoever wrote
   the change: a session, the owner in Obsidian, or a tool.
4. **Search across the record** by meaning as well as by word, once it is too big to skim.
5. **One base across the owner's projects** (this one, `boardgame-tracker`, the fleet), so a
   decision in one can be found from another. Each project's record still lives in its own
   repository.

**The natural next step on the same files: Basic Memory.** It is an MCP server over a folder of
Markdown that keeps a rebuildable local SQLite index. It reads `## Relations` lines as typed
edges, and its context tool follows them to return a neighbourhood. That covers points 1, 2 and
part of 4. [read: its repository] The files stay the source of truth and pull requests stay
the review gate, because what it writes lands in the same files, which are committed like any
edit.

- **Point 3 stays with the check above.** Basic Memory does not enforce this schema; the CI
  check does, for every writer.
- **Point 5:** whether one Basic Memory instance can serve several repositories' folders at
  once was **not checked**. That is the first thing to find out when the move is considered.
- **What it costs:** it is AGPL, comes from PyPI, and needs a `uvx`/`pip` install in each cloud
  container. Per `CLAUDE.md` it is a new dependency and needs the owner's yes.

Beyond Basic Memory, nothing found in this research justifies a graph database or an
LLM-extracted memory graph (Graphiti, Cognee) for a record this size. Each needs a server, an
LLM key and its own review process.

**What should trigger the move**, any one of these:

- the decision folder passes about 50 files;
- sessions are seen reading the whole folder, or `INDEX.md` plus many files, to answer one
  question about a decision;
- the owner wants to ask a question across projects ("have we decided this before elsewhere?");
- agents start writing decisions often enough that YAML edits by hand become a source of
  errors the check keeps catching.

**What to keep true now, so the move stays cheap:**

- Identifiers are stable: `D<n>` and the slug never change.
- The front-matter schema is fixed. A new key is added here, in this file's successor, and in
  the check, all in the same pull request.
- Link types come from the closed list above; no ad-hoc relation names.
- The `## Relations` block is generated in Basic Memory's line format.
- The CI check runs on every change, so the files are always valid input for whatever tool
  comes next.

### Decisions for the owner

1. ~~Split `docs/DECISIONS.md` into one file per decision.~~ Answered: yes, as the start.
   The split comes with the check and the index, as the schema above says.
2. On a phone: GitHub mobile (read-only), Obsidian with Obsidian Sync (paid), or Obsidian with
   Syncthing.
3. Trying Basic Memory later is a new dependency (AGPL, from PyPI). Per `CLAUDE.md` it waits
   for your yes; nothing is proposed now.
4. Do `CONTRACT.md`, `RESEARCH.md` and `IDEAS.md` get front matter too, or decisions only?
5. ~~(Job 1) The revisit trigger.~~ Answered: adopt nothing now; the trigger stands.

---

## Sources

Fetched [read]:

- https://github.com/safishamsi/graphify
- https://github.com/DeusData/codebase-memory-mcp
- https://github.com/oraios/serena
- https://code.claude.com/docs/en/plugins/code-intelligence
- PyPI JSON for `graphifyy`, `codebase-memory-mcp`, `serena-agent`, `aider-chat`; npm registry
  for `codebase-memory-mcp` and `@colbymchenry/codegraph`
- https://github.com/basicmachines-co/basic-memory
- https://github.com/TriliumNext/Trilium
- https://github.com/SkepticMystic/breadcrumbs
- https://github.com/modelcontextprotocol/servers/tree/main/src/memory
- https://github.com/foambubble/foam

Search snippets only [snippet]; blocked ones are marked:

- https://dev.to/nike-17/i-benchmarked-code-retrieval-for-ai-coding-agents-on-60-tasks-f9h [blocked]
- https://dev.to/harrisonsec/i-tested-codegraph-on-hono-the-tool-call-savings-reproduce-the-cost-savings-dont-389p [blocked]
- https://arxiv.org/abs/2603.27277 [blocked]
- https://www.augmentcode.com/learn/graphify-v099-benchmarks-knowledge-graph
- https://graphify.com/blog/how-to-give-claude-code-a-code-knowledge-graph
- https://glama.ai/mcp/servers/@pdavis68/RepoMapper
- https://dev.to/andrew-ooo/codegraph-review-pre-indexed-knowledge-graph-for-ai-agents-af1
- https://docs.falkordb.com/agentic-memory/graphiti-mcp-server.md
- https://github.com/tan-yong-sheng/triliumnext-mcp
- https://mcpservers.org/fr/servers/Ahmad-A0/silverbullet-mcp
- https://discuss.logseq.com/t/whats-new-with-logseq-db-may-16th-2026/35020
- https://obsidian.md/changelog/2025-05-21-desktop-v1.9.0
- https://community.obsidian.md/plugins/obsidian-git
- https://www.stephanmiller.com/obsidian-git-sync-mobile/
- https://glama.ai/mcp/servers/Atypical-Consulting/AdrMcp
- https://www.github.com/thomvaill/log4brains/discussions/22
