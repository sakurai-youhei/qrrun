param(
  [Parameter(ValueFromRemainingArguments = $true)]
  [string[]]$Paths
)

$ErrorActionPreference = 'Stop'

$module = Get-Module -ListAvailable -Name PSScriptAnalyzer |
  Sort-Object Version -Descending |
  Select-Object -First 1

if (-not $module) {
  Install-Module -Name PSScriptAnalyzer -Scope CurrentUser -Force -AllowClobber
}

Import-Module PSScriptAnalyzer

$issues = @()
foreach ($path in $Paths) {
  if (-not [string]::IsNullOrWhiteSpace($path)) {
    $issues += Invoke-ScriptAnalyzer -Path $path -Severity Error,Warning
  }
}

if ($issues) {
  $issues | Format-Table -AutoSize
  exit 1
}
