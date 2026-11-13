local wow = require("wow")

describe("ForeverFriends Snapshot (pure)", function()
    local ns

    before_each(function()
        wow.reset()
        ns = {}
        assert(loadfile("addons/ForeverFriends/Snapshot.lua"))("ForeverFriends", ns)
    end)

    describe("usable", function()
        it("accepts a non-empty string, a level and a count", function()
            assert.are.equal("Orc", ns.usable("Orc", "string"))
            assert.are.equal(23, ns.usable(23, "level"))
            assert.are.equal(0, ns.usable(0, "count"))
        end)

        it("rejects nil, an empty string and the wrong type", function()
            assert.is_nil(ns.usable(nil, "string"))
            assert.is_nil(ns.usable("", "string"))
            assert.is_nil(ns.usable(5, "string"))
            assert.is_nil(ns.usable("5", "level"))
            assert.is_nil(ns.usable({}, "string"))
        end)

        it("rejects a level below 1, a fraction, NaN and infinity", function()
            assert.is_nil(ns.usable(0, "level"))
            assert.is_nil(ns.usable(-1, "count"))
            assert.is_nil(ns.usable(1.5, "level"))
            assert.is_nil(ns.usable(0 / 0, "level"))
            assert.is_nil(ns.usable(math.huge, "level"))
        end)

        it("rejects a secret value for every kind", function()
            assert.is_nil(ns.usable(wow.secret(), "string"))
            assert.is_nil(ns.usable(wow.secret(), "level"))
            assert.is_nil(ns.usable(wow.secret(), "count"))
        end)

        it("rejects a secret that allows concatenation but not comparison", function()
            local later = setmetatable({}, {
                __concat = function() return setmetatable({}, getmetatable(wow.secret())) end,
                __eq = function() error("compare") end,
            })
            assert.is_nil(ns.usable(later, "string"))
        end)
    end)

    describe("buildEntry", function()
        local function read(extra)
            local r = { name = "Thrallina", realm = "Everlook", class = "SHAMAN", race = "Orc", level = 23 }
            for k, v in pairs(extra or {}) do r[k] = v end
            return r
        end

        it("writes the contract's shape", function()
            local skills = { { id = 171, name = "Alchimie", rank = 145, max = 150 } }
            local key, entry = ns.buildEntry(nil, read({ skills = skills }), 1763000000)
            assert.are.equal("Thrallina-Everlook", key)
            assert.are.same({ name = "Thrallina", realm = "Everlook", class = "SHAMAN", race = "Orc", level = 23,
                skills = skills, seen = 1763000000 }, entry)
        end)

        it("keeps the old fields when the new read is nil or secret", function()
            local old = { name = "Thrallina", realm = "Everlook", class = "SHAMAN", race = "Orc", level = 22,
                skills = { { id = 1, name = "", rank = 1, max = 1 } }, seen = 5 }
            local _, entry = ns.buildEntry(old, read({ race = wow.secret(), level = wow.secret() }), 9)
            assert.are.equal("Orc", entry.race)
            assert.are.equal(22, entry.level)
            assert.are.equal(1, #entry.skills)
            assert.are.equal(9, entry.seen)
        end)

        it("does not touch the old entry", function()
            local old = { name = "Thrallina", realm = "Everlook", class = "SHAMAN", level = 22 }
            ns.buildEntry(old, read(), 9)
            assert.are.equal(22, old.level)
        end)

        it("creates nothing without a usable name or realm", function()
            assert.is_nil(ns.buildEntry(nil, read({ name = wow.secret() }), 9))
            assert.is_nil(ns.buildEntry(nil, read({ realm = wow.secret() }), 9))
        end)

        it("creates nothing without a class, but updates an entry that has one", function()
            assert.is_nil(ns.buildEntry(nil, read({ class = wow.secret() }), 9))
            local old = { name = "Thrallina", realm = "Everlook", class = "SHAMAN", seen = 5 }
            local _, entry = ns.buildEntry(old, read({ class = wow.secret(), level = 24 }), 9)
            assert.are.equal("SHAMAN", entry.class)
            assert.are.equal(24, entry.level)
            assert.are.equal(5, entry.seen) -- the read did not name the class, so it is not a refresh
        end)
    end)

    describe("collectSkills", function()
        local function lines(list)
            return function(i) return list[i] end
        end

        it("drops headers and keeps the client's order", function()
            local list = { wow.skill(0, 0, 0, true), wow.skill(171, 145, 150), wow.skill(164, 10, 75) }
            local skills, truncated = ns.collectSkills(#list, lines(list))
            assert.are.same({ { id = 171, name = "Skill", rank = 145, max = 150 },
                { id = 164, name = "Skill", rank = 10, max = 75 } }, skills)
            assert.is_false(truncated)
        end)

        it("keeps the first 40 and says it cut", function()
            local list = {}
            for i = 1, 60 do list[i] = wow.skill(i, 1, 300) end
            local skills, truncated = ns.collectSkills(60, lines(list))
            assert.are.equal(40, #skills)
            assert.are.equal(40, skills[40].id)
            assert.is_true(truncated)
        end)

        it("drops a line with a secret id or rank, never guessing", function()
            local list = { wow.skill(171, wow.secret(), 150), wow.skill(wow.secret(), 1, 2), wow.skill(164, 1, 75) }
            local skills = ns.collectSkills(3, lines(list))
            assert.are.equal(1, #skills)
            assert.are.equal(164, skills[1].id)
        end)

        it("stores an empty name when the name is secret", function()
            local list = { wow.skill(171, 1, 75) }
            list[1].name = wow.secret()
            assert.are.equal("", ns.collectSkills(1, lines(list))[1].name)
        end)

        it("returns nil when there is nothing to replace the old list with", function()
            assert.is_nil(ns.collectSkills(0, lines({})))
            assert.is_nil(ns.collectSkills(nil, lines({})))
            assert.is_nil(ns.collectSkills(wow.secret(), lines({})))
            assert.is_nil(ns.collectSkills(2, function() error("boom") end))
            assert.is_nil(ns.collectSkills(1, lines({ wow.skill(0, 0, 0, true) })))
            assert.is_nil(ns.collectSkills(1, lines({ wow.secret() })))
        end)
    end)

    describe("pushLog and normalise", function()
        it("keeps the log to 50 lines, oldest out", function()
            local log = {}
            for i = 1, 60 do ns.pushLog(log, "line " .. i) end
            assert.are.equal(50, #log)
            assert.are.equal("line 11", log[1])
        end)

        it("replaces nil, a non-table and another version by an empty table", function()
            local empty = { version = 1, characters = {}, log = {} }
            assert.are.same(empty, ns.normalise(nil))
            assert.are.same(empty, ns.normalise("x"))
            assert.are.same(empty, ns.normalise({ version = 2, characters = { a = {} } }))
            assert.are.same(empty, ns.normalise({ characters = {} }))
        end)

        it("repairs the parts of a version 1 table and keeps its characters", function()
            local kept = { name = "A", realm = "R", class = "MAGE" }
            local db = ns.normalise({ version = 1, characters = { ["A-R"] = kept, bad = 5, [3] = {} }, log = "x" })
            assert.are.equal(kept, db.characters["A-R"])
            assert.is_nil(db.characters.bad)
            assert.is_nil(db.characters[3])
            assert.are.same({}, db.log)
            db = ns.normalise({ version = 1, characters = "x", log = { "a", 5, "b" } })
            assert.are.same({}, db.characters)
            assert.are.same({ "a", "b" }, db.log)
        end)

        it("trims a long loaded-back log", function()
            local log = {}
            for i = 1, 200 do log[i] = "l" .. i end
            local db = ns.normalise({ version = 1, characters = {}, log = log })
            assert.are.equal(50, #db.log)
            assert.are.equal("l200", db.log[50])
        end)
    end)
end)
