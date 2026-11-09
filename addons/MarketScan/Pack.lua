-- Pure logic: no client globals, so busted loads this file bare. Core.lua feeds it rows.
local _, ns = ...

-- Weighted lower median of `values`. `weights` is optional (every value weighs 1). For an even
-- total weight this is the lower of the two middle values, not their average: it stays an
-- integer copper price that really occurred in the auction house. Returns nil for no values.
function ns.median(values, weights)
    local count = #values
    if count == 0 then
        return nil
    end
    local order = {}
    for i = 1, count do
        order[i] = i
    end
    table.sort(order, function(a, b)
        if values[a] ~= values[b] then
            return values[a] < values[b]
        end
        return a < b
    end)
    local total = 0
    for i = 1, count do
        total = total + (weights and weights[i] or 1)
    end
    local running = 0
    for _, i in ipairs(order) do
        running = running + (weights and weights[i] or 1)
        if running * 2 >= total then
            return values[i]
        end
    end
end

-- Computes one row's usable fields. Any error here (a secret value, a nil, a string) means the
-- row is unusable; the caller counts it as skipped. Returns nil for a row with no buyout.
local function readRow(row)
    local itemID, count, buyout = row.itemID, row.count, row.buyoutPrice
    if not buyout or buyout <= 0 then
        return nil
    end
    if count < 1 then
        error("count below 1")
    end
    local unit = math.floor(buyout / count + 0.5)
    local key = itemID + 0 -- arithmetic makes a secret itemID fail here, not as a table key later
    return key, unit, count
end

-- Groups rows per item. A row without a buyout is ignored silently (not skipped: it is a
-- normal bid-only auction). A row that errors is counted in the second return value.
-- Result: { [itemID] = { min = copper, med = copper, qty = units, n = auctions } }
function ns.aggregate(rows)
    local buckets, skipped = {}, 0
    for _, row in ipairs(rows) do
        local ok, itemID, unit, count = pcall(readRow, row)
        if not ok then
            skipped = skipped + 1
        elseif itemID then
            local bucket = buckets[itemID]
            if not bucket then
                bucket = { units = {}, counts = {} }
                buckets[itemID] = bucket
            end
            bucket.units[#bucket.units + 1] = unit
            bucket.counts[#bucket.counts + 1] = count
        end
    end
    local agg = {}
    for itemID, bucket in pairs(buckets) do
        local lowest, qty = bucket.units[1], 0
        for i, unit in ipairs(bucket.units) do
            if unit < lowest then
                lowest = unit
            end
            qty = qty + bucket.counts[i]
        end
        agg[itemID] = {
            min = lowest,
            med = ns.median(bucket.units, bucket.counts),
            qty = qty,
            n = #bucket.units,
        }
    end
    return agg, skipped
end

-- "%.0f" not "%d": Lua 5.1 %d casts to C long, possibly 32 bits in the Windows client.
-- "min:med:qty:n" per item: compact, plain strings the SavedVariables parser reads untouched.
function ns.pack(agg)
    local packed = {}
    for itemID, entry in pairs(agg) do
        packed[itemID] = string.format("%.0f:%.0f:%.0f:%.0f", entry.min, entry.med, entry.qty, entry.n)
    end
    return packed
end

-- Inverse of ns.pack for one string; nil when the string is not in that shape.
function ns.unpack(str)
    local lowest, med, qty, n = string.match(str, "^(%d+):(%d+):(%d+):(%d+)$")
    if not lowest then
        return nil
    end
    return { min = tonumber(lowest), med = tonumber(med), qty = tonumber(qty), n = tonumber(n) }
end
