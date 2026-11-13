-- Client-facing part, kept thin. Reads the character being played and nothing else (D5): the only
-- client calls are UnitName, UnitClass, UnitRace, UnitLevel, GetRealmName, GetServerTime,
-- C_Secrets.ShouldUnitIdentityBeSecret and C_SkillInfo's two readers. Nothing here acts for the player.
local _, ns = ...

local SKILL_DELAY = 2 -- seconds; SKILL_LINES_CHANGED fires on every weapon skill-up, also in combat

local frame = CreateFrame("Frame")
local session = {} -- name, realm and key of this session's character, once a read has named them
local told = {} -- failures already logged this session, so a repeating one cannot flush the ring
local skillTimerPending = false

local function db()
    if not ForeverFriendsDB then
        ForeverFriendsDB = ns.normalise(nil)
    end
    return ForeverFriendsDB
end

local function log(message)
    ns.pushLog(db().log, date("%Y-%m-%d %H:%M:%S") .. " " .. message)
end

local function logOnce(kind, message)
    if not told[kind] then
        told[kind] = true
        log(message)
    end
end

-- Asked at every refresh: identity may be restricted only in some situations. An error or a missing
-- C_Secrets means "go on"; ns.usable is the defence either way. The comparison is inside the pcall
-- because the answer itself might be a secret.
local function identityRestricted()
    local secrets = C_Secrets
    if not (secrets and secrets.ShouldUnitIdentityBeSecret) then
        return false
    end
    local ok, restricted = pcall(function()
        return secrets.ShouldUnitIdentityBeSecret("player") == true
    end)
    return ok and restricted
end

local function readIdentity(read)
    local ok, name = pcall(UnitName, "player")
    if ok then
        read.name = name
    end
    local _, class
    ok, _, class = pcall(UnitClass, "player")
    if ok then
        read.class = class
    end
    local race
    ok, _, race = pcall(UnitRace, "player")
    if ok then
        read.race = race
    end
end

local function readSkills()
    if not C_SkillInfo then
        return nil
    end
    local ok, count = pcall(C_SkillInfo.GetNumSkillLines)
    if not ok then
        return nil
    end
    local skills, truncated = ns.collectSkills(count, C_SkillInfo.GetSkillLineInfo)
    if truncated then
        logOnce("truncated", "skills: more than " .. ns.MAX_SKILLS .. " lines, the first " .. ns.MAX_SKILLS .. " kept")
    end
    return skills
end

local function oldEntry(read)
    local name, realm = ns.usable(read.name, "string"), ns.usable(read.realm, "string")
    return name and realm and db().characters[name .. "-" .. realm]
end

-- Reads the player into the character's entry. A value the client will not give is simply not
-- written. Returns the entry, or nil when nothing was recorded.
local function refresh(levelOverride, withSkills)
    local read = {}
    local ok, realm = pcall(GetRealmName)
    if ok then
        read.realm = realm
    end
    if identityRestricted() then
        logOnce("restricted", "identity is secret: name, class and race not read")
    else
        readIdentity(read)
    end
    local ok2, level = pcall(UnitLevel, "player")
    if ok2 then
        read.level = level
    end
    if levelOverride ~= nil then
        read.level = ns.usable(levelOverride, "level") or read.level
    end
    if ns.usable(read.name, "string") and ns.usable(read.realm, "string") then
        session.name, session.realm = read.name, read.realm
    else
        -- the same character, which the client will not name just now
        read.name, read.realm = session.name, session.realm
    end
    if withSkills then
        read.skills = readSkills()
        if not read.skills then
            logOnce("skills", "skills: no usable lines, the old list kept")
        end
    end
    local key, entry = ns.buildEntry(oldEntry(read), read, GetServerTime())
    if not key then
        logOnce("nothing", "nothing recorded: the character could not be named")
        return nil
    end
    session.key = key
    db().characters[key] = entry
    return entry
end

local function describe(entry)
    return session.key .. (entry.level and " level " .. entry.level or "")
end

local function onAddonLoaded(name)
    if name == "ForeverFriends" then
        ForeverFriendsDB = ns.normalise(ForeverFriendsDB)
    end
end

local function onLogin()
    local entry = refresh(nil, true)
    if entry then
        log("login: " .. describe(entry))
    end
end

local function onLevelUp(level)
    local entry = refresh(level, false)
    if entry then
        log("level up: " .. describe(entry))
    end
end

local function onSkillLinesChanged()
    if skillTimerPending then
        return
    end
    skillTimerPending = true
    C_Timer.After(SKILL_DELAY, function()
        skillTimerPending = false
        refresh(nil, true)
    end)
end

local function onLogout()
    if refresh(nil, true) then
        log("logout")
    end
end

local handlers = {
    ADDON_LOADED = onAddonLoaded,
    PLAYER_LOGIN = onLogin,
    PLAYER_LEVEL_UP = onLevelUp,
    SKILL_LINES_CHANGED = onSkillLinesChanged,
    PLAYER_LOGOUT = onLogout,
}

frame:SetScript("OnEvent", function(_, event, ...)
    handlers[event](...)
end)
for event in pairs(handlers) do
    frame:RegisterEvent(event)
end

local function recorded()
    local n = 0
    for _ in pairs(db().characters) do
        n = n + 1
    end
    return n
end

SLASH_FOREVERFRIENDS1 = "/ff"
SLASH_FOREVERFRIENDS2 = "/foreverfriends"
SlashCmdList["FOREVERFRIENDS"] = function(message)
    print(ns.respond(message, ForeverFriendsData, { count = recorded(), key = session.key }))
end
