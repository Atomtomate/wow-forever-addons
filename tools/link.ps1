<#
.SYNOPSIS
  Create a directory junction so WoW: Forever loads an addon straight from this checkout.
.EXAMPLE
  .\tools\link.ps1 MarketScan
  .\tools\link.ps1 MarketScan -WowDir "D:\Games\World of Warcraft"
.NOTES
  A junction (mklink /J) needs no admin rights. A new addon folder needs a full client restart
  the first time; afterwards /reload picks up edits. Remove with: Remove-Item <AddOns>\<Name>
  (that removes the junction only, not the checkout).
#>
param(
  [Parameter(Mandatory = $true)][string]$Name,
  [string]$WowDir = "C:\Program Files (x86)\World of Warcraft",
  [string]$Flavor = "_forever_"
)
$ErrorActionPreference = "Stop"
$src = Join-Path (Resolve-Path (Join-Path $PSScriptRoot "..\addons")) $Name
if (-not (Test-Path $src)) { throw "No addon at $src" }
$addons = Join-Path $WowDir "$Flavor\Interface\AddOns"
if (-not (Test-Path $addons)) { throw "AddOns folder not found: $addons (check -WowDir and -Flavor)" }
$dst = Join-Path $addons $Name
if (Test-Path $dst) {
  $item = Get-Item $dst
  if ($item.LinkType -eq "Junction") { Write-Host "Junction already exists: $dst -> $($item.Target)"; exit 0 }
  throw "$dst exists and is not a junction; move it away first"
}
cmd /c mklink /J "$dst" "$src" | Out-Null
Write-Host "Linked $dst -> $src. Restart the client once if the addon is new."
