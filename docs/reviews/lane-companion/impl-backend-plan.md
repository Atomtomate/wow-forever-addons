# The companion, version one: plan (impl-backend, 2026-11-13)

No code written. Built against `docs/CONTRACT.md` (4b746f7 plus the folded corrections) and the director's plan sections 3, 4 C, 5, 6. Module `github.com/Atomtomate/wow-forever-addons`, `go 1.24`, standard library only.

## 1. Packages (all under `companion/internal/`; `companion/main.go` is `os.Exit(app.Main(...))`)
| Package | Owns | Takes in (so no PC is needed) |
|---|---|---|
| `wire` | the JSON types of §3 (`SyncRequest`, `SyncResponse`, `Character`, `Skill`, `Roster`), the numeric limits (50 characters, 40 skills, 64 bytes, 128 KiB), and `ValidateRoster` (bounds and types of what the hub sent) | nothing, pure |
| `savedvars` | Lua-subset parser (§1), `ReadAccounts(fsys, gameDir)`, merge by key (newer `seen` wins, ties by account name), `Upload()` (50 newest, 40 skills, names dropped, over-64-byte name/realm left out), file fingerprints | `fs.FS` rooted at the game folder |
| `dataaddon` | `Render(roster, written)`, the §2 escaper, `Write(dir, roster, now)` atomic and only when content differs from what is there ignoring `written`, the `.toc` | directory path, clock |
| `hubapi` | the only package that sends: `Sign`, headers, `SeqStore`, the 409 rule, `Link`, `LinkStatus`, `Unlink`, `Sync`; maps each answer to one `Outcome` (OK, Rejected for 400/413, Unlinked for 401, Throttled for 429, Unreachable) | origin, device, `ed25519.PrivateKey`, `SeqStore`, `*http.Client` |
| `linkflow` | §3.1 as a function: start or resume, poll every `interval`, expiry, returns the member | `hubapi.Client`, `config.Store`, clock, `sleep func` |
| `config` | folder layout, `config.json`, `state.json`, `device.key` (PKCS8 PEM), atomic write, `SeqStore` implementation | directory path |
| `install` | `.toc` reader, version compare, install and remove of `ForeverFriends`, the empty `ForeverFriendsData`, junction rule | embedded `fs.FS`, AddOns path, `Prompt` func |
| `gamefind` | validate or normalise a typed folder, rank candidate folders, `Running(images, gameDir)` | `exists func`, candidate list, image list |
| `syncer` | the background loop and its state machine (§6) | `Clock`, `Hub` interface (hubapi's `Sync`), `Procs` interface, `savedvars` and `dataaddon` through small interfaces, `config.State`, `applog` |
| `applog` | capped log (256 KiB, one `.1` rotation); never logs keys, signatures, codes after use, character names | `io.Writer` |
| `instance` | one-instance lock and stop request (§6) | lock file path |
| `platform` | `_windows.go` / `_other.go`: `ConfigDir`, `ProgramDir`, `GameCandidates`, `ListImages`, `OpenURL`, `Autostart{Enable,Disable,Enabled}`, `SpawnDetached`, `AttachConsole` | nothing; `app` wraps them in an `Env` struct of func fields |
| `app` | commands `setup` (default), `run`, `status`, `remove`, `version`; prompts over `io.Reader/Writer`; wires everything | `Env` (stdio, clock, HTTP client, platform funcs, embedded FS, args) |
| `hubfake` (test support) | an `httptest` hub that verifies signatures and sequence per §3.2 and serves the §3 answers | used by `hubapi`, `linkflow`, `syncer`, `app` tests |

Imports: `wire`, `savedvars`, `dataaddon`, `hubapi`, `config`, `install`, `gamefind`, `applog`, `instance`, `platform` import only the standard library (and `wire`); `linkflow` and `syncer`
import those leaves; `app` imports all; nothing imports `app`, so no cycle. Packages touching disk (`config`, `install`, `dataaddon`, `applog`, `instance`) take a root path and run in
`t.TempDir()`; none reaches for the network, clock, process list or registry. Root `embed.go` (package `foreverfriends`, added last): `//go:embed addons/ForeverFriends/*.toc
addons/ForeverFriends/*.lua`, exported `Addon embed.FS`, globs because `addons/*/spec/` sits inside an addon folder; its test asserts every file the `.toc` lists is embedded.

## 2. Build order (each step green on `go vet ./... && go test ./...` before the next; commits in this order)
1. `go.mod`, `wire`, CI Go job and Makefile targets (every later commit is checked); 2. `hubapi` (vector first, then requests and status table), `config`, `hubfake`;
3. `savedvars` (sample and golden from `tools/parse_sv.py`), `dataaddon` (`lua5.1` test); 4. `install`, `gamefind`, `instance`, `applog`; 5. `linkflow`, `syncer`;
6. `platform` (cross-build and `GOOS=windows go vet`), `app`; 7. README with "Verify on the PC", `release.yml`, input for `docs/HANDOFF.md`; then merge `lane/foreverfriends`, root `embed.go`, final cross-build.

## 3. Contract section to package
| Section | Implemented in | Section | Implemented in |
|---|---|---|---|
| §1 file | `savedvars` | §3.2 headers, signature, seq, 409 | `hubapi`, `config` (seq store) |
| §2 toc, Data.lua, escaping, atomic | `dataaddon` (+ `install` for the empty first file) | §3.3 sync body, outage, etag | `syncer` builds, `hubapi` sends, `wire` types |
| §3 intro: origin, no redirect, base64 | `hubapi` (`ParseOrigin`, client) | §3.3 status table | `hubapi` (Outcome) + `syncer` (reaction) |
| §3.1 link, resume, unlink | `linkflow`, `app` (`remove`) | §3.3 timing | `syncer` |
| §5 what never crosses | `savedvars.Upload` alone makes characters; `install`/`dataaddon` alone write, refusing paths outside their folder | §4 | hub's |

## 4. On disk
Windows: config `%LOCALAPPDATA%\ForeverFriends\` holding `config.json` (hub, device, device name, member, game folder, pending link: code, absolute expiry), `state.json` (seq,
uploaded fingerprints, roster etag, revoked flag), `device.key`, `companion.log`, `companion.log.1`, `run.lock`, `stop.request`. Program copy
`%LOCALAPPDATA%\Programs\ForeverFriends\ForeverFriendsCompanion.exe`. Autostart: `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, value `ForeverFriendsCompanion` =
`"<program copy>" run`. Local, not roaming: `%APPDATA%` would copy the private key to every PC of a domain profile and give two devices one key. The user-only guarantee on
Windows is that folder's inherited ACL (`0600` means nothing there); not shelled out to `icacls`. macOS/Linux: `os.UserConfigDir()/forever-friends/` with the same files,
`0700` folder, `0600` key; no program copy (it runs where the member put it), no autostart. The game folder is the flavour folder (`...\_forever_`), the `<game>` of the
contract; written only: `Interface\AddOns\ForeverFriends`, `...\ForeverFriendsData`. `state.json` is rewritten atomically (temp, fsync, rename) before every signed request;
`config.json` only on change.

## 5. Interactive setup, as the member sees it (double-click, or `ForeverFriendsCompanion.exe setup`)
Windows build is `-H=windowsgui` (no console flash from the autostart entry); setup/status/remove call `AttachConsole(parent)` or else `AllocConsole`, reopen
`CONIN$`/`CONOUT$`, set UTF-8 output. `run` never has one.

```
Forever Friends companion 0.1.0. Links this PC to the group's site; installs the addon; tells the site when the game runs.
Site address (from the site's "Game link" page) [https://hub.example]:   <type or Enter>
Game folder: found C:\Program Files (x86)\World of Warcraft\_forever_ . Use it? [Y/n]   (none found: "Drag the _forever_ folder here:")
Name of this PC on the site [JULIAN-PC]:                                  <Enter or type>
Your code:  WXYZ-BCDF      Opening your browser. If nothing opens, go to https://hub.example/companion/link?code=WXYZ-BCDF
Waiting for you to press Link on the site (10 minutes, Ctrl+C to stop; run again to continue)... Linked as Anna.
Addon installed: ForeverFriends 0.1.0, ForeverFriendsData. Companion copied to %LOCALAPPDATA%\Programs\ForeverFriends,
starts with Windows, running now. Restart the game once so it sees the new addon. Press Enter to close.
```
Steps and order: (1) refuse a second console copy if `run.lock` is held by a setup. (2) hub address: `ParseOrigin` (https, or http on 127.0.0.1/::1/localhost; no userinfo,
path, query, fragment; trailing `/` stripped). (3) game folder: candidates from `platform.GameCandidates` (default Program Files paths, plus the install path from `reg query
HKLM\SOFTWARE\WOW6432Node\Blizzard Entertainment\World of Warcraft` via a call below; both unverified), accepted only with `Interface` or `WTF` inside; a typed `World of
Warcraft` folder gets `\_forever_` appended when that exists; quotes stripped. (4) if `config.json` has an unexpired pending link for this hub: show the same code and resume
polling, no new start (the hub allows 5 starts per 15 min); expired or 401 on poll: ask "Start a new link? [Y/n]". Otherwise new key pair, `Link`, write config + key, show
code, `OpenURL` (the address is built here, never taken from the answer). 429: "The site is busy; try again in a few minutes". (5) on `linked`: install addons (below), copy
self, stop old instance, spawn `run` hidden, register autostart, last. Autostart only once linked is true. (6) Unlinked 401 on a later `setup`: says so and starts over. Re-
running `setup` when linked prints status (member, folder, running?, last problem) and offers "type remove, or Enter". `status` is that print; `remove`: confirm, request stop,
`Unlink` (401 counts as done; unreachable: "this PC stays listed on your profile until you remove it there; continue? [y/N]"), delete autostart value, key, config, state, log,
then the two addon folders, each only if it is not a link, and the program copy unless it is the running exe (then names the folder to delete). Non-interactive mode (for the
end-to-end run and scripts): `setup --yes --hub URL --game DIR --config-dir DIR --name N [--no-browser] [--no-autostart] [--no-self-copy]` prints `CODE: WXYZ-BCDF` on its own
line, flushes, polls until linked or expired, exit 0 only when linked (exit 2 expired/declined, 1 other). `run --once` does exactly one tick and one sync, then exits: 0 sync
OK, 3 unlinked (401), 4 unreachable. Both flags exist for §6 of the director's plan, not as options for members.

## 6. Background loop, state machine, one instance
`run`: acquire lock or exit 0 silently. Tick every 10 s (process look at least every 15 s): stat the SavedVariables files, list process images, check `stop.request`. A sync is
due when: schedule time passed (60 s game running, 300 s not), or the game state changed (immediately), or a fingerprint differs from `state.uploaded` and was identical on two
ticks (the client's write has finished; an unparsable file is simply read again next tick and never half-uploaded). One request at a time, blocking, 20 s timeout. Upload
carries characters when any fingerprint differs from `state.uploaded` (all files empty-state on first sync after link), merged across accounts; fingerprints are stored only on
200 or 400/413 ("marks as uploaded"). States: `Healthy` (last request answered ok), `Failing{n, first}` (n consecutive Unreachable; `n>=2` is "outage"), `Revoked`.
- Unreachable: n++ (first failure time kept); retry in 30 s for n<=5, then 120 s; characters and etag stay pending.
- OK after n>=2: body carries `outage{from:first, to:now}` once, UTC `Z`; if that request fails, the same `from` is kept for the next try; cleared on OK or on 400/413 (hub
  answered). OK after n<2: n=0, nothing sent.
- 400/413: log the one-sentence error, mark uploaded, keep schedule. 429: wait 300 s, neither a failure nor a success. 409 and the one resend are inside `hubapi`; a second 409
  is Unreachable.
- 401 on `sync`: `Revoked`, written to `state.json`, logged once, the loop exits; the autostart run exits at once while the flag is set; `status`/`setup` say "this PC is no
  longer linked" and linking again clears it. (Silent otherwise: no toast without a dependency.)
- Wake from sleep (tick gap over 60 s): `n` reset and failures in the next 60 s are not counted, so a Wi-Fi that is not back yet does not become a hub outage in the owner's
  list.
- Roster: send `roster_etag` from state only if `Data.lua` exists; on a `roster`, `dataaddon.Write` first, store the etag after it succeeded (a failed write, e.g. file in use,
  is logged and the hub just resends next sync). A response with `ok:true` but a bad roster or `member`/`linked` shape counts as Unreachable.
- Instance: `instance.Acquire` holds `run.lock` for the process life: Windows `syscall.CreateFile` with share mode 0 (a second open gets `ERROR_SHARING_VIOLATION`; the OS
  releases it on death, so no stale lock), Unix `syscall.Flock(LOCK_EX|LOCK_NB)`. The holder writes its pid inside for messages. A setup run creates `stop.request`, polls the
  lock for up to 45 s; the loop sees the file within one 10 s tick, releases the lock and exits; setup deletes the file after taking the lock. Not released: setup says "could
  not stop the running companion; end it in Task Manager" and replaces nothing. No `taskkill`, no pids killed. Only a member-started setup or remove stops it.

## 7. Install rules (`install.Install(embedded, addonsDir)`)
Version = `## Version` of the embedded `ForeverFriends.toc`; the companion's reported `companion` string is the same value (release workflow refuses a tag that is not `v` plus
it). Files installed = exactly the `.toc` and the files it lists (names with `..`, drive letters, absolute paths or empty rejected; `\` read as `/`), each written beside and
renamed; the `.toc` last, so a half install keeps the old version and is repaired next run. Existing folder:
- is a link (`Lstat` mode has `ModeSymlink` or, since Go 1.23, `ModeIrregular` for a junction, or `EvalSymlinks` differs from the path ignoring case): touch nothing, say "left
  alone: linked to your checkout". Applies to `ForeverFriends` only in practice; `Data` too.
- absent or empty: install. Contains a readable `ForeverFriends.toc`: installed version older: replace; same: rewrite files whose bytes differ (repair), nothing if equal;
  newer: leave alone, say so (never downgrade). Non-empty with no such `.toc`: refuse in non-interactive mode, ask "this is not ours, replace? [y/N]" otherwise. Replacing
  removes only `.lua`/`.toc`/`.xml` files not in the new list.
- game running: proceed anyway (the client has read the folder; it only needs a restart), and say so. A rename that fails on a locked file: leave the old version, report, try
  next setup. `ForeverFriendsData` is created at install with `.toc` (Interface copied from the embedded toc, validated as digits and commas) and a `Data.lua` of `members =
  {}`, `written = 0`, so the game already knows the folder at the next start (contract proposal below). Companion program copy: skip when `os.Executable()` is already the
  installed path; otherwise stop the old instance, write `.exe.new`, rename over; if the installed copy answers `version` with a higher number, ask before downgrading.

## 8. Windows calls, with no dependency
| Need | Call | Reached by |
|---|---|---|
| list processes | `CreateToolhelp32Snapshot`, `Process32First/Next`, `OpenProcess(0x1000)`, `CloseHandle` | `syscall` wrappers; `QueryFullProcessImageNameW` through `kernel32` `NewLazyDLL` |
| Run key | `RegCreateKeyExW`, `RegSetValueExW`, `RegDeleteValueW`, `RegQueryValueExW`, `RegCloseKey` | `advapi32` `NewLazyDLL` (no `reg.exe`: no child process, no quoting of the value) |
| open browser | `ShellExecuteW(0,"open",url,...,SW_SHOWNORMAL)` | `shell32` `NewLazyDLL`; the URL is http(s) built by the program |
| hidden background | `exec.Cmd` with `SysProcAttr{HideWindow:true, CreationFlags: CREATE_NO_WINDOW|DETACHED_PROCESS|CREATE_NEW_PROCESS_GROUP}` | `os/exec`, `syscall` |
| console for setup | `AttachConsole(-1)`/`AllocConsole`, `SetConsoleOutputCP(65001)` | `kernel32` `NewLazyDLL` |
| game folder hint | `RegOpenKeyExW`/`RegQueryValueExW` on `HKLM\SOFTWARE\WOW6432Node\Blizzard Entertainment\World of Warcraft` | `advapi32`; failure just means no hint |
Unix: `xdg-open` / `open` for the browser; `Flock`; process list from `/proc/*/exe` (Linux) and `ps -axo comm=` is **not** used on macOS (returns names, not paths): macOS
lists nothing and reports `game_running:false`, which is stated in the README (by-hand platform).

## 9. Tests, named
- `wire`: `TestLimitsWorstCaseUnder128KiB`, `TestValidateRosterBounds`, `TestRosterLevelNotInteger`.
- `hubapi`: `TestSignVector` (§3.2 key and signature), `TestSignatureOverExactBodyBytes`, `TestStatusTable` (one httptest case per §3.3 row plus link 400/413/429),
  `TestNoRedirectFollowed` (302/307 Unreachable, target never hit), `TestOkTrueNonJSONIsUnreachable`, `TestConflictResendsOnceThenFails`, `TestSeqStoredBeforeSend`,
  `TestSeqHeaderShape`, `TestOriginRules`, `TestBase64Padded`, `TestResponseBodyCapped`, `TestNewKeyPerLink`.
- `linkflow`: `TestLinkShowsCodeAndPolls`, `TestResumeBeforeExpiryShowsSameCode`, `TestResumeAfterExpiryStartsOver`, `TestPollUnauthorizedIsExpired`, `TestDeclinedStops`, `TestRateLimitMessage`.
- `savedvars`: `TestParseAgreesWithPythonGolden` (committed sample plus `golden.json` made once by `tools/parse_sv.py`), `TestParseAgreesWithPythonLive` (skips without `python3`),
  `TestEscapesAndLongStrings`, `TestGarbageNeverPanics` + `FuzzParse` seeds, `TestDepthAndSizeCaps`, `TestPartialFileIsAnError`, `TestMergeNewerSeenWins`, `TestUploadCaps` (51
  characters, 41 skills, 65-byte realm), `TestUploadDropsSkillNames`, `TestUnknownVersionRefused`, `TestMissingFieldsSkipEntry`.
- `dataaddon`: `TestEscapeEveryByte0to255` (through `lua5.1` when on PATH, else a Go reference unescaper), `TestHostileStringsStayLiterals` (`"]]..`, `\`, `\1` then a digit, newline,
  NUL, 127; loaded under an empty environment, no global written), `TestIntegersOnlyFromFormatter`, `TestWriteAtomicAndOnlyWhenDifferent` (`written` ignored), `TestTocInterfaceCopied`.
- `install`: `TestTocParse`, `TestVersionCompare`, `TestFreshInstall`, `TestOlderReplaced`, `TestSameRepaired`, `TestNewerLeftAlone`, `TestJunctionLeftAlone` (symlink here; junction is a PC
  item), `TestForeignFolderRefused`, `TestStaleFilesRemoved`, `TestTraversalNamesRejected`, `TestTocWrittenLast`, `TestDataFolderCreatedEmpty`, `TestRemoveSkipsLinks`, `TestRunningGameDoesNotBlock`.
- `gamefind`: `TestRunningMatchesPathPrefix` (separator-bounded; `_forever_x` is not `_forever_`), `TestNormaliseTypedFolder`. `config`/`instance`/`applog`: `TestAtomicWrite`, `TestKeyFileMode0600`,
  `TestKeyRoundTrip`, `TestSeqMonotonic`, `TestSecondAcquireFails`, `TestStopRequestReleases`, `TestLogRotates`, `TestLogNeverHoldsKey`.
- `syncer` (fake clock, `hubfake`): `TestSchedule60And300`, `TestGameStartAndExitSyncAtOnce`, `TestFileChangeUploadsAfterTwoStableTicks`, `TestOutageSentOnceAfterTwoFailures`,
  `TestOutageFromKeptWhileFailing`, `TestOneFailureNoOutage`, `TestRetry30Then120`, `Test401Revokes`, `Test400MarksUploaded`, `Test429Waits300NoFailure`, `TestEtagOnlyWhileDataLuaExists`,
  `TestFailedDataWriteKeepsOldEtag`, `TestWakeForgivesFailures`, `TestFirstSyncAlwaysCharacters`, `TestCharactersNotResentWhenUnchanged`.
- `app`: `TestSetupNonInteractiveLinksInstalls`, `TestSetupResumesPending`, `TestRunOnceExitCodes`, `TestRemoveCallsUnlinkThenDeletes`, `TestRemoveUnreachableAsks`,
  `TestStatusAfterRevoked`, `TestNothingWrittenOutsideTwoFolders`. Gates: `gofmt -l` empty, `go vet`, `GOOS=windows go vet ./...`, Windows and darwin cross-builds.
## 10. CI and release
`ci.yml` gets job `go`: matrix `ubuntu-latest`, `windows-latest`; `actions/setup-go@v5` with `go-version-file: go.mod`; `gofmt -l .` must be empty; `go vet ./...`; `go test
./...` (ubuntu adds `-race` and `apt-get install -y lua5.1` for the Lua tests; the Windows run covers the lock, console-free paths and `CreateFile`); ubuntu also runs
`GOOS=windows GOARCH=amd64 go build -o /dev/null ./companion` and the darwin one. Makefile: `golint`, `gotest` targets, `check` runs all four; `.githooks/pre-commit` runs them
when `.go` or `go.mod` is staged, skipping with a note if `go` is missing. `release.yml` on tag `v*` (and `workflow_dispatch` as a dry run that only uploads artifacts):
checkout, setup-go, check `${GITHUB_REF_NAME} == v + <## Version of ForeverFriends.toc>` else fail, `go test ./...`, build `windows/amd64` with `-trimpath -ldflags
"-H=windowsgui -X .../app.version=$V"` plus linux/amd64 and darwin/arm64, `sha256sum` into `SHA256SUMS`, then `gh release create "$TAG" --title "$TAG" --generate-notes
files...` with `permissions: contents: write`. No UPX, no packing, no obfuscation.
## 11. Verify on the PC (README of the companion, same list in `docs/HANDOFF.md`)
The director's list stands; additions: double-click prints in a console and `run` shows none (Task Manager, background process, no window at logon); `reg query
HKCU\Software\Microsoft\Windows\CurrentVersion\Run /v ForeverFriendsCompanion`; process listing finds the game (`Get-Process | ? { $_.Path -like "*\_forever_\*" } | select
Path`, then the companion's `status` says "game running" within 15 s and "not running" after exit, also with the game started as admin); the game folder hint key exists or not
(`reg query "HKLM\SOFTWARE\WOW6432Node\Blizzard Entertainment\World of Warcraft"`); a junction from `tools\link.ps1` over `ForeverFriends` is left alone (`Get-Item
<AddOns>\ForeverFriends | select LinkType`, then run setup and check the checkout is untouched); `Data.lua` replaced while the game runs and `/reload` shows the list; setup
while the old instance runs (the new exe replaces it, `Get-Process ForeverFriendsCompanion` shows one); remove from the installed copy; `icacls
"%LOCALAPPDATA%\ForeverFriends\device.key"` lists only the user, SYSTEM and Administrators; SmartScreen and Defender on the first download and first run; resume from sleep
does not create an outage.
## 12. Not doing, and risks
Not doing: self-update, tray, non-Windows autostart, two members per PC, MarketScan, the addon list, any GUI toolkit or module, any `taskkill`/PowerShell/`cmd` child, any request
outside the four routes, any path outside the two addon folders and the config/program folders; no edits to `addons/`, `spec/`, `docs/CONTRACT.md`, `docs/DECISIONS.md`.

- **Antivirus heuristics** (unsigned, copies itself, Run key, background network): lower by doing less that looks like malware (no `reg.exe`/`cmd`/PowerShell children, no UPX
  or packing, no self-update, no downloads, no process injection; Run key only after the member pressed Link), a reproducible `-trimpath` build in CI from a tag, `SHA256SUMS`
  beside the file, plain console text saying what it does before it does it, a first-run description in the hub's `/companion` page, a false-positive submission to Microsoft
  after the first release, and a Windows version-info resource if the owner accepts one more build step (a `.syso` needs `rsrc`, a tool outside the module; not planned). A
  code-signing certificate is the real fix and the owner's call.
- Process detection by image path cannot see a launcher or crash handler in the game folder (name of the real exe unknown); it reports "running" while the helper lives.
- Outage times use the PC's clock (the contract does not trust it). Junctions reporting `ModeIrregular` on Windows (Go 1.23 notes) is untested here.
- `AttachConsole`/`AllocConsole` with `-H=windowsgui` is the fiddliest Windows piece; fallback is a console build and accepting a flash at logon.
## 13. Where the director's sketch or the contract did not survive
Contract (proposals, not edits):
- §2: add "At install the companion writes `ForeverFriendsData` with a `Data.lua` of `members = {}` and `written = 0`", because the folder is otherwise created at the first
  sync and the game then needs a second restart before the list can ever load.
- §3.1 step 4: add "any status `link/status` is not named for (including 429 and 5xx) is retried at the same interval until the expiry", because the status table of §3.3 does
  not cover it and the link would otherwise fail on a blip.
- §3.3 429 row: add "is neither a failed request nor a success: it does not start or end an outage", because "the hub is not reachable" does not describe a proxy limit.
- §3.3 outage: add "after a wake from sleep, failures in the first minute are not counted", because a laptop's own network would otherwise show as hub downtime. Director's
Director's sketch:
- Addon install moves after `linked`, not before; a declined link then leaves nothing in the game folder (autostart already waited for it).
- Release builds are `windowsgui` with an attached or allocated console for interactive commands; a console build would flash at every logon.
- Config under `%LOCALAPPDATA%`, never `%APPDATA%` (roaming copies the key). Program copy in `%LOCALAPPDATA%\Programs\ForeverFriends`.
- Registry through `advapi32` and the browser through `ShellExecuteW`, not `reg.exe` and `rundll32`: no child processes, no quoting traps, one fewer AV signal.
- One instance by an exclusive-share file handle and a `stop.request` file instead of a named mutex and `taskkill`; nothing can kill the wrong process.
- `tools/parse_sv.py` has no test sample in the repository; the sample and a golden JSON are made here from the Python tool, and the Go parser also gets real Lua escape
  semantics (`\a \b \f \v`), where Python returns the letter; the sample avoids those so both agree.
- A shared `wire` package holds the types and limits; the version is one number, the embedded `## Version`, for addon and companion alike.

## 14. Open questions for the owner
1. Embed a default hub address into the release build (GitHub variable `HUB_URL`, offered as the Enter default), or always type it? I lean: embed, optional.
2. Release assets for macOS/Linux are "built, started by hand, untested": publish them, or Windows only? I lean: Windows plus Linux, no macOS.


