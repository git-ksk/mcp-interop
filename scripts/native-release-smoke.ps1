param(
  [Parameter(Mandatory=$true)][string]$Binary,
  [Parameter(Mandatory=$true)][string]$ExpectedVersion
)
$ErrorActionPreference = 'Stop'
$version = & $Binary version
if ($LASTEXITCODE -ne 0 -or ($version -notmatch [regex]::Escape($ExpectedVersion))) { throw "embedded version smoke failed: $version" }
$maturityJson = & $Binary maturity --json
if ($LASTEXITCODE -ne 0) { throw 'maturity command failed' }
$maturity = $maturityJson | ConvertFrom-Json
if ($maturity.artifact_type -ne 'mcp-interop/adapter-maturity') { throw 'unexpected maturity artifact type' }
if ($maturity.decisions.Count -ne 3) { throw "unexpected shipped adapter count: $($maturity.decisions.Count)" }
foreach ($decision in $maturity.decisions) {
  if ($decision.maturity -notin @('beta','stable')) { throw "unexpected maturity state: $($decision.maturity)" }
}
Write-Host "native packaged archive smoke: PASS ($ExpectedVersion)"
