# Friction notes

One file per time the *process* failed an agent of the fleet (`.claude/agents/`): the
definition it ran under was wrong or had to repeat itself, a tool or command fought back, a
document had to be worked around, or a "what not to report" rule made it drop something it
believes is real. The agent writes the note itself at the end of its task. Why the channel
exists is the fleet manual's to say (`.claude/agents/CLAUDE.md`, "Three channels beyond the
findings"). The session that called the agent commits the note with the pull request it is
working on; the owner or a `retro` reads it later.

## What a note contains

File name `YYYY-MM-DD-short-title.md`. First line `# <title>`, then three lines:
`**Agent:**` (who filed it), `**Summary:**` (what it cost, one sentence) and `**Retro:**`
(starts as `Unprocessed`). Then, with evidence and without sentiment:

- **What you were doing**: the definition, the task or branch, the step.
- **What happened**: the instruction that was wrong, quoted; the command that failed, with its
  output; the rule that stopped you and, in one line, what it stopped.
- **What it cost**: turns, a wrong guess, a finding dropped. No cost, no note.
- **What would remove it**: the edit and the file, or that you do not know.

Name files in backticks, not as links.

## What is not a friction note

- Anything about this project's code or docs: that is a finding, or a follow-up in the report.
- A cost the next run of the same definition would not pay (the network was down once).
- Something `CLAUDE.md` or the overlay told you where to find.
- The task being hard.
