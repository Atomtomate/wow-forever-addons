# The game link, version one: the plan (impl-director, 2026-11-13, second pass)

Cut against `docs/CONTRACT.md` at `4b746f7` (linking and signing) and hub D17 at `3bf8787`. It is
the next thing in both repos (both `docs/IDEAS.md` "in progress"); nothing parked is needed. All
three lanes are **large**: plan-before-code round each, settled plans read by the two plan
reviewers. The test vector of §3.2 was re-derived here with Go 1.24 `crypto/ed25519`: identical.

## 1. Decisions for the owner

- **Answered** (relayed, 2026-11-13): a PC is linked by browser approval and signs every request
  (contract §3.1, §3.2); no bearer token, no link line. The companion's "remove" choice is in.
- **Nothing new needs the owner.** The design keeps the hub's two rules: `/api/companion/*` is
  the named exception to rule 1, and the approval page is a member page. Rule 2 and D10 hold
  (no time is stored or shown per PC beyond the 150 s presence stamp). It adds no dependency:
  Ed25519 is in Go's `crypto/ed25519` and Node's `node:crypto` (D5, D7). The one residual risk
  (a member talked into approving someone else's code) is bounded by the approval page in §2 and
  by Remove on the profile; I would not put it to the owner.

## 2. Contract corrections still open after `4b746f7`

Every first-pass correction is folded. These are new, all in §3 unless named.

- **§3 intro:** change "the hub's base URL must be `https://`" to "the hub's address is an origin:
  `https://host[:port]`, no path, query or fragment (loopback may be `http`)", because the signed
  `<PATH>` is the path the hub sees; with a path prefix in the address, the companion and the hub
  would hash different paths and every request would be 401.
- **§3.1 step 2:** remove `verify_url` from the answer; the companion opens `<hub address>/companion/link?code=<user_code>`
  itself, because the hub builds absolute URLs from `PUBLIC_URL` or the Host header
  (`baseUrlFor` in `src/app.js`), which behind Caddy may be `http://` or another name, and a
  companion that opens whatever URL the hub names would open any page a forged answer names.
- **§3.1 step 2:** add "`public_key` and `X-FF-Signature` are standard base64 with padding
  (RFC 4648 §4)" and "the hub answers 400 for a `public_key` it already holds; the companion
  makes a new key pair for every link it starts", because Go's `RawStdEncoding` and Node's
  `base64url` are each one keystroke away, and a re-used key would give two device rows one key.
- **§3.1 step 2, 429:** replace "too many links ... too many are pending" with numbers: "at most 5
  link starts per address in 15 minutes and at most 20 pending links on the hub; an expired
  pending link is deleted, not counted", because the hub test needs a number and the companion
  needs to know a 429 here means "try again later", not "broken". The per-address limit only
  works with `TRUST_PROXY=1` behind Caddy (verify on the PC).
- **§3.1 step 2:** add "the `link` body is at most 4 KiB (else 413)", since it is the one route
  anyone can call without a key.
- **§3.1 step 3:** add what the approval page shows and checks: "the PC's name, the code, and
  the line *Press Link only if this code is shown by the companion on your own PC*; a Link and a
  Decline button; the form posts the device id with the code, and the hub links only a pending,
  unexpired device whose code matches; a code already used or expired gets a plain message. The
  code is accepted in either case, with or without the dash or spaces." And: "Decline deletes
  the pending device." The contract says "declined" in step 4 but nowhere offers it.
- **§3.1 step 3:** add "the page survives the login round trip: a member who is not logged in
  comes back to the same code after logging in", because `POST /login` today always redirects to
  `/`, so a logged-out member would lose the code (hub constraint; the contract only states the
  observable behaviour).
- **§3.1 step 4:** add "the companion keeps the device id, the key, the code and the expiry in its
  config before it shows the code; reopened before the expiry it shows the same code and resumes
  polling; after it, it starts over. Autostart is registered only once `linked` is true",
  because a member who closes the window mid-link would otherwise leave an orphan device on the
  hub and a companion that never learns it was approved.
- **§3.1 unlink:** add "a pending device may call it too (cancel)". **`link/status` and `unlink`
  keep `{ "v": 1 }`** and answer 400 `unsupported version` like `sync`: one version rule for all
  signed routes, and an empty body would need its own hash case in both suites.
- **§3.2 headers:** add "`X-FF-Device` is 16 lowercase hex; `X-FF-Seq` is decimal ASCII, no sign,
  no leading zero, at most 15 digits; a missing or malformed header is 401", because Node reads
  numbers as doubles (exact only to 2^53) and "a positive integer" admits `+7`, `07` and `7.0`,
  each of which hashes differently from `7`.
- **§3.2 order of checks:** change "before it looks at the body" to "the hub reads the raw body
  (within the route's size limit, else 413 first), hashes those bytes, runs the three checks, and
  parses JSON only after them", because a hub that parses with `express.json` first and
  re-serializes to hash will never match the companion's bytes.
- **§3.2:** add "the hub accepts a sequence number with one atomic update (`last_seq < seq`), so two
  requests with one number cannot both pass", and "a pending device that has expired counts as
  not existing (401)".
- **§3.2, 409:** change "on 409 it continues from the returned number plus one and sends again"
  to "...and sends again once; a second 409 in a row counts as a failed request. The companion
  sends one request at a time and stores the next number before it sends", because two
  processes with one key (a second instance, a copied config) would otherwise leapfrog each
  other for ever, and overlapping requests (the scheduled sync and the one at game start) would
  arrive out of order and 409 each other.
- **§3.3, 401 row:** split it: "on `link/status`: the link expired or was declined, start over
  when the member asks; on `sync`: this PC is no longer linked". As written, one row covers both
  and the §3.1 step 4 text says something different for the same status.
- **§5 and addons D8:** add "the PC's name and its public key" to what crosses to the hub. D8 says
  "to the hub: characters ... and whether the game is running. Nothing else", which a reviewer
  would now read as broken (the session folds; D8 is append-only, so a sentence in its new
  bullet, not an edit of the old one).

## 3. Lanes, owners, order

| Lane | Repo, branch, PR | Specialist | Size |
|---|---|---|---|
| H, hub side | hub, `lane-m-companion`, #12 | `impl-backend` (hub overlay: route, table and page are one lane) | large |
| A, the addon | addons, `lane/foreverfriends`, #3 | `impl-frontend` | large |
| C, the companion | addons, `lane/companion`, new PR stacked on #3, "merge after #3" | `impl-backend` | large |

0. The session folds §2 into `docs/CONTRACT.md` (PR #3) and commits. **Nobody starts before.**
1. The three plans in parallel, each in its own worktree; checked here; then the two plan
   reviewers. At most two opus runs at once, so two waves (H and C first: they share §3).
2. Build, in parallel. A is untouched by linking. C depends on A only through `embed.go`: the
   install logic takes an `fs.FS`, tested on `fstest.MapFS`. **A's first commit is
   `ForeverFriends.toc`** (`## Interface: 16001`, `## Version: 0.1.0`,
   `## OptionalDeps: ForeverFriendsData`, `## SavedVariables: ForeverFriendsDB`, the file list);
   C merges `lane/foreverfriends` once it lands and adds `embed.go` last.
3. End to end in this container once H and C are built (§6); then the four reviewers per PR.
4. Merge #12 (deployed first, so a companion has something to link to), #3, the companion PR;
   the owner tags `v0.1.0` on `main`.

No file is shared across lanes (`.luacheckrc` is A's, `Makefile` and `.github/` C's); the
contract and both `DECISIONS.md` are the session's.

## 4. The hand-offs, concretely

**H (hub).** Sketch unless marked constraint.
- Constraint: the four routes under `/api/companion/` sit before the session and CSRF middleware,
  never read the cookie, read the raw body with their own limit (4 KiB for `link`, 128 KiB for
  the rest), and answer every outcome, a parse error and an unknown path under the prefix
  included, as JSON with a status: never a redirect, never the HTML 404 or error page.
- Constraint: verification is `node:crypto`'s `verify(null, message, key, signature)` on a key
  made from the stored 32 bytes (JWK `{kty:'OKP', crv:'Ed25519', x}` or the SPKI prefix); the
  §3.2 vector is a unit test, both directions (verifies; one flipped byte does not).
- Pure module `src/companion.js`: the signed string, header shapes, body validation, the three
  tables of §4, merge decisions, roster shape and etag, user codes; `now` passed in.
- Schema, additions only. **One `devices` table** (confirming the session's sketch): `id` (16 hex),
  `user_id` (NULL while pending, `ON DELETE CASCADE`), `name`, `public_key` (UNIQUE), `user_code`
  and `expires_at` (set while pending, cleared on approval; unique among pending), `last_seq`
  (0), `companion_version`, `game_seen_at` (isoZ, cleared on `game_running: false`). **No column on
  `users`.** Pending and linked share the row and the signing path: `link/status` must be signed
  by the pending key, which is what proves the poller is the PC that showed the code; only
  `sync` additionally requires `user_id`. No `created_at` shown, no "last used" column (D10).
  `characters.from_game` through a new `CHARACTER_COLUMNS_ADDED_LATER`; the outage table
  (`from`, `to`, `received_at`, no user or device), newest 100 kept.
- `/companion/link` (member page, gating list): the approval form of §2, posting with CSRF to a
  member route; Decline. Login carries a same-site return path (a path starting with a single
  `/`, nothing else) so the code survives it.
- Profile "Game link" card: the member's PCs by name, each with Remove; no times. `/companion`:
  what it is, sends and never sends (§5), the site's address to type in, the download link
  (Admin setting); its copy invites, nothing nags elsewhere.
- Right now strip: "In game: Anna, Ben" when any of a member's devices has `game_seen_at` younger
  than 150 s, as of page load (no new poll); counts as content for "Quiet right now". Admin: the
  newest twenty outages, no member, labelled as seen from members' PCs.
- README (the routes, `TRUST_PROXY=1` behind Caddy for the link limit, the download setting) and
  `docs/IDEAS.md`.

**A (addon).** Unchanged by linking.
- Files: pure snapshot builder (values in, §1 entry out; nil or secret leaves the old value),
  pure member-list reader (type-checks every field of `ForeverFriendsData`, case-insensitive
  `who`), pure line formatting; one thin event file. Events: `PLAYER_LOGIN`, `PLAYER_LEVEL_UP`
  (use its argument, `UnitLevel` can lag), `SKILL_LINES_CHANGED`, `PLAYER_LOGOUT`. `/ff`,
  `/ff members`, `/ff who <name>`; no chat at login; no time about any member shown.
- Constraint: the secret check must not use `type()`: the stub's secret is a table, the client's
  is not. Use `C_Secrets.ShouldUnitIdentityBeSecret("player")` and a `pcall` around first use.
- `spec/support/wow.lua` gains `UnitClass`, `UnitRace`, `UnitLevel`, `UnitName`, `C_SkillInfo`,
  `C_Secrets` and a per-test secret identity; `.luacheckrc` lists the new globals.

**C (companion).** Sketch unless marked constraint.
- Layout as briefed: root `go.mod` and `embed.go`, thin `package main` in `companion/`, logic in
  `companion/internal/...` taking file system, clock, process list, HTTP client and key as
  parameters. Windows code behind `_windows.go` / `_other.go`.
- Constraint: standard library only. Windows calls with no `syscall` wrapper go through
  `syscall.NewLazyDLL` or `reg.exe`; opening the browser through `rundll32 url.dll,FileProtocolHandler`
  or `cmd /c start`; `golang.org/x/sys` would need the owner (D7).
- One package owns signing (the string, the headers, the sequence store, the 409 rule) and is
  the only thing that sends; its test asserts the §3.2 vector. The HTTP client has
  `CheckRedirect` returning `http.ErrUseLastResponse`.
- The link flow of §3.1 with §2's resume rule; the key file user-only (`0600`; on Windows the
  per-user config folder, verify on the PC). "Remove" calls `unlink`, then deletes key, config,
  autostart and the two addon folders unless a junction.
- SavedVariables parser for the client's subset, tested against `tools/parse_sv.py`'s sample;
  installer of exactly the `.toc` and its files, junctions left alone; `ForeverFriendsData`
  written atomically under §2; version from the embedded `## Version`, the release refusing a
  tag that is not `v` plus that line.
- One background instance per user; a setup run stops it before replacing the installed copy,
  only in a run the member started (D7: no self-update). A non-interactive mode with explicit
  config path, hub address and game folder for §6's end-to-end run.
- CI: a Go job on ubuntu and windows (`go vet`, `go test`, the Windows cross-build); `release.yml`
  on tags using `gh release create`.

## 5. Not in scope

The addon list, a tooltip line, in-game presence, the character's name next to "in game",
self-update, a tray icon, autostart on macOS and Linux, two members on one PC, deleting
characters through a sync, a realm on a character, auto-claimed trophies, a limit on signed
requests, any display of a sync, link or session time, renaming a PC on the hub.

## 6. Verification

- H: `npm test` plus: the §3.2 vector; gating (each `/api/companion/*` route answers JSON 401
  without headers and with a valid session cookie and no signature, never 302; an unknown path
  under the prefix is JSON 404; `/companion` and `/companion/link` redirect without a session);
  link (400 on a 31-byte key and on a known key, 429 at the sixth start, expiry, Decline, a used
  code); signing (bad signature, stale and equal seq give 409 with `seq`, malformed `X-FF-Seq`);
  sync rules of §3.3 and §4; presence at 149 s and 151 s, after `false`, two devices; Remove.
- A: `make lint test`; secret identity, the 40-skill cap, the 50-line ring, garbage
  `ForeverFriendsData` never errors.
- C: `go vet ./... && go test ./...`, the Windows cross-build; the §3.2 vector; per status-table
  row with `httptest` (no redirect followed, one resend on 409 then failure, outage after two
  failures, 401 stops, 400 marks uploaded); link resume after a restart; escaping of every byte
  0 to 255 round-tripped through `lua5.1` where it is on PATH.
- End to end, here: the hub on `127.0.0.1` with a temp database and one member; the companion's
  non-interactive mode starts a link; a test session logs in and approves the code through the
  hub's own form (GET `/companion/link?code=`, POST with the page's CSRF value, the same way
  `test/helpers.js` drives forms); the companion's next poll sees `linked`, its sync uploads a
  fixture SavedVariables sample, characters show on `/members`, and
  `lua5.1 -e 'dofile("<fixture>/Interface/AddOns/ForeverFriendsData/Data.lua")'` loads; Remove on
  the profile, and the next sync is 401.

**Verify in client** (A's README and `docs/HANDOFF.md`): `/dump UnitClass("player")`,
`/dump UnitRace("player")` (the Skyborne tokens), `/dump C_Secrets.ShouldUnitIdentityBeSecret("player")`,
`/dump issecretvalue`, `/run for i=1,C_SkillInfo.GetNumSkillLines() do local s=C_SkillInfo.GetSkillLineInfo(i) print(s.skillID,s.name,s.isHeader) end`
(the profession ids of §4), `/dump ForeverFriendsData and #ForeverFriendsData.members`,
`/dump ForeverFriendsDB` after a relog (research §1.3); a new addon folder needs a restart.

**Verify on the PC** (C's README and `docs/HANDOFF.md`): the double-click setup; the browser opens
on the code and survives a login; the game folder; the executable,
`Get-Process | Where-Object { $_.Path -like "*\_forever_\*" } | Select-Object Path`;
"in game" within 15 s of start, gone at exit; autostart after a reboot with no window
(`reg query HKCU\Software\Microsoft\Windows\CurrentVersion\Run`); SmartScreen and Defender on an
unsigned program that copies itself and adds a Run key; the key file's permissions
(`icacls "%APPDATA%\<config folder>\<key file>"`); replacing the installed copy while the
background run lives; `Data.lua` replaced while the game runs; a `tools\link.ps1` junction left
alone; remove. Hub on the PC: `PUBLIC_URL` is `https://`, `TRUST_PROXY=1`, and Caddy passes the
`X-FF-*` headers and the body unchanged (a link from a member's PC, then a sync, is the test).
