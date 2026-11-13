local wow = require("wow")

describe("ForeverFriends Lines (pure)", function()
    local ns, members

    before_each(function()
        wow.reset()
        ns = {}
        for _, file in ipairs({ "Roster", "Lines" }) do
            assert(loadfile("addons/ForeverFriends/" .. file .. ".lua"))("ForeverFriends", ns)
        end
        members = {
            { name = "Anna", characters = {
                { name = "Thrallina", class = "Shaman", level = 23, main = true },
                { name = "Bankalt", class = "Mage" },
            } },
            { name = "Ben", characters = {} },
        }
    end)

    local function data(list) return { version = 1, written = 0, members = list } end
    local own = { count = 3, key = "Thrallina-Everlook" }

    it("summarises the list", function()
        assert.are.equal("ForeverFriends: 2 members, 2 characters. /ff members, /ff who <name>, /ff status.",
            ns.respond("", data(members), own))
        local one = { { name = "Anna", characters = { { name = "Thrallina" } } } }
        assert.are.equal("ForeverFriends: 1 member, 1 character. /ff members, /ff who <name>, /ff status.",
            ns.respond("", data(one), own))
    end)

    it("lists the members and finds one", function()
        assert.are.equal("ForeverFriends: Anna, Ben.", ns.respond("members", data(members), own))
        assert.are.equal("ForeverFriends: Anna: Thrallina (Shaman 23, main), Bankalt (Mage)",
            ns.respond("who thrall", data(members), own))
        assert.are.equal("ForeverFriends: Ben", ns.respond("WHO  ben ", data(members), own))
        assert.are.equal('ForeverFriends: nobody matches "zed".', ns.respond("who zed", data(members), own))
    end)

    it("answers a missing, empty, unknown or hostile list in one line", function()
        local no = "ForeverFriends: no member list yet. The companion writes it once your PC is linked."
        assert.are.equal(no, ns.respond("", nil, own))
        assert.are.equal(no, ns.respond("", data({}), own))
        assert.are.equal(no, ns.respond("members", { version = 1, written = 0, members = {} }, own))
        assert.are.equal("ForeverFriends: the member list is in a format this addon does not know.",
            ns.respond("", { version = 9 }, own))
        assert.are.equal("ForeverFriends: the member list could not be read.",
            ns.respond("", { version = 1, members = 3 }, own))
    end)

    it("prints the usage for no name or an unknown subcommand", function()
        local usage = "ForeverFriends: /ff, /ff members, /ff who <name>, /ff status."
        assert.are.equal(usage, ns.respond("who", data(members), own))
        assert.are.equal(usage, ns.respond("dance", data(members), own))
    end)

    it("says what this account recorded", function()
        assert.are.equal("ForeverFriends: 3 characters recorded, this one is Thrallina-Everlook.",
            ns.respond("status", nil, own))
        assert.are.equal("ForeverFriends: 0 characters recorded.", ns.respond("status", nil, { count = 0 }))
        assert.are.equal("ForeverFriends: 1 character recorded.", ns.respond("status", nil, { count = 1 }))
    end)

    it("doubles | and drops control characters from everything it shows", function()
        local list = { { name = "|cffff0000Red|r\n", characters = { { name = "A\0B|Hitem:1|h", class = "x\127y" } } } }
        local line = ns.respond("who |cff", data(list), own)
        assert.are.equal("ForeverFriends: ||cffff0000Red||r: AB||Hitem:1||h (xy)", line)
        assert.are.equal('ForeverFriends: nobody matches "||c".', ns.respond("who |c", data(members), own))
        assert.are.equal("a||b", ns.clean("a|b\1"))
    end)

    it("cuts a long list at a member and counts what it left out", function()
        local many, chars = {}, {}
        for i = 1, 100 do many[i] = { name = ("n"):rep(30) .. i, characters = {} } end
        for i = 1, 60 do chars[i] = { name = ("c"):rep(40) .. i, class = "Shaman", level = 60 } end
        many[1].characters = chars
        local line = ns.respond("members", data(many), own)
        assert.is_true(#line <= 240)
        assert.is_truthy(line:match("and %d+ more%.$"))
        line = ns.respond("who n", data(many), own)
        assert.is_true(#line <= 240)
        assert.is_truthy(line:match(" more$"))
    end)
end)
