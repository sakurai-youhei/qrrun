param(
  [string]$Version,
  [string]$GithubSha,
  [string]$GoArch,
  [string]$WixArch
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

  Write-Output "[build-msi] $Message"
}

function Resolve-InputValue {
  param(
    [string]$CliValue,
    [string]$EnvValue,
    [string]$DefaultValue
  )

  if (-not [string]::IsNullOrWhiteSpace($CliValue)) {
    return $CliValue
  }

  if (-not [string]::IsNullOrWhiteSpace($EnvValue)) {
    return $EnvValue
  }

  return $DefaultValue
}

function Invoke-NativeCommand {
  param(
    [Parameter(Mandatory = $true)]
    [string]$FilePath,

    [string[]]$Arguments = @()
  )

  Write-Step "Running: $FilePath $($Arguments -join ' ')"
  & $FilePath @Arguments
  if ($LASTEXITCODE -ne 0) {
    throw "Command failed: $FilePath (exit code: $LASTEXITCODE)"
  }
}

$effectiveVersion = Resolve-InputValue -CliValue $Version -EnvValue $env:VERSION -DefaultValue 'v0.0.0-dev'
$effectiveGithubSha = Resolve-InputValue -CliValue $GithubSha -EnvValue $env:GITHUB_SHA -DefaultValue ''
$effectiveGoArch = Resolve-InputValue -CliValue $GoArch -EnvValue $env:GOARCH -DefaultValue 'amd64'
$effectiveWixArch = Resolve-InputValue -CliValue $WixArch -EnvValue $env:WIX_ARCH -DefaultValue $(if ($effectiveGoArch -eq 'arm64') { 'arm64' } else { 'x64' })

if (@('amd64', 'arm64') -notcontains $effectiveGoArch) {
  throw "GOARCH must be one of: amd64, arm64 (actual: '$effectiveGoArch')"
}

if (@('x64', 'arm64') -notcontains $effectiveWixArch) {
  throw "WIX_ARCH must be one of: x64, arm64 (actual: '$effectiveWixArch')"
}

$date = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
$commit = 'unknown'
if (-not [string]::IsNullOrEmpty($effectiveGithubSha) -and $effectiveGithubSha.Length -ge 7) {
  $commit = $effectiveGithubSha.Substring(0, 7)
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$distDir = Join-Path $repoRoot 'dist'
New-Item -ItemType Directory -Path $distDir -Force | Out-Null

$base = "qrrun_${effectiveVersion}_windows_${effectiveGoArch}"
$exePath = Join-Path $distDir "${base}.exe"
$wxsPath = Join-Path $distDir "${base}.wxs"
$wixObjPath = Join-Path $distDir "${base}.wixobj"
$msiPath = Join-Path $distDir "${base}.msi"
$wxsTemplatePath = Join-Path $repoRoot 'packaging/windows/qrrun.wxs'
$binarySource = "dist/${base}.exe"

$env:CGO_ENABLED = '0'
$env:GOOS = 'windows'

Write-Step "Building Windows executable for $effectiveVersion ($effectiveGoArch)"
Invoke-NativeCommand -FilePath 'go' -Arguments @(
  'build',
  '-trimpath',
  '-ldflags', "-s -w -X 'main.version=${effectiveVersion}' -X 'main.commit=${commit}' -X 'main.date=${date}'",
  '-o', $exePath,
  './cmd/qrrun'
)

$match = [regex]::Match($effectiveVersion, '^[vV]?(\d+)\.(\d+)\.(\d+)(?:-[A-Za-z]+\.(\d+))?')
if (-not $match.Success) {
  throw "Tag '${effectiveVersion}' must start with SemVer core (e.g. v1.2.3) for MSI versioning"
}

$major = [int]$match.Groups[1].Value
$minor = [int]$match.Groups[2].Value
$patch = [int]$match.Groups[3].Value
$build = 0
if ($match.Groups[4].Success) {
  $build = [int]$match.Groups[4].Value
}

$productVersion = "$major.$minor.$patch.$build"

Write-Step "Rendering WiX template with ProductVersion=$productVersion"
$wxsTemplate = Get-Content -LiteralPath $wxsTemplatePath -Raw
$wxsRendered = $wxsTemplate.Replace('$(var.ProductVersion)', $productVersion).Replace('$(var.BinarySource)', $binarySource)
Set-Content -LiteralPath $wxsPath -Value $wxsRendered -NoNewline

Invoke-NativeCommand -FilePath 'candle' -Arguments @(
  '-nologo',
  '-arch', $effectiveWixArch,
  '-out', $wixObjPath,
  $wxsPath
)

Invoke-NativeCommand -FilePath 'light' -Arguments @(
  '-nologo',
  '-ext', 'WixUIExtension',
  '-out', $msiPath,
  $wixObjPath
)

Write-Step "MSI build completed: $msiPath"
