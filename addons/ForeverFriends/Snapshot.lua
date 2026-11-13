-- Pure part: turns what the client returned into the entry of docs/CONTRACT.md §1. No client
-- globals, so busted loads it alone. The client decides nothing here; a value that cannot be used
-- is simply not written and the field keeps what it had.
local _, ns = ...

ns.MAX_SKILLS = 40
ns.MAX_LOG = 50

-- Returns v when it is a usable value of the kind ("string", "level" >= 1, "count" >= 0), else nil.
-- The probe runs first, under pcall, and type() only afterwards: a secret string reports the type
-- "string", so type() cannot tell it apart. Concatenating a secret may be allowed and yield another
-- secret; comparing one is what raises, so the string probe compares the concatenation with v.
function ns.usable(v, kind)
    if v == nil then
        return nil
    end
    local ok, same
    if kind == "string" then
        ok, same = pcall(function() return (v .. "") == v end)
        if not (ok and same and type(v) == "string" and v ~= "") then
            return nil
        end
        return v
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

local IDENTITY_FIELDS = { "name", "realm", "class", "race" }

-- Merges one read into the character's old entry. `read` holds the raw client values (name, realm,
-- class, race, level) and `skills`, an already collected list or nil. Returns key, entry, or nil
-- when there is nothing to write: no usable name and realm to key it by, or no class to say who it
-- is. `seen` moves only when this read named the character completely.
function ns.buildEntry(old, read, now)
    local name, realm = ns.usable(read.name, "string"), ns.usable(read.realm, "string")
    if not (name and realm) then
        return nil
    end
    local entry = {}
    for k, v in pairs(old or {}) do
        entry[k] = v
    end
    for _, field in ipairs(IDENTITY_FIELDS) do
        entry[field] = ns.usable(read[field], "string") or entry[field]
    end
    entry.level = ns.usable(read.level, "level") or entry.level
    entry.skills = read.skills or entry.skills
    if not entry.class then
        return nil
    end
    if ns.usable(read.class, "string") and ns.usable(now, "count") then
        entry.seen = now
    end
    return name .. "-" .. realm, entry
end

-- The non-header lines of the skill list in the client's order, the first MAX_SKILLS. `getLine(i)`
-- returns a table with skillID, name, isHeader, rank, maxRank. A line whose id, rank or maxRank is
-- unusable is dropped, never guessed. Returns the list and whether it was cut, or nil when there
-- is nothing to replace the old list with (count unusable or 0, every line dropped).
function ns.collectSkills(count, getLine)
    count = ns.usable(count, "count")
    if not count or count == 0 then
        return nil
    end
    local skills, truncated = {}, false
    for i = 1, count do
        -- Everything about the line is read inside the pcall, the header test included: comparing a
        -- secret raises on the real client.
        local ok, id, name, header, rank, max = pcall(function()
            local t = getLine(i)
            return t.skillID, t.name, t.isHeader == true, t.rank, t.maxRank
        end)
        if ok and not header then
            id, rank, max = ns.usable(id, "count"), ns.usable(rank, "count"), ns.usable(max, "count")
            if id and rank and max then
                if #skills >= ns.MAX_SKILLS then
                    truncated = true
                    break
                end
                skills[#skills + 1] = { id = id, name = ns.usable(name, "string") or "", rank = rank, max = max }
            end
        end
    end
    if #skills == 0 then
        return nil
    end
    return skills, truncated
end

-- Appends to the ring inside SavedVariables, so the log leaves the machine with the data.
function ns.pushLog(log, line)
    log[#log + 1] = line
    while #log > ns.MAX_LOG do
        table.remove(log, 1)
    end
end

-- A table the client loaded back is checked, not trusted. Anything of another version, or not a
-- table, is replaced by an empty one (contract §1); other characters are kept as they are.
function ns.normalise(db)
    if type(db) ~= "table" or db.version ~= 1 then
        return { version = 1, characters = {}, log = {} }
    end
    if type(db.characters) ~= "table" then
        db.characters = {}
    end
    for key, entry in pairs(db.characters) do
        if type(key) ~= "string" or type(entry) ~= "table" then
            db.characters[key] = nil
        end
    end
    local log = {}
    if type(db.log) == "table" then
        for _, line in ipairs(db.log) do
            if type(line) == "string" then
                log[#log + 1] = line
            end
        end
    end
    while #log > ns.MAX_LOG do
        table.remove(log, 1)
    end
    db.log = log
    return db
end
