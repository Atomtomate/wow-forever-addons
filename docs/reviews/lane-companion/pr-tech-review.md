**Verdict:** Not ready. Fix finding 1 before anything uploads to the real hub, because it wipes data on the site. Findings 2 and 3 are wrong behaviour a member will hit, and the rest are small. 9 findings.

Range reviewed: `4f41e79...27e1a08`, `docs/reviews` excluded. Findings 1 to 3 and 4's `sync: ok` were each reproduced by a probe test run in a scratch copy of the head (not committed). The parser also survived 60 s of `FuzzParse` (2.9M execs, no panic).

### 1. A character without skill lines is uploaded with `"skills": []`, and the hub then deletes that character's professions
- **file:line** `companion/internal/savedvars/read.go:133` always sets `Skills: []wire.Skill{}`. `companion/internal/wire/wire.go:39` has no `omitempty`, so `"skills":[]` goes out for every entry, including an entry whose SavedVariables table has no `skills` key. Contract §4 says: "The companion leaves `skills` out when the addon's entry has none." Code and contract disagree here.
- **Failure:** The addon writes no `skills` key in three cases:
  - the client's skill list is not ready at the read (§1 names this case);
  - any skill line is unusable;
  - `C_SkillInfo` is absent or shaped differently in Forever. Its index-vs-id question is still "verify in client".

  The probe gives `{"name":"A","realm":"R","class":"MAGE","level":3,"skills":[],"seen":5}`. On the real hub, `cleanEntry` in `/home/user/wt-hub-companion/src/companion.js:190-202` turns `[]` into `professions = ''`. `planUpload` then writes `'' ?? match.professions`, which is `''`. So every matched character's professions, typed on the site, are wiped by the first sync after linking and again by every later file change. If Forever's skill API differs, this hits every character of every member.
- **Confidence:** certain.
- **Fix:** add `json:"skills,omitempty"` and leave `Skills` nil unless the entry's list yields at least one line (`omitempty` also never sends `null`). Add a test with an entry that has no `skills` key, and one with `skills = {}`.

### 2. After the game folder changes, the stored `roster_etag` keeps the new folder's member list empty
- **file:line** `companion/internal/app/setup.go:81-87` saves a new `cfg.Game`. `setup.go:212-214` then has `dataaddon.EnsureEmpty` create an empty `Data.lua` there. Neither clears `state.RosterEtag`. `companion/internal/syncer/syncer.go:159-161` sends the etag whenever *a* `Data.lua` exists. Contract §3.3 says to send it only for "the member list the companion last wrote".
- **Failure:**
  1. Linked, synced once (etag stored).
  2. The game moves to another drive, and `setup --game D:\...\_forever_` is run.
  3. Every `run --once` afterwards sends `roster_etag` `0000000000000001`.
  4. The hub omits `roster`, and the new `Data.lua` stays at `members = {}` and `written = 0`.

  The probe asserted that "Anna" never appears there. The list stays empty until someone changes the list on the hub. The same happens when the AddOns content is deleted (Blizzard's UI reset) and setup runs before the next sync.
- **Confidence:** certain.
- **Fix:** clear `RosterEtag` whenever setup changes the game folder or `EnsureEmpty` creates `Data.lua`. Better: store a hash of the bytes written alongside the etag, and send the etag only while the file still holds those bytes.

### 3. A declined code is shown again on every run until it would have expired; "Run this again to get a new one" is false
- **file:line** `companion/internal/linkflow/linkflow.go:123-124` returns `ErrExpired` on 401 without clearing `cfg.Pending`. `linkflow.Resume` (`linkflow.go:44-54`) checks only the clock, so the next setup resumes the dead code.
- **Failure:**
  1. The member presses Decline, or the hub forgets the pending device.
  2. Setup prints "The code expired or was declined. Run this again to get a new one." and exits 2.
  3. Every re-run shows the same code, gets 401 at once, and prints the same line. The probe ran setup four times over three minutes: one link start, the same `BCDF-HBCD` each time.

  This lasts for the rest of the 10 minutes. Contract §3.1 says: "401 means the link expired or was declined; the companion starts over when the member asks."
- **Confidence:** certain.
- **Fix:** on 401, save `cfg.Pending = nil`, or set its expiry to now, before returning `ErrExpired`.

### 4. "Verify on the PC" step 14 cannot detect what it is for, and a file that does not parse is never reported anywhere
- **file:line** `companion/README.md:137-139`; `companion/internal/savedvars/read.go:45` (`Problem` is set and never read); `companion/internal/app/run.go:102-121`.
- **Failure:** Step 14 says `run --once` "should print `sync: ok` either way". It cannot work as written, for three reasons:
  - The probe wrote a file that does not parse. Result: `sync: ok`, exit 0, a sync with no `characters`, and an empty `companion.log`. So the check passes exactly when the CRLF parse fails.
  - After setup, the background copy holds `run.lock`, so `run --once` exits 5. Certain.
  - The release build is `-H=windowsgui`, and `run` skips `PrepareConsole` (`companion/internal/app/app.go:83`). By the README's own item 1, it then prints nothing in a terminal. Likely.

  Outside the check, a real SavedVariables form the parser refuses would leave characters never uploaded, with no line in the log or in `status`.
- **Confidence:** certain for the first two parts; likely for the third.
- **Fix:** log a file's `Problem` once per new fingerprint, and show the parsed character count in `status`. Rewrite step 14 to read the log or `status`.

### 5. Resume-from-sleep is never detected on Linux or macOS
- **file:line** `companion/internal/syncer/syncer.go:106` measures the gap with `now.Sub(s.lastTick)` on `time.Now()` values, which compares monotonic readings. Go's own `time` docs (`time.go:54`) warn that the monotonic clock may stop while the computer sleeps. On Linux it is `CLOCK_MONOTONIC`, which does stop during suspend.
- **Failure:**
  1. A Linux laptop suspends for an hour. The gap is never above 60 s, so no grace period starts.
  2. Its network comes back slowly, and the ticks at wake and 30 s later both fail.
  3. An outage is reported to the hub, which is exactly what the §3.3 rule exists to prevent.

  The test passes only because its fake clock has no monotonic reading. Windows reads interrupt time, which includes sleep, so it is unaffected.
- **Confidence:** certain on Linux.
- **Fix:** `now := s.d.Now().Round(0)` in `Tick`, so the gap is measured on the wall clock.

### 6. While the game runs, retries after five failures are slower than the normal schedule
- **file:line** `companion/internal/syncer/syncer.go:214-218`: `RetrySlow` is 120 s. Contract §3.3 says: "After a failed request it tries again sooner, at most every 30 s." While running, the normal schedule is 60 s, so 120 s is not sooner.
- **Failure:** The hub is down for 5 minutes during play. Once it is back, the first success can take up to 120 s instead of 60 s, so the member's in-game status returns up to a minute late.
- **Confidence:** certain. Severity is low.
- **Fix:** use `min(RetrySlow, every(s.running))`.

### 7. With `--config-dir`, the autostart entry starts `run` against the default folder
- **file:line** `companion/internal/app/setup.go:309-310` calls `SetAutostart(target)`, which registers `"exe" run` (`companion/internal/platform/platform_windows.go:192`). `startBackground` (`setup.go:301-304`) does pass `--config-dir`.
- **Failure:** `setup --config-dir D:\ff` without `--no-autostart` works until the next logon. At logon, `run` opens `%LOCALAPPDATA%\ForeverFriends`, prints "not linked", exits 1, and the companion never runs.
- **Confidence:** certain. Severity is low.
- **Fix:** pass the args to `SetAutostart`, or refuse autostart with a custom config folder.

### 8. An interrupted first install is treated as a foreign folder, and `--yes` then leaves it broken
- **file:line** `companion/internal/install/install.go:163-174`. The comment on `copyFiles` (`install.go:201-203`) promises that "a half-finished copy ... is repaired by the next run". That holds only when an older `.toc` exists.
- **Failure:** On a first install, `copyFiles` fails after `Core.lua` is written (disk full, a file locked by antivirus, the window closed). The folder now holds `.lua` files and no `.toc`. On the next run:
  - setup asks "holds something that is not this addon. Replace it? [y/N]";
  - with `--yes` it reports "left alone: … you chose not to replace it" and exits 0.

  A folder without a `.toc` is never loaded by the game.
- **Confidence:** certain. Severity is low.
- **Fix:** treat a folder whose entries are all files the embedded `.toc` lists, or their `.tmp` files, as ours.

### 9. Two tests do not assert what their names say
- **file:line** `companion/internal/install/install_test.go:57` (`TestFreshInstallTocLast`); `companion/internal/app/app_test.go:337`.
- **Failure:**
  - `TestFreshInstallTocLast` checks only that the files exist. Writing the `.toc` first would still pass, although that order is what keeps a half-finished update loadable as the old version.
  - The "nothing outside the two folders" test (`app_test.go:337`) matches the prefix `Interface/AddOns/ForeverFriends`, so a write to `ForeverFriendsX/` would also pass.
- **Confidence:** certain.
- **Fix:** have the copy fail on the last listed file and assert the old `.toc` survives. Compare the second path segment exactly.

**What carried it:** The brief's pointer to the real hub's code. Reading `cleanEntry` in `/home/user/wt-hub-companion/src/companion.js`, at the moment `read.go:133`'s `[]` looked like a harmless format choice, showed that it wipes professions.
