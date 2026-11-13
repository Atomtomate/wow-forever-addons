-- Pure part: reads the member list the companion wrote (docs/CONTRACT.md §2). Every string in it was
-- typed by people on the hub and the file is executed by the game, so the table is untrusted: this
-- file only looks at it, with rawget, in bounded loops, and skips what does not fit.
local _, ns = ...

local MAX_MEMBERS = 100
local MAX_CHARACTERS = 60 -- the hub's own cap per member
local MAX_MEMBER_NAME = 160 -- bytes: a 40-character display name in UTF-8
local MAX_CHARACTER_NAME = 64

local function text(value, max)
    if type(value) == "string" and #value >= 1 and #value <= max then
        return value
    end
end

local function level(value)
    if type(value) == "number" and value == math.floor(value) and value >= 1 and value ~= math.huge then
        return value
    end
end

local function readCharacter(raw)
    local name = type(raw) == "table" and text(rawget(raw, "name"), MAX_CHARACTER_NAME)
    if not name then
        return nil
    end
    return {
        name = name,
        class = text(rawget(raw, "class"), MAX_CHARACTER_NAME),
        level = level(rawget(raw, "level")),
        main = rawget(raw, "main") == true,
    }
end

local function readMember(raw)
    local name = type(raw) == "table" and text(rawget(raw, "name"), MAX_MEMBER_NAME)
    if not name then
        return nil
    end
    local member = { name = name, characters = {} }
    local list = rawget(raw, "characters")
    if type(list) == "table" then
        for i = 1, MAX_CHARACTERS do
            local raw_character = rawget(list, i)
            if raw_character == nil then
                break
            end
            member.characters[#member.characters + 1] = readCharacter(raw_character)
        end
    end
    return member
end

local function read(data)
    if type(data) ~= "table" or rawget(data, "version") ~= 1 then
        return nil, "version"
    end
    local list = rawget(data, "members")
    if type(list) ~= "table" then
        return nil, "unreadable"
    end
    local members = {}
    for i = 1, MAX_MEMBERS do
        local raw = rawget(list, i)
        if raw == nil then
            break
        end
        members[#members + 1] = readMember(raw)
    end
    if #members == 0 then
        return nil, "empty"
    end
    return members
end

-- Returns the members as plain tables, or nil and why: "missing" (no data addon), "version" (not a
-- table, or a version this addon does not know), "unreadable", "empty" (the companion's first file,
-- or nothing readable in it: the same as no list yet). Raises on nothing a table can hold, because
-- it only type-checks and rawgets.
function ns.readRoster(data)
    if data == nil then
        return nil, "missing"
    end
    return read(data)
end

function ns.countCharacters(members)
    local n = 0
    for _, member in ipairs(members) do
        n = n + #member.characters
    end
    return n
end

-- Members whose name, or any character's name, starts with the query. The query is plain text, never
-- a pattern. Folding is string.lower's: ASCII only, so umlauts match only in the same case.
function ns.matchMembers(members, query)
    local q = query:lower()
    local found = {}
    if q == "" then
        return found
    end
    local function starts(s)
        return s:lower():sub(1, #q) == q
    end
    for _, member in ipairs(members) do
        local hit = starts(member.name)
        for _, character in ipairs(member.characters) do
            hit = hit or starts(character.name)
        end
        if hit then
            found[#found + 1] = member
        end
    end
    return found
end
