**Verdict:** Yes. The companion stays inside D7's charter and `docs/CONTRACT.md` §5. Inside the game folder it reads only `WTF/Account/*/SavedVariables/ForeverFriends.lua` and writes only to the two addon folders. It runs nothing it downloaded, never updates itself, never kills a process (it asks a running copy to stop through `stop.request`), adds no dependency, and builds nothing `docs/IDEAS.md` marks later. Five findings, none of them a change of direction. Two are text a member reads that claims more than the code does. Three are places where the record (the contract, the README) no longer says what was built.

Range: `4f41e79...27e1a08`, `docs/reviews/` excluded. Hub side checked against `Atomtomate/wow-forever-hub` branch `lane-m-companion` where it mattered.

### 1. Setup's opening promise is not what setup does

- **What:** Setup opens with "It reads only the addon's own file. Nothing changes until you answer." On every later run nothing is asked before the changes happen. That includes the run the README itself describes ("Running a newer download later updates the addon and the program copy"). A run with a saved game folder and a live link goes straight on: it updates the addon, stops the background copy, replaces and restarts the program copy and registers autostart again. Only then does it ask "Press Enter to close". Also, "reads only the addon's own file" leaves out two reads: every 10 s it reads the path of every running program (that is how it finds the game), and it reads the Blizzard install key in the registry.
- **Where:** `companion/internal/app/setup.go:58-59`, `setup.go:61-91`, `companion/README.md:26-27` (the transcript). Bears on D7 and the group's tone (the hub's D10): this is the trust line a member reads right after a SmartScreen warning.
- **Why it matters:** A member who re-runs a download is told nothing will change, and then everything does. A member who sees the program open every process (in Defender or Process Monitor) after being told it reads one file has a reason to stop trusting the rest of the text.
- **Suggested resolution:** Change the text in the code and in the README transcript. Print "nothing changes until you answer" only on a first run. Say what is read, for example: "Of the game's files it reads only the addon's own; it looks at which programs run only to see whether the game is one of them."

### 2. `remove` can say the opposite of what happened, and does not say what it leaves

- **What:** `remove` always ends with "Removed. The site no longer lists this PC." It prints this even when the unlink failed. In that case the member has just answered "Continue anyway?" to a prompt saying "this PC stays listed on your profile until you remove it there", and with `remove --yes` the failure is never mentioned at all. An unreadable key or state is also treated as "nothing was ever registered" (`remove.go:233`). Separately, nothing tells the member what stays behind:
  - the addon's file in `WTF` (§5 forbids the companion from touching it);
  - the characters already on the hub (a sync never deletes, D8, and `remove` does not either);
  - the program folder, when `remove` runs from the installed copy.

  The prompt at the end of setup promises to "take everything away".
- **Where:** `companion/internal/app/remove.go:194`, `:204`, `:223`, `:233`; `companion/internal/app/setup.go:95`; `companion/README.md:39`, `:132-134`. Bears on D8 ("A sync never deletes") and `docs/CONTRACT.md` §3.1 (unlink).
- **Why it matters:** The PC stays on the member's profile while the program says it is gone. A member who removes the companion to take their data off the site finds their characters still on their profile, with nothing that told them to delete them there.
- **Suggested resolution:** Change the code's text so the last line follows the unlink's outcome. Add one sentence to the confirmation and to Verify item 12: the characters stay on the site (delete them on your profile), and so does the game's own `ForeverFriends.lua` under `WTF`.

### 3. A header the contract does not list

- **What:** Every request carries `User-Agent: ForeverFriendsCompanion/<version>`. D7's "never" list includes "send anything that `docs/CONTRACT.md` does not list", and §3 names only `Content-Type`, the three `X-FF-*` headers and the bodies.
- **Where:** `companion/internal/app/app.go:128`, `companion/internal/hubapi/hubapi.go:113-115`; `docs/DECISIONS.md` D7, `docs/CONTRACT.md` §3 and §5.
- **Why it matters:** It is small and harmless: Go would send a generic agent anyway, and the version is useful to the hub's operator. But D7's rule is absolute and this is the one place the code goes past it. A rule that is quietly broken once is a rule the next change breaks too.
- **Suggested resolution:** Record it with one line in `docs/CONTRACT.md` §3 ("requests carry `User-Agent: ForeverFriendsCompanion/<version>`"), or send Go's default.

### 4. §3.3's failure path is not what the syncer does, in three unrecorded ways

- **What:**
  - **(a)** The contract says `outage` is "sent once, on the first success after at least two consecutive failed requests". As built, every attempt from the third failure on carries it (the author reported this against the plan, but the contract was not changed).
  - **(b)** The contract says "After a failed request it tries again sooner, at most every 30 s". As built, five retries come at 30 s and then retries slow to every 120 s. While the game runs, that is *later* than the 60 s schedule, not sooner.
  - **(c)** The contract says only that failures in the first 60 s after a wake are not counted. As built, a wake also throws away the failures counted before the sleep, so an outage that began before a laptop slept is never reported.
- **Where:** `companion/internal/syncer/syncer.go:21-25`, `:106-110`, `:156-158`, `:214-218`; `docs/CONTRACT.md:263` and `:319-323`.
- **Why it matters:** The contract is the one home of the hub's expectations. The hub stores every outage it receives, without a device id and without dedupe (`src/devices.js:64` on `lane-m-companion`), so (a) means an answer lost after the hub stored it becomes a second, overlapping outage. (b) means a member's "in game" and the end of an outage come back up to two minutes late. That coarsens what D7 item 5 asked for: downtime "as members' PCs saw it".
- **Suggested resolution:** Change the doc. Amend §3.3 to say what is built, or bring the code back to the text. Either way, no format changes.

### 5. Sentences in `companion/README.md` that do not hold

- **What:**
  - Line 15: "never sends what `docs/CONTRACT.md` §5 does not list". §5 is the list of what *never* crosses, so this reads backwards. D7's rule is about what the whole contract lists.
  - Line 22: download "from the site's Game link page". The hub's page is `/companion` ("The companion"), which is what setup's prompt and the root README say. "Game link" is a card on the profile.
  - Line 44: "never replaces a newer addon or program with an older one without asking". The addon is never replaced by an older one and nobody is asked (`install.go:178-180`). Only the program asks. That question depends on reading `version` from the windowed build through a pipe, which no test does and no Verify item names.
  - Line 88: "Only Windows is tested by a person." Nothing has run on a PC yet, and the list below says "none is claimed to work until done".
- **Where:** `companion/README.md:15`, `:22`, `:44`, `:88`.
- **Why it matters:** The owner reads this README. Line 88 claims a test that has not happened, and line 44 promises a question that is never asked.
- **Suggested resolution:** Change the doc. Fix the four sentences, and add a Verify item: "run an older download over a newer install: it asks before replacing the program and leaves the addon alone."

## Decisions

- Go module at the repository root; `embed.go` carries only `*.toc` and `*.lua` (no README, no `spec/`): recorded (root `README.md`, `embed.go`).
- The addon and the companion share one version, the `.toc`'s `## Version`, and the release tag must match it: recorded in both READMEs and `release.yml`, not in `docs/DECISIONS.md`.
- A pushed `v*` tag publishes at once (not as a draft): Windows amd64, Linux amd64, `SHA256SUMS`, generated notes. `workflow_dispatch` builds without publishing: recorded (both READMEs).
- `permissions: contents: write` for the whole release workflow: silent. It is the minimum that publishing needs, so this is no finding.
- The Windows build uses the windowed subsystem, interactive commands attach a console, and `version` prints nothing in a terminal: recorded (`companion/README.md`).
- Program copy in `%LOCALAPPDATA%\Programs\ForeverFriends`, config in `%LOCALAPPDATA%\ForeverFriends` (local, not roaming), autostart under HKCU `Run`: recorded (README "On disk", §5).
- A running copy is replaced by asking it to stop (`stop.request`, wait 45 s), never by killing it: recorded (README).
- Setup runs the installed copy (`version`) as a child process to compare versions, and asks before a downgrade: the question is recorded, the child process is silent and harmless. The addon is never downgraded: recorded wrongly (finding 5).
- A junction is left alone on install and remove; a folder that is not ours is replaced only on a "yes" that `--yes` never gives: recorded (README).
- "Game running" means any process image inside the game folder, with no executable name: recorded (§3.3 field, Verify item 4).
- The game folder is guessed from the registry and Program Files, with no guess off Windows: recorded (Verify item 3, "Other systems").
- A tick every 10 s: recorded (§3.3, "at least every 15 s").
- Retries slow to 120 s after five: silent (finding 4).
- The outage rides on every attempt after two failures: silent in the contract (finding 4).
- A wake discards earlier failures: silent (finding 4).
- `User-Agent` carries the version: silent (finding 3).
- `remove` unlinks first and can go on without the hub; it deletes both addon folders, the config files, the Run value and the program copy: recorded (README item 12, prompt). What stays: silent (finding 2).
- The log is 2 x 256 KiB, holding start, stop and failures, with no keys, codes or names: recorded (README, §5's "small log").
- Off Windows there is no program copy and no background start; setup prints how to run it: recorded (D7, "Other systems").
- Metered connections get no special case: the docs are silent, and the volume (one small request every 5 min while idle) needs none. No finding.
- Lua CI now uses Ubuntu packages instead of the `leafo` actions; Go is checked on Linux and Windows with the race detector and cross-builds; the pre-commit hook runs the Go checks: recorded (workflow comment, root `README.md`, `CLAUDE.md`).

## Follow-ups

- `companion/internal/savedvars/read.go:133` with `companion/internal/wire/wire.go:39`: an entry with no `skills` is uploaded as `"skills": []`. §4 says "The companion leaves `skills` out when the addon's entry has none". The hub (`src/companion.js:190` on `lane-m-companion`) then clears that character's professions, for example ones typed on the site for a character whose first skill read was empty. This is a defect, `pr-tech-review`'s.
- `companion/internal/app/setup.go:153-154`: answering "n" to "The earlier link expired. Start a new one?" returns `ExitOK`, so setup goes on to install, copy, start and register autostart for an unlinked PC, and says "It starts with Windows and is running now" (`run` then exits as not linked). §3.1 step 4 says autostart only once linked. This is a defect, `pr-tech-review`'s.
- `docs/HANDOFF.md`: it is dated 2026-11-10. `CLAUDE.md` wants what no test proves listed there, so the companion's "Verify on the PC" list belongs in it at the close.

## What carried it

`docs/CONTRACT.md` §3.3 "Timing" and §4's last sentence, read line by line against the constants in `syncer.go` and the JSON tags in `wire.go`. That turned up finding 4 and the `skills` follow-up, neither of which is visible from the README.
