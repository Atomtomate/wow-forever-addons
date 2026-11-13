-- Pure part: the one line of chat each /ff command answers with. Everything that came from the member
-- list or from the player's typing goes through clean() before it is shown.
local _, ns = ...

local PREFIX = "ForeverFriends: "
local LINE_BUDGET = 240 -- bytes `fit` aims for; a longer line only wraps in chat
local SUFFIX_RESERVE = 20 -- room for " and 999 more"
local USAGE = "/ff, /ff members, /ff who <name>, /ff status."

-- Drops control characters and doubles every |, so a name cannot carry a colour code or hyperlink.
function ns.clean(s)
    return (s:gsub("[%z\1-\31\127]", ""):gsub("|", "||"))
end

-- As many parts as fit in the budget, then "and N more". Always at least one part.
local function fit(parts, separator, budget)
    local out = ""
    for i, part in ipairs(parts) do
        local candidate = out == "" and part or out .. separator .. part
        if i > 1 and #candidate > budget - SUFFIX_RESERVE then
            return out .. " and " .. (#parts - i + 1) .. " more"
        end
        out = candidate
    end
    return out
end

local function plural(n, word)
    return n .. " " .. word .. (n == 1 and "" or "s")
end

local function describe(character)
    local facts = character.class and ns.clean(character.class) or ""
    if character.level then
        facts = facts .. (facts == "" and "" or " ") .. character.level
    end
    if character.main then
        facts = facts .. (facts == "" and "" or ", ") .. "main"
    end
    local name = ns.clean(character.name)
    return facts == "" and name or name .. " (" .. facts .. ")"
end

local function member(m, budget)
    local name = ns.clean(m.name)
    local parts = {}
    for _, character in ipairs(m.characters) do
        parts[#parts + 1] = describe(character)
    end
    if #parts == 0 then
        return name
    end
    return name .. ": " .. fit(parts, ", ", math.min(100, budget - #name - 2))
end

local function who(members, query)
    local matches = ns.matchMembers(members, query)
    if #matches == 0 then
        return 'nobody matches "' .. ns.clean(query) .. '".'
    end
    local segments = {}
    for _, m in ipairs(matches) do
        segments[#segments + 1] = member(m, LINE_BUDGET - #PREFIX)
    end
    return fit(segments, "; ", LINE_BUDGET - #PREFIX)
end

local function names(members)
    local parts = {}
    for _, m in ipairs(members) do
        parts[#parts + 1] = ns.clean(m.name)
    end
    return fit(parts, ", ", LINE_BUDGET - #PREFIX) .. "."
end

local REASONS = {
    missing = "no member list yet. The companion writes it once your PC is linked.",
    empty = "no member list yet. The companion writes it once your PC is linked.",
    version = "the member list is in a format this addon does not know.",
    unreadable = "the member list could not be read.",
}

-- The single line to print. `data` is the raw ForeverFriendsData global, `own` is { count = characters
-- recorded, key = this session's character or nil }. No time of any kind is ever in a line.
function ns.respond(message, data, own)
    local command, argument = (message or ""):match("^%s*(%S*)%s*(.-)%s*$")
    command = command:lower()
    if command == "status" then
        local line = plural(own.count, "character") .. " recorded"
        line = own.key and line .. ", this one is " .. ns.clean(own.key) .. "." or line .. "."
        return PREFIX .. line
    end
    if command ~= "" and command ~= "members" and not (command == "who" and argument ~= "") then
        return PREFIX .. USAGE
    end
    local members, reason = ns.readRoster(data)
    if not members then
        return PREFIX .. REASONS[reason]
    end
    local line
    if command == "members" then
        line = names(members)
    elseif command == "who" then
        line = who(members, argument)
    else
        local characters = ns.countCharacters(members)
        line = plural(#members, "member") .. ", " .. plural(characters, "character") .. ". "
            .. "/ff members, /ff who <name>, /ff status."
    end
    return PREFIX .. line
end
