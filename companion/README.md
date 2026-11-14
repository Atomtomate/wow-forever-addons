# The Forever Friends companion

> 2026-11-14: this Go program is now the reference implementation and test oracle; the companion
> members install will be the Rust and Tauri app in `companion-app/` (`docs/DECISIONS.md` D9).

One program per member's PC, written in Go with the standard library only (`docs/DECISIONS.md` D7).
The wire formats it speaks are `docs/CONTRACT.md`; this file is about running it.

## What it does, and what it never does

It links the PC to the group's site (the hub) through an approval the member gives in the browser,
installs the ForeverFriends addon it carries inside itself, uploads the member's characters from the
addon's SavedVariables file with signed requests, tells the site once a minute whether the game is
running, writes the site's member list where the addon can load it, and notices when the site cannot
be reached (it reports that stretch with the next request that gets through).

What it never does is `docs/DECISIONS.md` D7, and what crosses the wire and what does not is
`docs/CONTRACT.md` §5. In short: of the game's files it reads one, the addon's SavedVariables file, and
it writes only inside the two addon folders; it looks at which programs run, to see whether the game is
one of them, and at nothing inside them; it never updates itself, a new version is a new download.

## Setup, as a member sees it

Download `ForeverFriendsCompanion-windows-amd64.exe` from the site's `/companion` page, which also shows
the site's address, and double-click it. Windows SmartScreen warns once because the program is unsigned:
"More info", then "Run anyway".

```
Forever Friends companion 0.1.0. It links this PC to your group's site, installs the addon and tells
the site when the game runs. Of the game's files it reads only the addon's own; it looks at which
programs are running only to see whether the game is one of them.
Nothing changes until you answer.
Game folder: found C:\Program Files (x86)\World of Warcraft\_forever_ . Use it? [Y/n]
Site address (shown on the site's /companion page) []: https://forever.example
Name of this PC on the site [JULIAN-PC]:
Your code:  HKRT-BCDF
If no browser opened, go to https://forever.example/companion/link?code=HKRT-BCDF
Waiting for you to press Link on the site (10 minutes; Ctrl+C stops, run this again to continue)...
Linked as Anna.
Addon ForeverFriends 0.1.0: installed.
Restart the game once so that it loads the addon (a running game does not see a new folder).
Companion copied to C:\Users\Anna\AppData\Local\Programs\ForeverFriends.
It starts with Windows and is running now.
Press Enter to close, or type "remove" to unlink this PC and remove the companion:
```

Closing the window before pressing Link is fine: running it again shows the same code until it expires,
then asks before starting a new one; a declined or expired code is not shown again. "Nothing changes
until you answer" is said on a first run only. A later run of a newer download says what it will do
instead: it updates the addon and the program copy, stops the running copy first (up to 45 seconds) and
starts the new one.

**The installed version is the `## Version` of the installed `ForeverFriends.toc`**, which the addon and
the program share. A download older than that replaces neither the addon nor the program copy, says so in
one line, and goes on with everything else. An addon folder that is a link (a developer's checkout) is
left alone, and a folder that is not ours is replaced only on a "yes" that `--yes` never gives.

## Commands

`setup` (the default when double-clicked), `run` (the background command), `status`, `remove`, `version`.

Non-interactive, for scripts and for the end-to-end run against a hub on this machine:

```
companion setup --yes --hub http://127.0.0.1:3000 --game <game>/_forever_ --config-dir <dir> \
    --name TESTPC --no-browser --no-autostart --no-self-copy
companion run --once --config-dir <dir>
companion remove --yes --config-dir <dir>
```

`setup --yes` prints a line `CODE: HKRT-BCDF`, polls until the code is approved or expires, and exits 0
only when linked (2: expired or declined, 1: anything else). `run --once` does one sync and prints
`sync: ok|unlinked|unreachable|refused|throttled` with exit 0, 3, 4, 6, 7 (5: another companion runs).
`--yes` never answers the questions that guard something the member did not ask for (replacing a
folder that is not ours, replacing a newer program).

## On disk

| What | Windows | macOS and Linux |
|---|---|---|
| config folder | `%LOCALAPPDATA%\ForeverFriends\` | `os.UserConfigDir()/forever-friends/` |
| `config.json` | the site's address, device id and name, member, game folder, a pending link | same |
| `state.json` | sequence number, which files were uploaded, the member list's etag, "revoked" | same |
| `device.key` | the private key, PKCS8 PEM; never leaves the PC | same, mode 0600 |
| `companion.log`, `.1` | at most 256 KiB each, no keys, codes or character names | same |
| `run.lock`, `stop.request` | the one-instance lock, and how a setup run asks a running copy to leave | same |
| program copy | `%LOCALAPPDATA%\Programs\ForeverFriends\ForeverFriendsCompanion.exe` | none: it runs where it is |
| autostart | `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, value `ForeverFriendsCompanion` | none |

The config folder is local, not roaming: a roaming profile would copy the key to every PC of the account.

## Other systems

The group has Mac and Linux players (`docs/DECISIONS.md` D7). In version one the same program builds for
both and is started by hand: `companion setup`, then `companion run` in a terminal that stays open.
Linking, uploading characters and writing the member list work there; the program is **never shown as
in game** from a Mac, and from Linux only if the game is found, which is unverified under Wine and
Proton (it reads `/proc/<pid>/exe`, which there names the loader, not a file in the game folder). There
is no autostart on either. Finding the game on those systems, and autostart, are the next pull request.
`release.yml` publishes files for macOS (arm64 and amd64) and Linux; nobody has run the macOS ones, and
macOS will ask before it lets an unsigned program run.

## Releasing

The tag must be `v` plus the `## Version` line of `addons/ForeverFriends/ForeverFriends.toc`, and
`.github/workflows/release.yml` refuses any other. The addon and the companion share that one number, so a
companion-only fix still means raising that line. The workflow runs the tests, builds the files with
`-trimpath` (the Windows one for the windowed subsystem, so autostart shows no console), writes
`SHA256SUMS` and publishes with `gh release create`; run by hand (`workflow_dispatch`) it only keeps the
files as a run artifact.

## Verify on the PC

Nothing below can be proved by the tests in this repository (they run on Linux and, in CI, on a Windows
runner without a game); nothing has been run on a PC yet. Each is a check on the owner's PC.

1. **Double-click setup**: a console appears with the text above; from a terminal the same text appears in
   that terminal. `ForeverFriendsCompanion.exe version` prints the version in a terminal.
2. **Browser and login**: the browser opens on the code page and, when not logged in, returns to the same
   code after the login. `Ctrl+C`, run again: same code, no new start on the site. Press Decline on the
   site: the next run starts a new code.
3. **Game folder**: the Program Files guess is right, or the typed folder is accepted.
4. **The game's executable lies inside the folder**:
   `Get-Process | Where-Object { $_.Path -like "*\_forever_\*" } | Select-Object Path`; then `status`
   says "Game running now: true" within 15 seconds of starting the game and "false" after it exits, also
   when the game was started as administrator.
5. **No window at logon**: after a reboot no console flashes; the background copy is running
   (`Get-Process ForeverFriendsCompanion`, exactly one) and
   `reg query HKCU\Software\Microsoft\Windows\CurrentVersion\Run /v ForeverFriendsCompanion` names the
   program copy.
6. **Replacing the installed copy while it runs**: run a second download; `Get-Process ForeverFriendsCompanion`
   still shows one, started after setup; `stop.request` is gone.
7. **An older download**: run it over a newer install; it says the addon and the program copy were left
   alone, and `Get-Item` on both shows unchanged write times.
8. **Key file**: `icacls "%LOCALAPPDATA%\ForeverFriends\device.key"` lists only this user, SYSTEM and
   Administrators.
9. **A junction is left alone**: after `tools\link.ps1 ForeverFriends`,
   `Get-Item "<AddOns>\ForeverFriends" | Select-Object LinkType` says `Junction`; run setup; the output says
   "left alone" and the checkout is unchanged (`git status` there).
10. **`Data.lua` replaced while the game runs**: change the member list on the site; after `/reload` the
    addon shows it (`/ff members`). A first install needs one game restart.
11. **Sleep**: put the PC to sleep and wake it with the network down; the log shows no outage report later.
12. **SmartScreen and Defender** on the first download and the first run of an unsigned program that copies
    itself and adds a Run key; if Defender flags it, submit the file to Microsoft as a false positive.
13. **Remove**: `ForeverFriendsCompanion.exe remove`: the PC disappears from the profile on the site, the Run
    value, both addon folders and the config folder are gone, and the program copy's folder is named when
    it cannot delete itself. The characters stay on the site (delete them on your profile) and so does the
    game's own `ForeverFriends.lua`; the last lines of the output say so.
14. **The real `ForeverFriends.lua`**: the parser reads CRLF and LF and only the forms the client writes;
    after the game has written the file, `ForeverFriendsCompanion.exe status` shows "Characters in the
    addon's file: N" with N above zero and no "Cannot use" line. If it names a problem, that form needs
    handling; the log (`companion.log`) holds the same line once per rewrite.
15. **The hub behind Caddy**: `PUBLIC_URL` is `https://`, `TRUST_PROXY=1`, and Caddy passes the `X-FF-*`
    headers and the body unchanged: a link from a member's PC, then a sync, is the test.
