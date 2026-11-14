# Assets

Every non-code file in `companion-app/` and where it came from.

| File | Source | Licence |
|------|--------|---------|
| `app/icons/icon.svg` | Drawn for this repository in this pull request: a rounded square with two interlocking rings (friends, forever). Plain shapes written by hand, not derived from anyone's artwork. | Original, made for this repository. Covered by the repository's licence once the owner chooses one. |
| `app/icons/32x32.png`, `128x128.png`, `128x128@2x.png`, `icon.png`, `icon.ico`, `icon.icns` | Generated from `icon.svg` with `cargo tauri icon app/icons/icon.svg -o app/icons` (tauri-cli 2.12.4, run from `companion-app/`). The extra files the command also writes (Android, iOS, Windows Store sizes) are deleted: only the six above are used. | Original, made for this repository. Covered by the repository's licence once the owner chooses one. |

The page (`app/ui/`) uses the system font and no images, fonts or scripts from anywhere else.
