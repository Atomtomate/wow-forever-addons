# Review of `docs/TOOLING.md` at 86fe07e

The fleet submodule (`Atomtomate/agents`) could not be filled in this cloud session. For a
documentation-only pull request, one general-purpose reviewer checked accuracy, direction and
consistency in place of the four fleet reviewers.

**Checked and correct:** the line counts; the 4 Go call sites of `Approve(` and the 2 calls of
`ns.pushLog`; the decision titles D1–D8; `pr: 4` (merge commit 370ca31); D7's statement that
it replaces part of D4 (`docs/DECISIONS.md` lines 90–93).

**Findings, and how each was folded:**

1. **D7's `depends_on: [D5, D6]` is not in D7's text.** Fixed: now `[D6]`, marked as inferred,
   and the other edges in the sketch are labelled as inferences.
2. **The Rust app, `companion-app/` and D9 have no source in the repository.** Fixed: they are
   attributed to the brief for this work and to the parallel session, and marked as not on
   `main` yet.
3. **The summary asks about the revisit trigger, but the owner's list did not include it.**
   Fixed: it is now item 5.
4. **"Nothing is uploaded" was too broad: graphify's semantic pass uploads.** Fixed: the
   exception is stated, with "never on the private hub without asking".
5. **`relates_to` mixed strings and a mapping, and `superseded_by_part` was undocumented.**
   Fixed: there is a separate `prs:` key, and every supersession key is listed.
6. **Tags were used inconsistently, and the SavedVariables row did not say what counted.**
   Fixed: the tags are normalised, and the row gives the grep command and says tests are
   included.
7. **The proposal to split decisions is not in `docs/HANDOFF.md` or `docs/IDEAS.md`.** Those
   files are owned by another session right now, so the proposed lines are in the pull
   request body instead.

**Direction:** no `CLAUDE.md` violation. Nothing is installed, no file owned by another session
is edited, and Basic Memory is flagged as needing the owner's yes.
