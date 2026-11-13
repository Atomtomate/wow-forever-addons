local wow = require("wow")

describe("ForeverFriends Roster (pure)", function()
    local ns

    before_each(function()
        wow.reset()
        ns = {}
        assert(loadfile("addons/ForeverFriends/Roster.lua"))("ForeverFriends", ns)
    end)

    local function data(members)
        return { version = 1, written = 1763064000, members = members }
    end

    local sample = data({
        { name = "Anna", characters = {
            { name = "Thrallina", class = "Shaman", level = 23, main = true },
            { name = "Bankalt", class = "Mage" },
        } },
        { name = "Ben", characters = {} },
    })

    it("reads the contract's sample", function()
        local members = ns.readRoster(sample)
        assert.are.same({
            { name = "Anna", characters = {
                { name = "Thrallina", class = "Shaman", level = 23, main = true },
                { name = "Bankalt", class = "Mage", main = false },
            } },
            { name = "Ben", characters = {} },
        }, members)
        assert.are.equal(2, ns.countCharacters(members))
    end)

    it("says why there is no list", function()
        assert.are.same({ nil, "missing" }, { ns.readRoster(nil) })
        assert.are.same({ nil, "empty" }, { ns.readRoster(data({})) })
        assert.are.same({ nil, "version" }, { ns.readRoster({ version = 2, members = {} }) })
        assert.are.same({ nil, "version" }, { ns.readRoster({ members = {} }) })
        assert.are.same({ nil, "version" }, { ns.readRoster(5) })
        assert.are.same({ nil, "unreadable" }, { ns.readRoster({ version = 1, members = "x" }) })
    end)

    it("treats the companion's first file as no list yet", function()
        assert.are.same({ nil, "empty" }, { ns.readRoster({ version = 1, written = 0, members = {} }) })
    end)

    it("survives a secret value and a table that raises when indexed", function()
        assert.is_nil((ns.readRoster(wow.secret())))
        local hostile = setmetatable({ version = 1 }, { __index = function() error("boom") end })
        assert.is_nil((ns.readRoster(hostile)))
        local inner = data({ setmetatable({ name = "A" }, { __index = function() error("boom") end }) })
        assert.are.equal("A", ns.readRoster(inner)[1].name)
    end)

    it("skips what does not fit and keeps the rest", function()
        local members = ns.readRoster(data({
            "a string", { name = 5 }, { name = ("x"):rep(161) }, { name = "" },
            { name = ("é"):rep(80) },
            { name = "Keep", characters = {
                { name = 7 }, { name = ("y"):rep(65) }, "text", { name = "Fine", class = 5, level = "5", main = "yes" },
            } },
        }))
        assert.are.equal(2, #members)
        assert.are.same({ { name = "Fine", main = false } }, members[2].characters)
    end)

    it("accepts a 160-byte member name and a 64-byte character name", function()
        local members = ns.readRoster(data({ { name = ("m"):rep(160), characters = { { name = ("c"):rep(64) } } } }))
        assert.are.equal(160, #members[1].name)
        assert.are.equal(64, #members[1].characters[1].name)
    end)

    it("keeps only whole levels from 1 to 120 and main only when true", function()
        local function level(v)
            local members = ns.readRoster(data({ { name = "A", characters = { { name = "C", level = v } } } }))
            return members[1].characters[1].level
        end
        assert.are.equal(1, level(1))
        assert.are.equal(120, level(120))
        for _, bad in ipairs({ 0, 121, 1e9, 1.5, 0 / 0, math.huge, "5", true }) do
            assert.is_nil(level(bad))
        end
        local m = ns.readRoster(data({ { name = "A", characters = { { name = "C", main = true } } } }))
        assert.is_true(m[1].characters[1].main)
    end)

    it("stops at a hole and at the caps", function()
        local list = { { name = "A" }, nil, { name = "C" } }
        assert.are.equal(1, #ns.readRoster(data(list)))
        local many, chars = {}, {}
        for i = 1, 101 do many[i] = { name = "M" .. i } end
        for i = 1, 61 do chars[i] = { name = "C" .. i } end
        many[1].characters = chars
        local members = ns.readRoster(data(many))
        assert.are.equal(100, #members)
        assert.are.equal(60, #members[1].characters)
    end)

    it("reads 10000 members without looping on them", function()
        local many = {}
        for i = 1, 10000 do many[i] = {} end
        assert.are.same({ nil, "empty" }, { ns.readRoster(data(many)) })
    end)

    describe("matchMembers", function()
        local members
        before_each(function() members = ns.readRoster(sample) end)

        it("matches a member name or a character name by prefix, in any case", function()
            assert.are.equal("Anna", ns.matchMembers(members, "an")[1].name)
            assert.are.equal("Anna", ns.matchMembers(members, "THRALL")[1].name)
            assert.are.equal("Anna", ns.matchMembers(members, "bankalt")[1].name)
            assert.are.equal(1, #ns.matchMembers(members, "ben"))
        end)

        it("finds nothing for an empty query or a stranger", function()
            assert.are.equal(0, #ns.matchMembers(members, ""))
            assert.are.equal(0, #ns.matchMembers(members, "zed"))
        end)

        it("takes the query as plain text, never a pattern", function()
            for _, q in ipairs({ "%", "(", "[a-", ".-", ".", "%a" }) do
                assert.are.equal(0, #ns.matchMembers(members, q))
            end
        end)
    end)
end)
