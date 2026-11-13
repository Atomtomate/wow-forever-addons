-- WoW runs Lua 5.1. Every global an addon creates must be listed in its addon block below, so
-- an accidental global (which can taint Blizzard code) fails the lint instead of shipping.
std = "lua51"
max_line_length = 120
codes = true
exclude_files = { "data/**", ".release/**", "**/Libs/**" }

-- WoW API the addons read. Extend per luacheck warning; never add a write here.
read_globals = {
    -- frames, events, timers
    "CreateFrame", "C_Timer", "UIParent", "GameTooltip", "DEFAULT_CHAT_FRAME",
    -- time and build
    "GetServerTime", "GetTime", "GetBuildInfo", "date", "time",
    -- player and realm
    "GetRealmName", "UnitName", "UnitFullName", "GetNormalizedRealmName", "UnitClass", "UnitRace", "UnitLevel",
    "C_Secrets", "C_SkillInfo",
    -- items and money
    "C_Item", "GetCoinTextureString", "C_CurrencyInfo",
    -- auction house (read-only calls only; D5)
    "C_AuctionHouse", "AuctionHouseFrame",
    -- addon loading
    "C_AddOns", "IsLoggedIn",
    -- chat and slash
    "SlashCmdList", "print", "format", "strsplit", "strtrim", "tinsert", "tremove", "wipe",
    -- utilities Blizzard exposes as globals
    "hooksecurefunc", "issecurevariable", "securecall", "geterrorhandler", "debugstack",
}

files["addons/MarketScan/**"] = {
    globals = { "MarketScanDB", "SLASH_MARKETSCAN1", "SLASH_MARKETSCAN2", SlashCmdList = { fields = { "MARKETSCAN" } } },
}

files["addons/ForeverFriends/**"] = {
    globals = {
        "ForeverFriendsDB", "SLASH_FOREVERFRIENDS1", "SLASH_FOREVERFRIENDS2",
        SlashCmdList = { fields = { "FOREVERFRIENDS" } },
    },
    read_globals = { "ForeverFriendsData" },
}

files["spec/**"] = {
    std = "+busted",
    globals = { "_G" },
    ignore = { "122" }, -- tests set read-only globals on purpose to simulate the client
}

files["addons/*/spec/**"] = {
    std = "+busted",
    globals = { "_G" },
    ignore = { "122" },
}

-- Pure files see no client global at all: "pure" is checked here, not only said in a comment.
for _, file in ipairs({
    "addons/ForeverFriends/Snapshot.lua", "addons/ForeverFriends/Roster.lua",
    "addons/ForeverFriends/Lines.lua", "addons/MarketScan/Pack.lua",
}) do
    files[file] = { new_globals = {}, new_read_globals = {} }
end
