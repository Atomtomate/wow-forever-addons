local wow = require("wow")

describe("MarketScan Pack (pure)", function()
    local ns

    before_each(function()
        ns = {}
        assert(loadfile("addons/MarketScan/Pack.lua"))("MarketScan", ns)
    end)

    local function row(itemID, count, buyoutPrice)
        return { itemID = itemID, count = count, buyoutPrice = buyoutPrice, minBid = 1 }
    end

    describe("median", function()
        it("takes the middle of an odd count", function()
            assert.are.equal(5, ns.median({ 9, 1, 5 }))
        end)
        it("takes the lower middle of an even count", function()
            assert.are.equal(2, ns.median({ 4, 1, 3, 2 }))
        end)
        it("weights by quantity", function()
            -- 1 unit at 10, 9 units at 50: most units cost 50
            assert.are.equal(50, ns.median({ 10, 50 }, { 1, 9 }))
        end)
        it("is nil for nothing", function()
            assert.is_nil(ns.median({}))
        end)
    end)

    describe("aggregate", function()
        it("groups duplicates, unit prices and quantities", function()
            local agg, skipped = ns.aggregate({
                row(100, 1, 500),
                row(100, 5, 2000), -- 400 each
                row(100, 2, 1001), -- 500.5 -> 501 each
                row(200, 1, 7),
            })
            assert.are.equal(0, skipped)
            assert.are.same({ min = 400, med = 400, qty = 8, n = 3 }, agg[100])
            assert.are.same({ min = 7, med = 7, qty = 1, n = 1 }, agg[200])
        end)

        it("ignores auctions without a buyout without counting them skipped", function()
            local agg, skipped = ns.aggregate({ row(1, 1, 0), row(1, 1, nil), row(2, 1, 10) })
            assert.are.equal(0, skipped)
            assert.is_nil(agg[1])
            assert.is_not_nil(agg[2])
        end)

        it("skips a row holding a secret value and keeps the rest", function()
            local agg, skipped = ns.aggregate({
                row(1, 1, 10),
                row(1, wow.secret(), 10),
                row(wow.secret(), 1, 10),
                row(1, 1, wow.secret()),
                row(1, 1, 30),
            })
            assert.are.equal(3, skipped)
            assert.are.same({ min = 10, med = 10, qty = 2, n = 2 }, agg[1])
        end)

        it("skips malformed rows", function()
            local _, skipped = ns.aggregate({ row(1, 0, 10), row(nil, 1, 10), row(1, "x", 10) })
            assert.are.equal(3, skipped)
        end)

        it("handles empty input", function()
            local agg, skipped = ns.aggregate({})
            assert.are.same({}, agg)
            assert.are.equal(0, skipped)
        end)
    end)

    describe("pack and unpack", function()
        it("round-trips exactly", function()
            local agg = ns.aggregate({ row(100, 1, 500), row(100, 3, 3000), row(7, 20, 123456789) })
            local packed = ns.pack(agg)
            assert.are.equal("500:1000:4:2", packed[100])
            for itemID, entry in pairs(agg) do
                assert.are.same(entry, ns.unpack(packed[itemID]))
            end
        end)
        it("round-trips copper above 2^31 exactly", function()
            local agg = ns.aggregate({ row(5, 1, 3000000000), row(5, 2, 6000000001) })
            local packed = ns.pack(agg)
            assert.are.equal("3000000000:3000000001:3:2", packed[5])
            assert.are.same(agg[5], ns.unpack(packed[5]))
        end)
        it("packs nothing to nothing", function()
            assert.are.same({}, ns.pack({}))
        end)
        it("rejects a malformed string", function()
            assert.is_nil(ns.unpack("1:2:3"))
            assert.is_nil(ns.unpack("a:b:c:d"))
        end)
    end)
end)
