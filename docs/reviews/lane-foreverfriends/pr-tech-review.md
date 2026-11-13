**Verdict:** What `Snapshot.lua` writes is `docs/CONTRACT.md` §1 and what `Roster.lua` reads is §2 within §3.3's bounds, and no member-list or typed text reaches chat without `ns.clean`. One defect: the secret defence compares a possibly-secret value with `nil` outside its pcall, in two places. Three test gaps: the specs cannot see that defect, would stay green if the probe were deleted, and do not exercise `cut`'s limit handling.

Reviewed: `c890b92` against `d7dc548`. Mutations ran in a scratch copy (`git archive c890b92`), never in this checkout.

### 1. Medium: a possibly-secret value is compared with `nil` outside any pcall

- **`addons/ForeverFriends/Snapshot.lua:14`** (`if v == nil then`) and **`addons/ForeverFriends/Core.lua:99`**
  (`if levelOverride ~= nil then`) compare the raw client value before the probe runs. Under
  the model in `Snapshot.lua:10-12` and the brief, comparing a secret raises.
  `Core.lua:39-41` pcalls a `== true` comparison for exactly that reason.
- **Failure:** on the real client, any secret that reaches `ns.usable` raises at line 14, not inside the
  pcall at line 19 or 25. Three paths the code says it handles:
  (a) identity is secret while `C_Secrets` is missing, raises, or itself answers with a secret.
  `identityRestricted()` then returns false and `readIdentity` stores the secret name. The
  comment at `Core.lua:31-32` says "`ns.usable` is the defence either way". `refresh` then
  raises at `Core.lua:102`.
  (b) a secret skill name (`Snapshot.lua:88`). `collectSkills` raises, so `readSkills` raises, so
  every `refresh(nil, true)` aborts before `buildEntry`. Nothing is recorded at login, after
  skill changes or at logout, and each of those events throws a Lua error.
  (c) a secret `PLAYER_LEVEL_UP` argument raises at `Core.lua:99`.
  The specs that model these inputs pass only because the stub's `__eq` is never consulted
  against `nil`. Those are `addons/ForeverFriends/spec/core_spec.lua:151`, `:209`, and
  `addons/ForeverFriends/spec/snapshot_spec.lua:35-39` and `:125-129`. In lua5.1,
  `pcall(function() return wow.secret() == nil end)` returns `true, false`.
- **Confidence:** likely. It is certain under the stated model (comparison raises), and only the
  client could make an exception for `== nil`. The fix costs nothing either way.
- **Fix:** delete `Snapshot.lua:14-16`. Both probes already reject `nil`, because `nil .. ""` and
  `nil + 0` raise inside the pcall. Make `Core.lua:99-101` read
  `read.level = ns.usable(levelOverride, "level") or read.level`. With both changes the suite
  still passes 91/91 (checked in the scratch copy).

### 2. Medium (test gap): no spec would fail if `ns.usable`'s secret probe were removed

- **`addons/ForeverFriends/Snapshot.lua:19,25`** run the probe before `type()` because on the client
  a secret's `type()` is its underlying type. Every spec secret is a table, so
  `type(v) == "string"` rejects it before the probe matters.
- **Failure:** replace both probes with bare checks:
  `if not (type(v) == "string" and v ~= "")` and
  `if not (type(v) == "number" and v == v and ...)`. All 91 specs stay green. On the client
  that version would store a secret string in SavedVariables.
  `addons/ForeverFriends/spec/snapshot_spec.lua:41-47` is named "allows concatenation but not
  comparison", but its comparison never runs. Lua 5.1 calls `__eq` only when both operands
  share the same handler. The concat result carries the stub's `boom` and `later` carries its
  own function, so `==` returns false with no call (checked: `pcall` gives `true, false` and
  the handler never runs). The spec passes on `same == false` and on `type(later) == "table"`.
- **Confidence:** certain.
- **Fix:** `Snapshot.lua` reads the global `type` at call time. A spec can swap `_G.type` so that a
  `wow.secret()` reports `"string"` or `"number"`, then assert `ns.usable` returns nil. With
  that, the mutant above stores the string secret and raises on the number secret, and the
  current code returns nil for both (checked in the scratch copy).

### 3. Low (test gap): `cut`'s UTF-8 and lone-`|` handling has no spec that reaches it

- **`addons/ForeverFriends/Lines.lua:137-150`**: the code is right today. Across 63 cut points through
  `/ff status` the result is valid UTF-8, has even trailing pipes and is at most 240 bytes.
  Nothing would catch it breaking.
- **Failure:** replace `cut`'s body with `return s:sub(1, max)`. All 91 specs stay green.
  `addons/ForeverFriends/spec/lines_spec.lua:79-84`, the only long-text case, cuts at byte 240.
  That byte is the second byte of a complete "é": the prefix is 16 bytes, the name 160, the
  `": "` 2, which puts byte 240 at offset 61 of the é run. The 128-byte pipe run of `class`
  lies past the cut, so `#line:match("|*$") % 2` counts zero pipes.
- **Confidence:** certain.
- **Fix:** take `ns.respond("status", nil, { count = 1, key = k })`, which goes straight through
  `cut`. Use `k` = `("x"):rep(n) .. ("é"):rep(200)` and `k` = `("x"):rep(n) .. ("|"):rep(300)`
  for `n` = 0 to 3, and assert valid UTF-8 and an even pipe tail.

### 4. Low: two spec names claim what their bodies do not assert

- **`addons/ForeverFriends/spec/core_spec.lua:181`** is named "records nothing ... keeping the old
  entry", but line 191 asserts that the level-up wrote level 24 to the entry, and that event
  also logs a `level up:` line. **`core_spec.lua:155`** says "two seconds", but the stub's
  `C_Timer.After` (`spec/support/wow.lua:77`) drops the delay, so `SKILL_DELAY = 0` would pass.
- **Failure:** a reader who trusts the name of :181 believes a restricted identity freezes the entry,
  which is the opposite of what the code does and the spec checks. Nothing guards the delay
  that `Core.lua:6` justifies.
- **Confidence:** certain.
- **Fix:** rename :181 to say that only level moves. Have the `C_Timer` stub record the delay so
  :155 can assert 2.

**Follow-ups**

- `spec/support/wow.lua:9-10` (lines this PR did not change) says the stub raises "on every such
  operation". In Lua 5.1 its `__eq` and `__lt` run only between two tables that share the same
  handler, so a comparison with `nil`, a string, a number or a boolean never raises. That is
  why findings 1 and 2 are invisible to the suite. MarketScan's specs use the same stub.

**What carried it**

- The brief's note that Lua 5.1 consults `__eq` only between tables. On reading
  `Snapshot.lua:14`, it said at once why a comparison the code treats as raising elsewhere
  could not fail any spec. The scratch-copy mutations then turned that into findings 1 to 3.
