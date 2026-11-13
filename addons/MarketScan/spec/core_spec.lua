local wow = require("wow")

-- Writes a table the way the client writes SavedVariables: plain Lua, sorted for stability.
local function serialize(value)
    if type(value) == "table" then
        local keys = {}
        for k in pairs(value) do keys[#keys + 1] = k end
        table.sort(keys, function(a, b) return tostring(a) < tostring(b) end)
        local parts = {}
        for _, k in ipairs(keys) do
            parts[#parts + 1] = "[" .. serialize(k) .. "]=" .. serialize(value[k])
        end
        return "{" .. table.concat(parts, ",") .. "}"
    end
    return type(value) == "string" and string.format("%q", value) or tostring(value)
end

describe("MarketScan Core (simulated client)", function()
    local ns, button

    local function openAuctionHouse()
        _G.AuctionHouseFrame = CreateFrame("Frame")
        wow.fire("AUCTION_HOUSE_SHOW")
        button = wow.frames[#wow.frames]
    end

    local function fillAuctions(n, itemCount)
        for i = 1, n do
            wow.auctions[i] = wow.auction(1000 + (i % itemCount), 1 + (i % 3), 100 * (1 + i % 7))
        end
    end

    local function drain()
        local ticks = 0
        while wow.tick() > 0 do ticks = ticks + 1 end
        return ticks
    end

    local function scanOnce()
        button:Click()
        wow.fire("REPLICATE_ITEM_LIST_UPDATE")
        drain()
    end

    before_each(function()
        wow.reset()
        ns = wow.loadAddon("MarketScan")
        wow.fire("ADDON_LOADED", "MarketScan")
    end)

    it("initialises the database on ADDON_LOADED, ignoring other addons", function()
        assert.are.same({ version = 1, scans = {}, log = {} }, MarketScanDB)
        local db = MarketScanDB
        wow.fire("ADDON_LOADED", "SomethingElse")
        assert.are.equal(db, MarketScanDB)
    end)

    it("keeps an existing database", function()
        wow.reset()
        _G.MarketScanDB = { version = 1, scans = { { t = 1 } }, log = {} }
        wow.loadAddon("MarketScan")
        wow.fire("ADDON_LOADED", "MarketScan")
        assert.are.equal(1, #MarketScanDB.scans)
    end)

    it("creates the button once", function()
        openAuctionHouse()
        local count = #wow.frames
        wow.fire("AUCTION_HOUSE_SHOW")
        assert.are.equal(count, #wow.frames)
        assert.are.equal("Scan prices", button:GetText())
        assert.are.equal(AuctionHouseFrame, button.parent)
    end)

    it("logs and skips the button when AuctionHouseFrame is missing", function()
        local count = #wow.frames
        wow.fire("AUCTION_HOUSE_SHOW")
        assert.are.equal(count, #wow.frames)
        assert.are.equal(1, #MarketScanDB.log)
    end)

    it("calls ReplicateItems once per click and never by itself", function()
        openAuctionHouse()
        wow.fire("AUCTION_HOUSE_SHOW")
        wow.fire("AUCTION_HOUSE_CLOSED")
        wow.fire("AUCTION_HOUSE_SHOW")
        wow.fire("REPLICATE_ITEM_LIST_UPDATE")
        drain()
        assert.are.equal(0, wow.replicateCalls)
        button:Click()
        assert.are.equal(1, wow.replicateCalls)
    end)

    it("forgets a pending request when the auction house closes", function()
        openAuctionHouse()
        fillAuctions(10, 3)
        button:Click()
        wow.fire("AUCTION_HOUSE_CLOSED")
        wow.fire("AUCTION_HOUSE_SHOW")
        wow.fire("REPLICATE_ITEM_LIST_UPDATE")
        drain()
        assert.are.equal(0, #MarketScanDB.scans)
    end)

    it("recovers when ReplicateItems raises", function()
        openAuctionHouse()
        C_AuctionHouse.ReplicateItems = function() error("boom") end
        button:Click()
        assert.are.same({ "Auction house is busy, try again in a moment." }, wow.printed)
        fillAuctions(5, 2)
        wow.fire("REPLICATE_ITEM_LIST_UPDATE")
        drain()
        assert.are.equal(0, #MarketScanDB.scans)
        assert.is_truthy(MarketScanDB.log[#MarketScanDB.log]:find("ReplicateItems failed", 1, true))
    end)

    it("does not scan when the throttle says busy", function()
        openAuctionHouse()
        wow.throttleReady = false
        button:Click()
        assert.are.equal(0, wow.replicateCalls)
        assert.are.same({ "Auction house is busy, try again in a moment." }, wow.printed)
    end)

    it("does not scan once the auction house is closed", function()
        openAuctionHouse()
        wow.fire("AUCTION_HOUSE_CLOSED")
        button:Click()
        assert.are.equal(0, wow.replicateCalls)
        assert.are.equal(1, #wow.printed)
    end)

    it("ignores an update nobody asked for", function()
        openAuctionHouse()
        fillAuctions(10, 3)
        wow.fire("REPLICATE_ITEM_LIST_UPDATE")
        drain()
        assert.are.equal(0, #MarketScanDB.scans)
    end)

    it("reads 1,200 rows across ticks and stores one scan", function()
        openAuctionHouse()
        fillAuctions(1200, 40)
        wow.auctions[5] = wow.auction(1, 1, 0) -- no buyout
        wow.auctions[6][3] = wow.secret() -- unusable count
        button:Click()
        wow.fire("REPLICATE_ITEM_LIST_UPDATE")
        assert.are.equal(1, #wow.timers) -- first chunk of 500 done, the rest waits for a frame
        assert.are.equal(0, #MarketScanDB.scans)
        assert.are.equal(2, drain())
        assert.are.equal(1, #MarketScanDB.scans)
        local scan = MarketScanDB.scans[1]
        assert.are.equal(wow.time, scan.t)
        assert.are.equal("TestRealm", scan.realm)
        assert.are.equal(1200, scan.rows)
        assert.are.equal(1, scan.skipped)
        local items = 0
        for itemID, packed in pairs(scan.items) do
            items = items + 1
            assert.is_true(itemID >= 1000 and ns.unpack(packed) ~= nil)
        end
        assert.are.equal(40, items)
        assert.are.equal(1, #wow.printed)
        assert.are.equal("MarketScan: 1,200 auctions, 40 items, 1 skipped. Saved; /reload or log out writes the file.",
            wow.printed[1])
        assert.are.equal(2, #MarketScanDB.log) -- requested, scan summary
    end)

    it("counts a failing read as skipped", function()
        openAuctionHouse()
        fillAuctions(3, 3)
        local real = C_AuctionHouse.GetReplicateItemInfo
        C_AuctionHouse.GetReplicateItemInfo = function(i)
            if i == 1 then error("secret") end
            return real(i)
        end
        scanOnce()
        assert.are.equal(1, MarketScanDB.scans[1].skipped)
    end)

    it("aborts a scan in progress when the auction house closes", function()
        openAuctionHouse()
        fillAuctions(1200, 40)
        button:Click()
        wow.fire("REPLICATE_ITEM_LIST_UPDATE")
        wow.fire("AUCTION_HOUSE_CLOSED")
        drain()
        assert.are.equal(0, #MarketScanDB.scans)
    end)

    it("trims to 20 scans and 50 log lines", function()
        openAuctionHouse()
        fillAuctions(5, 2)
        for i = 1, 25 do
            wow.time = wow.time + i
            scanOnce()
        end
        assert.are.equal(20, #MarketScanDB.scans)
        assert.are.equal(50, #MarketScanDB.log)
        assert.is_true(MarketScanDB.scans[1].t < MarketScanDB.scans[20].t)
    end)

    it("survives a simulated /reload in a form the parser reads", function()
        openAuctionHouse()
        fillAuctions(30, 5)
        scanOnce()
        local text = "MarketScanDB = " .. serialize(MarketScanDB)
        local env = {}
        assert(loadstring(text))
        setfenv(loadstring(text), env)()
        assert.are.equal(serialize(MarketScanDB), serialize(env.MarketScanDB))
        assert.are.equal(serialize(MarketScanDB.scans[1]), serialize(env.MarketScanDB.scans[1]))
    end)

    describe("/marketscan", function()
        it("reports no scans, then the last one", function()
            wow.slash("MARKETSCAN", "status")
            assert.are.equal("MarketScan: no scans yet.", wow.printed[1])
            openAuctionHouse()
            fillAuctions(12, 4)
            scanOnce()
            wow.printed = {}
            wow.slash("MARKETSCAN", "status")
            assert.is_truthy(wow.printed[1]:find("12 auctions, 4 items, 0 skipped (1 stored)", 1, true))
        end)

        it("prints the log lines", function()
            ns.log("hello")
            wow.slash("MARKETSCAN", "log")
            assert.is_truthy(wow.printed[1]:find("hello", 1, true))
        end)

        it("clears only after a second confirming command", function()
            openAuctionHouse()
            fillAuctions(5, 2)
            scanOnce()
            wow.slash("MARKETSCAN", "clear")
            assert.are.equal(1, #MarketScanDB.scans)
            wow.slash("MARKETSCAN", "clear confirm")
            assert.are.equal(0, #MarketScanDB.scans)
        end)

        it("registers both slash words and explains itself on unknown input", function()
            assert.are.equal("/marketscan", SLASH_MARKETSCAN1)
            assert.are.equal("/ms", SLASH_MARKETSCAN2)
            wow.slash("MARKETSCAN", "")
            assert.are.equal(1, #wow.printed)
        end)
    end)

    it("reads only allowed C_AuctionHouse names, never an acting one (D5)", function()
        local seen = {}
        local real = C_AuctionHouse
        _G.C_AuctionHouse = setmetatable({}, {
            __index = function(_, name)
                seen[name] = true
                return real[name] -- the stub itself raises for names it does not have
            end,
        })
        openAuctionHouse()
        fillAuctions(10, 3)
        scanOnce()
        local allowed = {
            ReplicateItems = true, GetNumReplicateItems = true, GetReplicateItemInfo = true,
            GetReplicateItemTimeLeft = true, IsThrottledMessageSystemReady = true,
        }
        for name in pairs(seen) do
            assert.is_true(allowed[name], "unexpected C_AuctionHouse." .. name)
        end
        assert.is_true(seen.ReplicateItems and seen.GetReplicateItemInfo)
        assert.has_error(function() return real.PlaceBid end)
    end)
end)
