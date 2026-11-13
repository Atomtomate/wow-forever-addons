-- Client-facing part, kept thin. Reads the auction house only when the player presses the
-- button (D5): the sole C_AuctionHouse calls are ReplicateItems, GetNumReplicateItems,
-- GetReplicateItemInfo and IsThrottledMessageSystemReady. Nothing here bids, buys, posts or cancels.
local _, ns = ...

local MAX_SCANS = 20
local MAX_LOG = 50
local CHUNK = 500 -- rows read per frame, so a large scan does not hitch the client

local frame = CreateFrame("Frame")
local ahOpen = false
local requested = false -- true between our button press and the scan finishing
local current = nil -- the scan being read; a stale timer compares against it and stops

-- Ring buffer inside SavedVariables, so the log leaves the machine with the data.
function ns.log(msg)
    local log = MarketScanDB and MarketScanDB.log
    if not log then
        return
    end
    log[#log + 1] = date("%Y-%m-%d %H:%M:%S") .. " " .. msg
    while #log > MAX_LOG do
        table.remove(log, 1)
    end
end

local function say(msg)
    print(msg)
end

local function commas(n)
    local s = tostring(n)
    local out = s:reverse():gsub("(%d%d%d)", "%1,"):reverse()
    return (out:gsub("^,", ""))
end

local function countItems(items)
    local n = 0
    for _ in pairs(items) do
        n = n + 1
    end
    return n
end

-- GetReplicateItemInfo(index) returns 18 values (AuctionHouseDocumentation.lua): 3 = count,
-- 8 = minBid, 10 = buyoutPrice, 17 = itemID. The call runs under pcall: a failing read is a
-- skipped row, never a broken scan. Returns ok, row.
local function readInfo(ok, ...)
    if not ok then
        return false
    end
    local count, minBid, buyoutPrice, itemID = select(3, ...), select(8, ...), select(10, ...), select(17, ...)
    return true, { itemID = itemID, count = count, buyoutPrice = buyoutPrice, minBid = minBid }
end

local function finish(scan)
    local agg, skippedRows = ns.aggregate(scan.rows)
    local items = ns.pack(agg)
    local skipped = scan.failed + skippedRows
    local scans = MarketScanDB.scans
    scans[#scans + 1] = {
        t = GetServerTime(),
        realm = GetRealmName(),
        rows = scan.total,
        skipped = skipped,
        items = items,
    }
    while #scans > MAX_SCANS do
        table.remove(scans, 1)
    end
    local summary = string.format(
        "%s auctions, %s items, %d skipped", commas(scan.total), commas(countItems(items)), skipped
    )
    ns.log("scan: " .. summary)
    say("MarketScan: " .. summary .. ". Saved; /reload or log out writes the file.")
    current, requested = nil, false
end

local function readChunk(scan)
    if current ~= scan then
        return
    end
    local last = math.min(scan.next + CHUNK - 1, scan.total - 1)
    for i = scan.next, last do
        local ok, row = readInfo(pcall(C_AuctionHouse.GetReplicateItemInfo, i))
        if ok then
            scan.rows[#scan.rows + 1] = row
        else
            scan.failed = scan.failed + 1
        end
    end
    scan.next = last + 1
    if scan.next < scan.total then
        C_Timer.After(0, function()
            readChunk(scan)
        end)
    else
        finish(scan)
    end
end

local function onReplicateUpdate()
    if not requested or current then
        return -- not our scan, or one already being read
    end
    local ok, total = pcall(function()
        return C_AuctionHouse.GetNumReplicateItems() + 0 -- a secret value fails here, inside the pcall
    end)
    if not ok or type(total) ~= "number" then
        requested = false
        ns.log("scan failed: GetNumReplicateItems unusable")
        say("MarketScan: could not read the auction list.")
        return
    end
    if total < 1 then
        requested = false
        ns.log("scan returned no rows")
        say("MarketScan: the auction house returned no rows.")
        return
    end
    current = { total = total, next = 0, rows = {}, failed = 0 }
    readChunk(current)
end

local function requestScan()
    if not ahOpen then
        say("MarketScan: open the auction house first.")
        return
    end
    local ready = C_AuctionHouse.IsThrottledMessageSystemReady
    if current or (ready and not ready()) then
        say("Auction house is busy, try again in a moment.")
        return
    end
    requested = true
    ns.log("scan requested")
    local ok, err = pcall(C_AuctionHouse.ReplicateItems)
    if not ok then
        requested = false
        ns.log("ReplicateItems failed: " .. tostring(err))
        say("Auction house is busy, try again in a moment.")
    end
end

local button
local function ensureButton()
    if button then
        return
    end
    if not AuctionHouseFrame then
        ns.log("AuctionHouseFrame missing; no button (use /marketscan help)")
        return
    end
    button = CreateFrame("Button", nil, AuctionHouseFrame, "UIPanelButtonTemplate")
    button:SetSize(100, 22)
    button:SetPoint("TOPRIGHT", AuctionHouseFrame, "TOPRIGHT", -30, -3)
    button:SetText("Scan prices")
    button:SetScript("OnClick", requestScan)
end

local function onAddonLoaded(name)
    if name ~= "MarketScan" then
        return
    end
    MarketScanDB = MarketScanDB or { version = 1, scans = {}, log = {} }
    MarketScanDB.scans = MarketScanDB.scans or {}
    MarketScanDB.log = MarketScanDB.log or {}
end

local function onAuctionHouseClosed()
    ahOpen = false
    requested = false -- a pending request must not outlive the window it was made in (D5)
    if current then
        current = nil
        ns.log("scan aborted: auction house closed")
    end
end

local handlers = {
    ADDON_LOADED = onAddonLoaded,
    AUCTION_HOUSE_SHOW = function()
        ahOpen = true
        ensureButton()
    end,
    AUCTION_HOUSE_CLOSED = onAuctionHouseClosed,
    REPLICATE_ITEM_LIST_UPDATE = onReplicateUpdate,
}

frame:SetScript("OnEvent", function(_, event, ...)
    handlers[event](...)
end)
for event in pairs(handlers) do
    frame:RegisterEvent(event)
end

local function status()
    local scans = MarketScanDB.scans
    local last = scans[#scans]
    if not last then
        say("MarketScan: no scans yet.")
        return
    end
    say(string.format(
        "MarketScan: last scan %s on %s, %s auctions, %s items, %d skipped (%d stored).",
        date("%Y-%m-%d %H:%M:%S", last.t), last.realm, commas(last.rows), commas(countItems(last.items)),
        last.skipped, #scans
    ))
end

local function showLog()
    local log = MarketScanDB.log
    if #log == 0 then
        say("MarketScan: log is empty.")
        return
    end
    for _, line in ipairs(log) do
        say(line)
    end
end

local function clear(arg)
    if arg ~= "confirm" then
        say(string.format(
            "MarketScan: this deletes %d saved scans; type /marketscan clear confirm.", #MarketScanDB.scans
        ))
        return
    end
    MarketScanDB.scans = {}
    ns.log("scans cleared")
    say("MarketScan: scans cleared.")
end

SLASH_MARKETSCAN1 = "/marketscan"
SLASH_MARKETSCAN2 = "/ms"
SlashCmdList["MARKETSCAN"] = function(msg)
    local command, arg = (msg or ""):match("^%s*(%S*)%s*(%S*)")
    command = command:lower()
    if command == "status" then
        status()
    elseif command == "log" then
        showLog()
    elseif command == "clear" then
        clear(arg:lower())
    else
        say("MarketScan: /marketscan status | log | clear. Scan with the button at the auction house.")
    end
end
