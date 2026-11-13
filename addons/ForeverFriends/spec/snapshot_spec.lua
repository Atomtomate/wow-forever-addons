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

        -- On the client a secret keeps its underlying type(), and comparing it is what raises. Lua 5.1
        -- calls __eq only between two tables sharing the handler, so these secrets share one, and
        -- type() is made to lie. Without the probe each of these specs fails.
        local function withType(v, claimed, fn)
            local real = _G.type
            _G.type = function(x) return rawequal(x, v) and claimed or real(x) end
            local ok, err = pcall(fn)
            _G.type = real
            assert(ok, err)
        end

        it("rejects a secret that reports the type string", function()
            local secret = wow.secret()
            withType(secret, "string", function() assert.is_nil(ns.usable(secret, "string")) end)
        end)

        it("rejects a secret that reports the type number", function()
            local secret = wow.secret()
            withType(secret, "number", function()
                assert.is_nil(ns.usable(secret, "level"))
                assert.is_nil(ns.usable(secret, "count"))
            end)
        end)

        it("rejects a secret that allows concatenation and arithmetic but not comparison", function()
            local mt = {}
            mt.__concat = function() return setmetatable({}, mt) end
            mt.__add = function() return setmetatable({}, mt) end
            mt.__eq = function() error("compare") end
            local text, number = setmetatable({}, mt), setmetatable({}, mt)
            withType(text, "string", function() assert.is_nil(ns.usable(text, "string")) end)
            withType(number, "number", function() assert.is_nil(ns.usable(number, "level")) end)
        end)
    end)

    describe("buildEntry", function()
        local function read(extra)
            local r = { name = "Thrallina", realm = "Everlook", class = "SHAMAN", race = "Orc", level = 23 }
            for k, v in pairs(extra or {}) do r[k] = v end
            return r
        end

        it("writes the contract's shape under the key name-realm", function()
            local skills = { { id = 171, name = "Alchimie", rank = 145, max = 150 } }
            local key, entry = ns.buildEntry({}, read({ skills = skills }), 1763000000)
            assert.are.equal("Thrallina-Everlook", key)
            assert.are.same({ name = "Thrallina", realm = "Everlook", class = "SHAMAN", race = "Orc", level = 23,
                skills = skills, seen = 1763000000 }, entry)
        end)

        it("finds the old entry itself and keeps its fields on nil and secret", function()
            local old = { name = "Thrallina", realm = "Everlook", class = "SHAMAN", race = "Orc", level = 22,
                skills = { { id = 1, name = "", rank = 1, max = 1 } }, seen = 5 }
            local characters = { ["Thrallina-Everlook"] = old }
            local _, entry = ns.buildEntry(characters, read({ race = wow.secret(), level = wow.secret() }), 9)
            assert.are.equal("Orc", entry.race)
            assert.are.equal(22, entry.level)
            assert.are.equal(1, #entry.skills)
            assert.are.equal(9, entry.seen)
            assert.are.equal(22, old.level)
        end)

        it("creates nothing without a usable name or realm, nor a remembered one", function()
            assert.is_nil(ns.buildEntry({}, read({ name = wow.secret() }), 9))
            assert.is_nil(ns.buildEntry({}, read({ realm = wow.secret() }), 9))
            assert.is_nil(ns.buildEntry({}, read({ name = wow.secret() }), 9, {}))
        end)

        it("creates nothing without a class, but updates an entry that has one", function()
            assert.is_nil(ns.buildEntry({}, read({ class = wow.secret() }), 9))
            local old = { name = "Thrallina", realm = "Everlook", class = "SHAMAN", seen = 5 }
            local characters = { ["Thrallina-Everlook"] = old }
            local _, entry = ns.buildEntry(characters, read({ class = wow.secret(), level = 24 }), 9)
            assert.are.equal("SHAMAN", entry.class)
            assert.are.equal(24, entry.level)
            assert.are.equal(5, entry.seen) -- the read did not give the class, so it is not a refresh
        end)

        it("uses the remembered character for a read the client would not name, without moving seen", function()
            local old = { name = "Thrallina", realm = "Everlook", class = "SHAMAN", level = 22, seen = 5 }
            local characters = { ["Thrallina-Everlook"] = old }
            local remembered = { name = "Thrallina", realm = "Everlook" }
            local key, entry = ns.buildEntry(characters, read({ name = wow.secret(), level = 24 }), 9, remembered)
            assert.are.equal("Thrallina-Everlook", key)
            assert.are.equal(24, entry.level)
            assert.are.equal(5, entry.seen)
        end)

        it("does not move seen when the read did not name the character, class or not", function()
            local old = { name = "Thrallina", realm = "Everlook", class = "SHAMAN", seen = 5 }
            local characters = { ["Thrallina-Everlook"] = old }
            local remembered = { name = "Thrallina", realm = "Everlook" }
            local _, entry = ns.buildEntry(characters, read({ name = wow.secret() }), 9, remembered)
            assert.are.equal(5, entry.seen)
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

        it("keeps the old list when any line has a secret or missing id, rank or maxRank", function()
            for _, bad in ipairs({ wow.skill(171, wow.secret(), 150), wow.skill(wow.secret(), 1, 2),
                wow.skill(171, 1, nil), wow.skill(171, nil, 150) }) do
                assert.is_nil(ns.collectSkills(2, lines({ wow.skill(164, 1, 75), bad })))
            end
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

        it("keeps a version 1 table and makes sure of its two tables", function()
            local kept = { name = "A", realm = "R", class = "MAGE" }
            local db = ns.normalise({ version = 1, characters = { ["A-R"] = kept }, log = { "a" } })
            assert.are.equal(kept, db.characters["A-R"])
            assert.are.same({ "a" }, db.log)
            local repaired = ns.normalise({ version = 1, characters = "x", log = 3 })
            assert.are.same({ version = 1, characters = {}, log = {} }, repaired)
        end)
    end)
end)
