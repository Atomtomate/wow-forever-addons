**Verdict:** a little heavier than the work needs, about 300 of 3,300 production lines (the 230-line `hubfake` and the tests aside), and none of it structural: the weight is a few extras beyond the contract (a program-downgrade guard, a parser dialect the client does not write, a registry hint) and fields nothing reads.
Package grain, as asked: `wire`, `hubapi`, `config`, `savedvars`, `dataaddon`, `install`, `syncer`, `linkflow`, `platform` and `app` each hold 130 to 800 lines with their own reason to change and their own tests; keep them. `applog` (40 lines, one construction site), `instance` (a lock plus three one-line wrappers over `os.WriteFile`, `os.Stat`, `os.Remove`) and `gamefind` (four functions, `First` has one caller) are thin, but folding them saves a package clause and a doc comment each, so it is not worth a change. The `Env` and `Platform` structs, the `_windows.go` / `_other.go` pairs and the non-interactive flags are on the "required" list and not counted.

Reviewed: `git diff c890b92...6685ddb` at `6685ddb`. Read every non-test file under `companion/`, the tests of `savedvars`, `syncer`, `app`, `install`, `dataaddon`, `hubapi`, `linkflow`, `config`, `wire`, `gamefind`, the CI and release files and the README. Ran the built binary once (`--version`, below).

### 1. Fields, returns and guards that nothing reads
- **What** — Written and never read in production: `LinkResponse.OK`, `StatusResponse.OK`, `SyncResponse.OK` and `.Member` (`companion/internal/wire/wire.go:73,82,108`; `hubapi.reduce` already decides OK); `savedvars.File.Problem` (`savedvars/read.go:23,45`); the bool that `Syncer.Tick` returns (`syncer/syncer.go:101`, no caller reads it); the "changed" bool of `dataaddon.Write` (`dataaddon/dataaddon.go:97`, `app/run.go` discards it, one test reads it and the same test already reads the file); `syncer.Status`'s `Idle`, `Healthy`, `Failing` (`syncer.go:39-41`; production only compares with `Revoked`, three tests read `Healthy`); `hubfake.Hub.Interval` (`hubfake/hubfake.go:41,179`, no test sets it). Guards that cannot fire: `"--version"` in `app.Main` (`app/app.go:83,113`: an argument starting with `-` is never taken as the command, and I ran the build: `--version` is refused by setup's flag parser, `version` prints `0.1.0`); `d.Store != nil` (`syncer.go:80`, `Tick` dereferences `Store` unconditionally); `c.env.Hostname != nil` (`app/setup.go:159`, `DefaultEnv` and every test set it).
- **Why it recurs** — yes: each new response type copies `OK bool` (three already), each new loop method copies "return whether it did something".
- **The smaller version** — drop the fields; `Tick` returns nothing; `Write` returns `error`; `Status` becomes `Revoked() bool`, and the tests that assert `Healthy` assert `LastOutcome() == hubapi.OK`.
- **Net effect** — about -33 lines, one enum gone, nothing left that a reader must work out is unused.

### 2. This PC's signed client, and its pending link, are each built more than once
- **What** — `hubapi.Client{Origin, Device, Key, Seq, HTTP, UserAgent}` is assembled in `app/run.go:61-67`, `app/remove.go:108-114` and `linkflow/linkflow.go:107-111`, each after its own `LoadKey`; two of the three also run `ParseOrigin(cfg.Hub)` and `linkflow` does not, so the rule is already inconsistent. Separately `linkflow.Pending` (`linkflow.go:34`) is a second type for `config.Pending` (`config/config.go:33`), converted at both ends (`linkflow.go:45-52`, `77-88`) with the 5 s default applied twice (`:51`, `:79`).
- **Why it recurs** — yes: a fourth command that talks to the hub is a fourth copy, and a new `Client` field is three edits.
- **The smaller version** — one `func (c *cmdCtx) client(dir, cfg, store) (*hubapi.Client, error)` in `app` that loads the key and validates the origin once; `linkflow.Wait` takes the client; `config.Pending` gets `Expires()` and `Interval()` methods that hold the default, and `linkflow.Pending` goes.
- **Net effect** — about -20 lines; one place decides whether `cfg.Hub` is re-validated.

### 3. The downgrade guard on the program copy, and what it forced on `version`
- **What** — `setup` runs the installed copy with `version` to compare it with itself and asks before replacing a newer one: `platform.QueryVersion` (`platform/platform.go:17-28`, the only non-OS code there besides `Executable`), the `Platform.QueryVersion` field and its `DefaultEnv` line, `copyIt` in `app/setup.go:279-282`, `TestSetupNewerInstalledCopyIsNotDowngradedWithYes` (`app/app_test.go:314-324`). To make the pipe read work, `Main` keeps `version` out of `PrepareConsole` (`app.go:80-83`), so on Windows `version` prints nothing in a terminal, which the README states as "by design" (`companion/README.md:106`) and lists as a Verify item. D7 asks only that a newer companion replaces the addon; the addon's own never-downgrade rule (`install.Install`) stays and already covers the file the game loads. An old exe over a newer program copy speaks the same `v: 1` and is cured by the next download.
- **Why it recurs** — no.
- **The smaller version** — setup always copies the running exe over the program copy; `version` attaches a console like `status`.
- **Net effect** — about -45 lines, a subprocess launch removed, and one quirk and one README caveat gone. (Alternative if the guard is wanted: store the version in `config.json` at copy time instead of running the file.)

### 4. A parser dialect the client does not write
- **What** — `savedvars/parse.go` accepts long strings `[[..]]`, long comments `--[[..]]`, hex numbers and single-quoted strings: `longOpen`, `longBody`, `isLongString` (`:150-182`, `:375-381`), their branches in `skip`, `value` and `table` (`:117-124`, `:196-202`, `:336`), and the hex paths in `number` (`:223-228`, `:245-257`), about 70 of 398 lines. The package doc says "only the dialect the client emits"; the client writes double-quoted strings with escapes, decimal numbers and `-- [n]` comments (the sample's own `-- [1]` lines), and nothing in `docs/RESEARCH.md` or the contract says otherwise. The `hex`, `long` and `plain` lines of `testdata/sample.lua` are the author's additions to cover them. The cost of being wrong is cheap: such a file fails closed as `ErrSyntax` and is retried. The source is the Python tool `tools/parse_sv.py`, whose grammar the Go parser and the golden test now track.
- **Why it recurs** — no, but the golden file ties the two parsers' grammar together, so every future change to one means the other.
- **The smaller version** — quoted strings, decimal numbers, `true`/`false`/`nil`, `--` line comments, `;` and `,` separators; the sample and `golden.json` lose three entries and `TestEscapesAndLongStrings` loses its long-string case.
- **Net effect** — about -75 lines of parser and three concepts. I could not check the client's output from this repo; if the owner has seen one of these in a real file, keep that one.

### 5. The README restates D7 and the workflow
- **What** — `companion/README.md:14-17` is `docs/DECISIONS.md` D7's "What it never does" almost word for word (and `docs/CONTRACT.md` §5, `CLAUDE.md` and the overlay hold it too: five copies of one rule); "Other systems" (`README.md:81-90`) is D7's last paragraph with one added fact (`/proc/<pid>/exe`); "Releasing" (`README.md:91-99`) restates the tag rule and the build steps that `.github/workflows/release.yml:2-4` and its steps already say.
- **Why it recurs** — yes: every new document about the companion copies the never-list; the copy nobody updates is where D7 and the manual drift.
- **The smaller version** — the README points to D7 for the never-list and the platform limits and keeps what D7 lacks (the Wine/Proton caveat); "Releasing" keeps the one consequence (a companion-only fix still raises the addon's `## Version`).
- **Net effect** — about -15 lines of prose, and three fewer places to edit when D7 changes. Only the README's copies are in this diff.

### 6. `cleanDeviceName` repeats the hub's rule
- **What** — `app/app.go:163-179` drops control and format characters, cuts to 40 and defaults to `PC`, which is what `docs/CONTRACT.md` §3.1 step 2 says the hub does to `device_name` (the hub also drops line separators, so the copy has already drifted); plus `deviceNameLimit`, the `unicode` import and `TestDeviceNameCleaned` (`app_test.go:373-380`). The hub is the trust edge and cleans anyway; the copy only changes what the prompt shows.
- **Why it recurs** — no.
- **The smaller version** — default to `strings.TrimSpace(hostname)` and send what was typed.
- **Net effect** — about -22 lines, one fewer copy of a hub rule.

### 7. Standard library and an unreachable check
- **What** — `sameFingerprints` (`syncer/syncer.go:139-149`) is `maps.Equal`; the copy loop in `Store.Snapshot` (`config/config.go:160-163`) is `maps.Clone` (`go.mod` says 1.24). `OpenURL` checks for an http(s) prefix in both OS files (`platform_other.go:55-57`, `platform_windows.go:146-148`, the latter hand-rolling `HasPrefix` twice), but its one caller builds `hub + "/companion/link?code=..."` from an address `ParseOrigin` already limited to https or loopback http, so it cannot fire. `errorFileNotFound = syscall.Errno(2)` (`platform_windows.go:40`) is `syscall.ERROR_FILE_NOT_FOUND`.
- **Why it recurs** — the `OpenURL` check is the pair's shape for any next OS function; the rest does not.
- **The smaller version** — as above, two library calls and two deleted checks.
- **Net effect** — about -22 lines.

### 8. The registry hint for the game folder (a judgment call)
- **What** — `regString` and its call in `GameCandidates` (`platform/platform_windows.go:76-78`, `89-104`) plus `hkeyLocalMachine`, `keyQueryValue`, `keyWow6432`, `procRegQueryValueEx`: about 22 lines of `unsafe` calls that nothing here can run, which the README lists as unverified (Verify item 3). A miss costs one drag-and-drop prompt that `resolveGame` offers anyway, and `--game` exists; the two `Program Files` guesses stay.
- **Why it recurs** — no.
- **The smaller version** — keep the environment-variable candidates only.
- **Net effect** — about -22 lines of the least testable code in the PR and one Verify item. Worth keeping only if the owner's installs are often off `Program Files`.

### 9. The embedded `.toc`'s Interface line is validated on every write
- **What** — `dataaddon.interfaceValue` and the `error` return of `Toc` (`dataaddon/dataaddon.go:23,77-83`) are threaded through `Write` and `EnsureEmpty`, and `TestTocInterfaceCopiedAndValidated` tries `"16001\n## Evil: 1"`. The value comes from `install.ParseToc`, which splits at newlines, out of the build's own `ForeverFriends.toc`: a bad one is a build defect, not a run-time state.
- **Why it recurs** — no.
- **The smaller version** — `Toc(iface) []byte`; one assertion in `install/embedded_test.go` that the embedded Interface matches `^\d+(,\d+)*$`, so a bad value fails CI instead of every PC. `ParseToc`'s file-name guard stays: it protects the writes-only-inside-the-folders invariant.
- **Net effect** — about -12 lines and two error paths.

### 10. Dead scaffolding in two tests
- **What** — `TestRetry30ThenSlower` (`syncer/syncer_test.go:203-213`) builds `r`, advances it twice and asserts nothing on it before `r2` does the real work; `TestCRLFReadsLikeLF` (`savedvars/savedvars_test.go:243-275`) builds `tableOf`, `pythonShape` and `normalise` wrappers to compare two parse results where `reflect.DeepEqual(a, b)` already compares maps of values and `*Table` deeply; `len(b) == 0` at `syncer_test.go:176` cannot be true after `json.Marshal` of a struct.
- **Why it recurs** — no.
- **The smaller version** — delete the first half of the retry test; compare with `reflect.DeepEqual`.
- **Net effect** — about -25 lines; each test says what it checks.

**What carried it:** the brief's "Required by written rules, so not yours to cut" paragraph, at the moment I first read the 12-field `Platform` struct and `DefaultEnv` and would otherwise have raised them.
