-- Our own small simulated WoW client: only what the addons here need. Loaded by busted as a
-- helper; specs get it with require("wow"). wow.reset() gives every test a fresh client, so no
-- state leaks between tests. Paths are relative to the repository root, where busted runs.
local wow = {}
package.loaded["wow"] = wow

local AH_FIELDS = 18 -- GetReplicateItemInfo returns 18 values

-- A secret value: the real client raises when addon code does arithmetic, comparison or indexing
-- on one, so the stub raises on every such operation.
function wow.secret()
    local function boom()
        error("attempt to use a secret value", 2)
    end
    return setmetatable({}, {
        __add = boom, __sub = boom, __mul = boom, __div = boom,
        __lt = boom, __le = boom, __eq = boom, __index = boom,
    })
end

-- One auction as GetReplicateItemInfo returns it. Positions: 1 name, 2 texture, 3 count,
-- 4 qualityID, 5 usable, 6 level, 7 levelType, 8 minBid, 9 minIncrement, 10 buyoutPrice,
-- 11 bidAmount, 12 highBidder, 13 bidderFullName, 14 owner, 15 ownerFullName, 16 saleStatus,
-- 17 itemID, 18 hasAllInfo.
function wow.auction(itemID, count, buyoutPrice, minBid)
    return { "Item" .. itemID, 134400, count, 1, true, 0, nil, minBid or 1, 1, buyoutPrice, 0,
        nil, nil, nil, nil, 0, itemID, true }
end

local function newFrame(kind, name, parent)
    local frame = { kind = kind, name = name, parent = parent, events = {}, scripts = {}, shown = true, text = "" }
    function frame:RegisterEvent(event) self.events[event] = true end
    function frame:UnregisterEvent(event) self.events[event] = nil end
    function frame:SetScript(hook, fn) self.scripts[hook] = fn end
    function frame:GetScript(hook) return self.scripts[hook] end
    function frame:SetSize(w, h) self.size = { w, h } end
    function frame:SetPoint(...) self.point = { ... } end
    function frame:SetText(text) self.text = text end
    function frame:GetText() return self.text end
    function frame:Show() self.shown = true end
    function frame:Hide() self.shown = false end
    function frame:IsShown() return self.shown end
    function frame:Click()
        local onClick = self.scripts.OnClick
        if onClick then onClick(self) end
    end
    return frame
end

local function newAuctionHouse()
    local api = {
        ReplicateItems = function() wow.replicateCalls = wow.replicateCalls + 1 end,
        GetNumReplicateItems = function() return #wow.auctions end,
        GetReplicateItemInfo = function(index)
            local row = wow.auctions[index + 1] -- the client indexes from 0
            if row then return unpack(row, 1, AH_FIELDS) end
        end,
        GetReplicateItemTimeLeft = function() return 3 end,
        IsThrottledMessageSystemReady = function() return wow.throttleReady end,
    }
    -- Any other name is a call the addon has no business making (D5): fail loudly.
    return setmetatable(api, {
        __index = function(_, name) error("forbidden C_AuctionHouse." .. tostring(name), 2) end,
    })
end

function wow.reset()
    wow.frames, wow.timers, wow.printed = {}, {}, {}
    wow.auctions, wow.replicateCalls, wow.throttleReady = {}, 0, true
    wow.time, wow.realm = 1700000000, "TestRealm"

    _G.CreateFrame = function(kind, name, parent)
        local frame = newFrame(kind, name, parent)
        wow.frames[#wow.frames + 1] = frame
        return frame
    end
    _G.C_Timer = { After = function(_, fn) wow.timers[#wow.timers + 1] = fn end }
    _G.C_AuctionHouse = newAuctionHouse()
    _G.GetServerTime = function() return wow.time end
    _G.GetRealmName = function() return wow.realm end
    _G.date = function(format, t) return os.date(format, t or wow.time) end
    _G.print = function(...)
        local parts = {}
        for i = 1, select("#", ...) do parts[i] = tostring((select(i, ...))) end
        wow.printed[#wow.printed + 1] = table.concat(parts, " ")
    end
    _G.SlashCmdList = {}
    _G.MarketScanDB, _G.AuctionHouseFrame = nil, nil
end

-- Sends an event to every frame that registered for it, in creation order.
function wow.fire(event, ...)
    for _, frame in ipairs(wow.frames) do
        local onEvent = frame.scripts.OnEvent
        if frame.events[event] and onEvent then onEvent(frame, event, ...) end
    end
end

-- Runs the callbacks queued so far, once. Callbacks queued meanwhile wait for the next tick,
-- like a real frame boundary. Returns how many ran.
function wow.tick()
    local queued = wow.timers
    wow.timers = {}
    for _, fn in ipairs(queued) do fn() end
    return #queued
end

function wow.slash(key, message) _G.SlashCmdList[key](message) end

-- Loads an addon the way the client does: the TOC's Lua files in order, each called with
-- (name, ns). Returns the shared ns table.
function wow.loadAddon(name)
    local dir = "addons/" .. name .. "/"
    local toc = assert(io.open(dir .. name .. ".toc", "r"))
    local ns = {}
    for line in toc:lines() do
        local file = line:match("^%s*(%S+%.lua)%s*$")
        if file then assert(loadfile(dir .. file))(name, ns) end
    end
    toc:close()
    return ns
end

wow.reset()
return wow
