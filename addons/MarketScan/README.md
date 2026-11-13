# MarketScan

One button scans the whole auction house and writes per-item prices into SavedVariables.
It reads only: it never bids, buys, posts or cancels (D5), and it never scans by itself.

## Install

Run `tools/link.ps1 MarketScan` from the checkout (a directory junction into
`_forever_\Interface\AddOns`), then restart the client once. Afterwards `/reload` picks up edits.

## Use

1. Open the auction house. A "Scan prices" button appears on its window.
2. Press it. If the auction house is busy you get one chat line and nothing happens; try again later.
3. When the list arrives the addon reads it in slices of 500 rows per frame and prints one line,
   for example `MarketScan: 1,234 auctions, 456 items, 3 skipped. Saved; /reload or log out writes the file.`
4. The file `WTF/Account/<ACCOUNT>/SavedVariables/MarketScan.lua` is written on `/reload` or logout.

Slash commands (`/marketscan` or `/ms`): `status` (last scan, counts), `log` (print the log lines),
`clear` then `clear confirm` (wipe the stored scans).

## The data

```lua
MarketScanDB = {
    version = 1,
    scans = {            -- oldest first, at most 20
        { t = 1700000000, realm = "Name", rows = 1234, skipped = 3,
          items = { [2589] = "25:30:1800:42", ... } },   -- itemID = "min:median:qty:n"
    },
    log = { "2026-11-09 20:15:01 scan: ..." },            -- ring buffer, at most 50 lines
}
```

Prices are integer copper per unit. Per item: `min` lowest unit buyout, `median` unit buyout weighted
by stack size (the lower middle when the weight is even), `qty` total units for sale, `n` number of
auctions. Auctions without a buyout are ignored. `skipped` counts rows that could not be read
(secret values or malformed fields). `tools/parse_sv.py` reads the file.

## Verify in client

Tests cover the logic and a simulated client; these need the real one.

1. Interface 16001: `/dump select(4, GetBuildInfo())`.
2. `/dump C_AuctionHouse.ReplicateItems` prints `function` at the auction house.
3. Does a second scan within 15 minutes get refused, and how (silence, an error, an empty list)?
   Press the button twice and read `/marketscan log`.
4. Row count and `MarketScan.lua` size on your realm after one scan.
5. Is `skipped` above 0 on a real scan? Then some field came back as a secret value; note which
   (the log does not say; test with `/run local a=C_AuctionHouse.GetReplicateItemInfo(0) print(pcall(function() return a+0 end))`).
6. Does the button show and sit sensibly on `AuctionHouseFrame`? If the frame has another name,
   the log says "AuctionHouseFrame missing".
7. Does the file load back at login: `/dump MarketScanDB and #MarketScanDB.scans` after a relog.
8. Does `REPLICATE_ITEM_LIST_UPDATE` fire once or several times per scan? Only the first is used.
9. Does `AuctionHouseFrame` exist when `AUCTION_HOUSE_SHOW` reaches the addon (does the button appear),
   and is its TOPRIGHT placement acceptable?
10. `/run print(format("%d", 1e11))`: is `%d` 64-bit in this client? (The addon avoids it.)
11. The written file is `MarketScan.lua` and holds `MarketScanDB`.
12. Does a 100k-row scan hitch? (500 rows per frame is a guess.)
