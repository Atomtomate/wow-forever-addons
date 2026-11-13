-- Pure part: turns what the client returned into the entry of docs/CONTRACT.md §1. No client
-- globals (the lint enforces it), so busted loads it alone. A value that cannot be used is simply
-- not written and the field keeps what it had.
local _, ns = ...

ns.MAX_SKILLS = 40
ns.MAX_LOG = 50

-- Returns v when it is a usable value of the kind ("string", "level" >= 1, "count" >= 0), else nil.
-- A client value is never compared outside a pcall: comparing a secret raises, nil included, and
-- nil needs no test of its own because the probe raises on it too. The probe runs before type()
-- because a secret string reports the type "string". Concatenating a secret may be allowed and
-- yield another secret; comparing one is what raises, so the probe compares the result with v.
function ns.usable(v, kind)
    local ok, same
    if kind == "string" then
        ok, same = pcall(function() return (v .. "") == v end)
        if ok and same and type(v) == "string" and v ~= "" then
            return v
        end
        return nil
    end
    ok, same = pcall(function() return v + 0 == v and v == v end)
    if not (ok and same and type(v) == "number" and v ~= math.huge and v ~= -math.huge) then
        return nil
    end
    if v ~= math.floor(v) or v < (kind == "level" and 1 or 0) then
        return nil
    end
    return v
end

-- Merges one read into the character's entry in `characters` and returns key, entry, or nil when
-- there is nothing to write. `read` holds the raw client values (name, realm, class, race, level)
-- and `skills`, a list from collectSkills or nil. `remembered` ({ name, realm } or nil) names the
-- character the session already knew, for a read in which the client would not give the name now.
-- This is the only place the key is spelled and the old entry is found. `seen` moves only on a read
-- in which the client gave name, realm and class (contract §1).
function ns.buildEntry(characters, read, now, remembered)
    local name, realm = ns.usable(read.name, "string"), ns.usable(read.realm, "string")
    local named = name and realm
    if not named then
        name, realm = remembered and remembered.name, remembered and remembered.realm
    end
    if not (name and realm) then
        return nil
    end
    local key = name .. "-" .. realm
    local entry = {}
    for k, v in pairs(characters[key] or {}) do
        entry[k] = v
    end
    local class = ns.usable(read.class, "string")
    entry.name, entry.realm = name, realm
    entry.class = class or entry.class
    entry.race = ns.usable(read.race, "string") or entry.race
    entry.level = ns.usable(read.level, "level") or entry.level
    entry.skills = read.skills or entry.skills
    if not entry.class then
        return nil
    end
    if named and class and ns.usable(now, "count") then
        entry.seen = now
    end
    return key, entry
end

-- The non-header lines of the skill list in the client's order, the first MAX_SKILLS. `getLine(i)`
-- returns a table with skillID, name, isHeader, rank, maxRank. Returns the list and whether it was
-- cut, or nil, meaning "keep the old list": the count is unusable or 0 (a client not ready yet), or
-- any line could not be used, since a list with one line missing makes the hub drop that profession.
function ns.collectSkills(count, getLine)
    count = ns.usable(count, "count")
    if not count or count == 0 then
        return nil
    end
    local skills = {}
    for i = 1, count do
        -- Everything about the line is read inside the pcall, the header test included.
        local ok, id, name, header, rank, max = pcall(function()
            local t = getLine(i)
            return t.skillID, t.name, t.isHeader == true, t.rank, t.maxRank
        end)
        if not ok then
            return nil
        end
        if not header then
            if #skills >= ns.MAX_SKILLS then
                return skills, true
            end
            id, rank, max = ns.usable(id, "count"), ns.usable(rank, "count"), ns.usable(max, "count")
            if not (id and rank and max) then
                return nil
            end
            skills[#skills + 1] = { id = id, name = ns.usable(name, "string") or "", rank = rank, max = max }
        end
    end
    if #skills == 0 then
        return nil
    end
    return skills, false
end

-- Appends to the ring inside SavedVariables, so the log leaves the machine with the data.
function ns.pushLog(log, line)
    log[#log + 1] = line
    while #log > ns.MAX_LOG do
        table.remove(log, 1)
    end
end

-- A table the client loaded back whose version this addon does not know, or that is not a table, is
-- replaced by an empty one (contract §1). Otherwise it is kept, with the two tables made sure of.
function ns.normalise(db)
    if type(db) ~= "table" or db.version ~= 1 then
        return { version = 1, characters = {}, log = {} }
    end
    db.characters = type(db.characters) == "table" and db.characters or {}
    db.log = type(db.log) == "table" and db.log or {}
    return db
end
