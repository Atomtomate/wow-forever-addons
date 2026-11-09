<#
.SYNOPSIS
  Copy an addon's SavedVariables file into data/ and summarise it, after a /reload or logout.
.EXAMPLE
  .\tools\pull_sv.ps1 MarketScan
  .\tools\pull_sv.ps1 MarketScan -Account 123456#1
.NOTES
  The client writes WTF\Account\<ACCOUNT>\SavedVariables\<Name>.lua only on /reload or logout.
  With several accounts in WTF\Account, pass -Account; otherwise the newest file wins.
  data/ is git-ignored: game data stays on this machine unless you choose to share it.
#>
param(
  [Parameter(Mandatory = $true)][string]$Name,
  [string]$WowDir = "C:\Program Files (x86)\World of Warcraft",
  [string]$Flavor = "_forever_",
  [string]$Account = ""
)
$ErrorActionPreference = "Stop"
$root = Resolve-Path (Join-Path $PSScriptRoot "..")
$wtf = Join-Path $WowDir "$Flavor\WTF\Account"
$pattern = if ($Account) { Join-Path $wtf "$Account\SavedVariables\$Name.lua" } else { Join-Path $wtf "*\SavedVariables\$Name.lua" }
$file = Get-ChildItem $pattern -ErrorAction SilentlyContinue | Sort-Object LastWriteTime -Descending | Select-Object -First 1
if (-not $file) { throw "No SavedVariables file matches $pattern. Did you /reload after the addon ran?" }
$data = Join-Path $root "data"
New-Item -ItemType Directory -Force $data | Out-Null
Copy-Item $file.FullName (Join-Path $data "$Name.lua") -Force
Write-Host "Copied $($file.FullName) ($([math]::Round($file.Length / 1KB)) KB, written $($file.LastWriteTime))"
python (Join-Path $root "tools\parse_sv.py") (Join-Path $data "$Name.lua") --json (Join-Path $data "$Name.json")
