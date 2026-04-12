param(
  [Parameter(ValueFromRemainingArguments = $true)]
  [string[]]$Paths
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

if ($PSVersionTable.PSVersion.Major -ge 7) {
  $PSNativeCommandUseErrorActionPreference = $true
}

function Write-Step {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Message
  )

  Write-Output "[psscriptanalyzer] $Message"
}

Write-Step 'Ensuring PSScriptAnalyzer module is available'

$module = Get-Module -ListAvailable -Name PSScriptAnalyzer |
  Sort-Object Version -Descending |
  Select-Object -First 1

if (-not $module) {
  Write-Step 'Installing PSScriptAnalyzer module'
  Install-Module -Name PSScriptAnalyzer -Scope CurrentUser -Force -AllowClobber
}

Import-Module PSScriptAnalyzer -ErrorAction Stop

$resolvedPaths = @()
foreach ($path in $Paths) {
  if ([string]::IsNullOrWhiteSpace($path)) {
    continue
  }

  if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
    throw "File not found for analysis: $path"
  }

  $resolvedPaths += (Resolve-Path -LiteralPath $path).Path
}

if ($resolvedPaths.Count -eq 0) {
  Write-Step 'No PowerShell files provided. Skipping analysis.'
  exit 0
}

$issues = @()
foreach ($path in $resolvedPaths) {
  Write-Step "Analyzing $path"
  $issues += Invoke-ScriptAnalyzer -Path $path -Severity Error,Warning,Information
}

if ($issues) {
  Write-Step "Found $($issues.Count) issue(s)"
  $issues | Format-Table -AutoSize
  exit 1
}

Write-Step 'No issues found'
