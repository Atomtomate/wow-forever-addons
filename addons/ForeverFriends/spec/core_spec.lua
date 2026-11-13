local wow = require("wow")

describe("ForeverFriends Core (simulated client)", function()
    local function load()
        wow.loadAddon("ForeverFriends")
        wow.fire("ADDON_LOADED", "ForeverFriends")
    end

    local function lines(count)
        local list = {}
        for i = 1, count do list[i] = wow.skill(100 + i, i, 300) end
        wow.skills = list
    end

    local function lastLog()
        return ForeverFriendsDB.log[#ForeverFriendsDB.log]
    end

    before_each(function()
        wow.reset()
        wow.skills = { wow.skill(171, 145, 150) }
        load()
    end)

    it("initialises the database and ignores other addons", function()
        assert.are.same({ version = 1, characters = {}, log = {} }, ForeverFriendsDB)
        local db = ForeverFriendsDB
        wow.fire("ADDON_LOADED", "SomethingElse")
        assert.are.equal(db, ForeverFriendsDB)
    end)

    it("records the contract's entry at login and says nothing in chat", function()
        wow.fire("PLAYER_LOGIN")
        assert.are.same({
            name = "Thrallina", realm = "TestRealm", class = "SHAMAN", race = "Orc", level = 23,
            skills = { { id = 171, name = "Skill", rank = 145, max = 150 } }, seen = wow.time,
        }, ForeverFriendsDB.characters["Thrallina-TestRealm"])
        assert.is_truthy(lastLog():match("login: Thrallina%-TestRealm level 23$"))
        assert.are.same({}, wow.printed)
    end)

    it("keeps the characters of an existing table and starts from nil without error", function()
        wow.reset()
        local old = { name = "Old", realm = "Realm", class = "MAGE" }
        _G.ForeverFriendsDB = { version = 1, characters = { ["Old-Realm"] = old }, log = {} }
        load()
        wow.fire("PLAYER_LOGIN")
        assert.is_table(ForeverFriendsDB.characters["Old-Realm"])
        assert.is_table(ForeverFriendsDB.characters["Thrallina-TestRealm"])
    end)

    it("replaces a table of another version", function()
        wow.reset()
        _G.ForeverFriendsDB = { version = 7, characters = { x = {} } }
        load()
        assert.are.same({ version = 1, characters = {}, log = {} }, ForeverFriendsDB)
    end)

    it("records a level-up from its argument and falls back to UnitLevel", function()
        wow.fire("PLAYER_LOGIN")
        wow.fire("PLAYER_LEVEL_UP", 24)
        assert.are.equal(24, ForeverFriendsDB.characters["Thrallina-TestRealm"].level)
        assert.is_truthy(lastLog():match("level up: Thrallina%-TestRealm level 24$"))
        wow.player.level = 25
        wow.fire("PLAYER_LEVEL_UP", wow.secret())
        assert.are.equal(25, ForeverFriendsDB.characters["Thrallina-TestRealm"].level)
    end)

    it("reads the skills once, after a two second delay, for a burst of SKILL_LINES_CHANGED", function()
        wow.fire("PLAYER_LOGIN")
        local reads = 0
        local original = C_SkillInfo.GetNumSkillLines
        C_SkillInfo.GetNumSkillLines = function() reads = reads + 1 return original() end
        wow.skills = { wow.skill(171, 150, 150) }
        for _ = 1, 20 do wow.fire("SKILL_LINES_CHANGED") end
        assert.are.same({ 2 }, wow.delays)
        assert.are.equal(0, reads)
        assert.are.equal(1, wow.tick())
        assert.are.equal(1, reads)
        assert.are.equal(150, ForeverFriendsDB.characters["Thrallina-TestRealm"].skills[1].rank)
        wow.fire("SKILL_LINES_CHANGED")
        assert.are.equal(1, wow.tick())
    end)

    it("refreshes everything at logout", function()
        wow.fire("PLAYER_LOGIN")
        wow.skills = { wow.skill(164, 5, 75) }
        wow.player.level = 24
        wow.fire("PLAYER_LOGOUT")
        local entry = ForeverFriendsDB.characters["Thrallina-TestRealm"]
        assert.are.equal(164, entry.skills[1].id)
        assert.are.equal(24, entry.level)
        assert.are.equal("logout", lastLog():match("logout$"))
    end)

    it("moves only the level while identity is secret, and logs that once", function()
        wow.fire("PLAYER_LOGIN")
        local before = ForeverFriendsDB.characters["Thrallina-TestRealm"]
        wow.restrictIdentity()
        wow.player.level = 24
        wow.fire("PLAYER_LEVEL_UP", 24)
        wow.fire("PLAYER_LOGOUT")
        local entry = ForeverFriendsDB.characters["Thrallina-TestRealm"]
        assert.are.equal("SHAMAN", entry.class)
        assert.are.equal("Orc", entry.race)
        assert.are.equal(24, entry.level) -- level is not identity; the session still knows who this is
        assert.are.equal(before.seen, entry.seen)
        assert.are.equal(1, #entry.skills)
        local n = 0
        for _, line in ipairs(ForeverFriendsDB.log) do
            if line:match("identity is secret") then n = n + 1 end
        end
        assert.are.equal(1, n)
    end)

    it("records nothing at all when identity is secret from the start", function()
        wow.restrictIdentity()
        wow.fire("PLAYER_LOGIN")
        assert.are.same({}, ForeverFriendsDB.characters)
        assert.are.same({}, wow.printed)
    end)

    it("keeps the old class when only the class is secret and the predicate says no", function()
        wow.fire("PLAYER_LOGIN")
        wow.player.class = wow.secret()
        wow.player.level = 30
        wow.fire("PLAYER_LOGOUT")
        local entry = ForeverFriendsDB.characters["Thrallina-TestRealm"]
        assert.are.equal("SHAMAN", entry.class)
        assert.are.equal(30, entry.level)
    end)

    it("works without C_Secrets and with a predicate that raises", function()
        _G.C_Secrets = nil
        wow.fire("PLAYER_LOGIN")
        assert.is_table(ForeverFriendsDB.characters["Thrallina-TestRealm"])
        wow.reset()
        wow.identityRestricted = "error"
        load()
        wow.fire("PLAYER_LOGIN")
        assert.is_table(ForeverFriendsDB.characters["Thrallina-TestRealm"])
    end)

    it("keeps the old skill list when the client has none yet", function()
        wow.fire("PLAYER_LOGIN")
        wow.skills = {}
        wow.fire("PLAYER_LOGOUT")
        assert.are.equal(171, ForeverFriendsDB.characters["Thrallina-TestRealm"].skills[1].id)
    end)

    it("keeps the whole old list when one skill line cannot be read", function()
        wow.fire("PLAYER_LOGIN")
        wow.skills = { wow.skill(171, 150, 150), wow.skill(164, wow.secret(), 75) }
        wow.fire("PLAYER_LOGOUT")
        local skills = ForeverFriendsDB.characters["Thrallina-TestRealm"].skills
        assert.are.equal(1, #skills)
        assert.are.equal(145, skills[1].rank)
    end)

    it("does not move seen on a read in which the name was secret", function()
        wow.fire("PLAYER_LOGIN")
        local seen = ForeverFriendsDB.characters["Thrallina-TestRealm"].seen
        wow.time = wow.time + 100
        wow.player.name = wow.secret()
        wow.fire("PLAYER_LOGOUT")
        assert.are.equal(seen, ForeverFriendsDB.characters["Thrallina-TestRealm"].seen)
    end)

    it("survives a secret skill name and a secret level-up argument", function()
        wow.skills = { wow.skill(171, 145, 150) }
        wow.skills[1].name = wow.secret()
        wow.fire("PLAYER_LOGIN")
        assert.are.equal("", ForeverFriendsDB.characters["Thrallina-TestRealm"].skills[1].name)
        assert.has_no.errors(function() wow.fire("PLAYER_LEVEL_UP", wow.secret()) end)
    end)

    it("keeps the first 40 of 60 skill lines and logs the cut once", function()
        lines(60)
        wow.fire("PLAYER_LOGIN")
        wow.fire("PLAYER_LOGOUT")
        assert.are.equal(40, #ForeverFriendsDB.characters["Thrallina-TestRealm"].skills)
        local n = 0
        for _, line in ipairs(ForeverFriendsDB.log) do
            if line:match("skills: more than 40") then n = n + 1 end
        end
        assert.are.equal(1, n)
    end)

    it("keeps the log to 50 lines over many events", function()
        for i = 1, 100 do wow.fire("PLAYER_LEVEL_UP", 20 + (i % 5)) end
        assert.are.equal(50, #ForeverFriendsDB.log)
    end)

    describe("/ff", function()
        local data = { version = 1, written = 1, members = {
            { name = "Anna", characters = { { name = "Thrallina", class = "Shaman", level = 23, main = true } } },
        } }

        it("registers /ff and /foreverfriends", function()
            assert.are.equal("/ff", SLASH_FOREVERFRIENDS1)
            assert.are.equal("/foreverfriends", SLASH_FOREVERFRIENDS2)
        end)

        it("prints exactly one line per command", function()
            _G.ForeverFriendsData = data
            wow.fire("PLAYER_LOGIN")
            for _, message in ipairs({ "", "members", "who anna", "who nobody", "who", "status", "nonsense" }) do
                wow.printed = {}
                wow.slash("FOREVERFRIENDS", message)
                assert.are.equal(1, #wow.printed, message)
                assert.is_truthy(wow.printed[1]:match("^ForeverFriends: "))
            end
            wow.printed = {}
            wow.slash("FOREVERFRIENDS", "who anna")
            assert.are.equal("ForeverFriends: Anna: Thrallina (Shaman 23, main)", wow.printed[1])
            wow.printed = {}
            wow.slash("FOREVERFRIENDS", "status")
            assert.are.equal("ForeverFriends: 1 character recorded, this one is Thrallina-TestRealm.", wow.printed[1])
        end)

        it("answers no list yet without the data addon and for its empty first file", function()
            local no = "ForeverFriends: no member list yet. The companion writes it once your PC is linked."
            wow.slash("FOREVERFRIENDS", "")
            assert.are.equal(no, wow.printed[1])
            _G.ForeverFriendsData = { version = 1, written = 0, members = {} }
            wow.slash("FOREVERFRIENDS", "members")
            assert.are.equal(no, wow.printed[2])
        end)

        it("never raises on a garbage member list", function()
            local hostile = setmetatable({ version = 1 }, { __index = function() error("boom") end })
            local bad = { 5, "x", {}, hostile, wow.secret(), { version = 1, members = { {}, 1, "a", false } } }
            local many = {}
            for i = 1, 10000 do many[i] = {} end
            bad[#bad + 1] = { version = 1, members = many }
            for _, value in ipairs(bad) do
                _G.ForeverFriendsData = value
                for _, message in ipairs({ "", "members", "who a" }) do
                    assert.has_no.errors(function() wow.slash("FOREVERFRIENDS", message) end)
                end
            end
        end)
    end)

    describe("the toc", function()
        local function toc()
            local file = assert(io.open("addons/ForeverFriends/ForeverFriends.toc", "r"))
            local text = file:read("*a")
            file:close()
            return text
        end

        it("carries the headers the companion and the client rely on", function()
            local text = toc()
            assert.is_truthy(text:match("## Interface: 16001\n"))
            assert.is_truthy(text:match("## OptionalDeps: ForeverFriendsData\n"))
            assert.is_truthy(text:match("## SavedVariables: ForeverFriendsDB\n"))
            assert.is_truthy(text:match("## Version: %d+%.%d+%.%d+\n"))
        end)

        it("lists exactly the Lua files that exist", function()
            local listed = {}
            for file in toc():gmatch("\n(%S+%.lua)") do
                listed[#listed + 1] = file
                local handle = assert(io.open("addons/ForeverFriends/" .. file, "r"), file .. " is listed but missing")
                handle:close()
            end
            local ls = assert(io.popen("ls addons/ForeverFriends/*.lua"))
            local count = 0
            for _ in ls:lines() do count = count + 1 end
            ls:close()
            assert.are.equal(count, #listed)
        end)
    end)
end)
