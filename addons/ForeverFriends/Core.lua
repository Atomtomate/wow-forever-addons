-- Client-facing part, kept thin. Reads the character being played and nothing else (D5): the only
-- client calls are UnitName, UnitClass, UnitRace, UnitLevel, GetRealmName, GetServerTime,
-- C_Secrets.ShouldUnitIdentityBeSecret and C_SkillInfo's two readers. Nothing here acts for the player.
local addonName, ns = ...

local SKILL_DELAY = 2 -- seconds; SKILL_LINES_CHANGED fires on every weapon skill-up, also in combat

local frame = CreateFrame("Frame")
local session = {} -- name and realm of this session's character, once a read has named them
local told = {} -- failures already logged this session, so a repeating one cannot flush the ring
local skillTimerPending = false

-- Nobody has run this on the real client, and an error at login is the one thing this addon must
-- never cause: a client call that raises reads as "no value".
local function attempt(fn, ...)
    local ok, first, second = pcall(fn, ...)
    if ok then
        return first, second
    end
end

local function log(message)
    ns.pushLog(ForeverFriendsDB.log, date("%Y-%m-%d %H:%M:%S") .. " " .. message)
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
    return attempt(function()
        return secrets.ShouldUnitIdentityBeSecret("player") == true
    end)
end

local function readSkills()
    local skills, truncated
    if C_SkillInfo then
        skills, truncated = ns.collectSkills(attempt(C_SkillInfo.GetNumSkillLines), C_SkillInfo.GetSkillLineInfo)
    end
    if truncated then
        logOnce("truncated", "skills: more than " .. ns.MAX_SKILLS .. " lines, the first " .. ns.MAX_SKILLS .. " kept")
    end
    if not skills then
        logOnce("skills", "skills: no usable list, the old one kept")
    end
    return skills
end

-- Reads the player into the character's entry. A value the client will not give is simply not
-- written. Returns the key, or nil when nothing was recorded.
local function refresh(levelOverride)
    local read = { realm = attempt(GetRealmName) }
    if identityRestricted() then
        logOnce("restricted", "identity is secret: name, class and race not read")
    else
        read.name = attempt(UnitName, "player")
        local _, class = attempt(UnitClass, "player")
        local _, race = attempt(UnitRace, "player")
        read.class, read.race = class, race
    end
    read.level = ns.usable(levelOverride, "level") or attempt(UnitLevel, "player")
    read.skills = readSkills()
    local key, entry = ns.buildEntry(ForeverFriendsDB.characters, read, attempt(GetServerTime), session)
    if not key then
        logOnce("nothing", "nothing recorded: the character could not be named")
        return nil
    end
    ForeverFriendsDB.characters[key] = entry
    session.name, session.realm, session.key = entry.name, entry.realm, key
    return key, entry
end

local function describe(key, entry)
    return key .. (entry.level and " level " .. entry.level or "")
end

local function onAddonLoaded(name)
    if name == addonName then
        ForeverFriendsDB = ns.normalise(ForeverFriendsDB)
    end
end

local function onLogin()
    local key, entry = refresh()
    if key then
        log("login: " .. describe(key, entry))
    end
end

local function onLevelUp(level)
    local key, entry = refresh(level)
    if key then
        log("level up: " .. describe(key, entry))
    end
end

local function onSkillLinesChanged()
    if skillTimerPending then
        return
    end
    skillTimerPending = true
    C_Timer.After(SKILL_DELAY, function()
        skillTimerPending = false
        refresh()
    end)
end

local function onLogout()
    if refresh() then
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
    for _ in pairs(ForeverFriendsDB.characters) do
        n = n + 1
    end
    return n
end

SLASH_FOREVERFRIENDS1 = "/ff"
SLASH_FOREVERFRIENDS2 = "/foreverfriends"
SlashCmdList["FOREVERFRIENDS"] = function(message)
    print(ns.respond(message, ForeverFriendsData, { count = recorded(), key = session.key }))
end
