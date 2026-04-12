param(
  [Parameter(Mandatory = $true)]
  [string]$MsiPath,

  [ValidateSet('amd64', 'arm64')]
  [string]$GoArch = 'amd64'
)

$ErrorActionPreference = 'Stop'

function Get-MsiPropertyValue {
  param(
    [Parameter(Mandatory = $true)]
    [object]$Database,

    [Parameter(Mandatory = $true)]
    [string]$PropertyName
  )

  $query = "SELECT `Value` FROM `Property` WHERE `Property`='$PropertyName'"
  $view = $Database.OpenView($query)
  $view.Execute()
  $record = $view.Fetch()
  if ($null -eq $record) {
    throw "MSI property '$PropertyName' was not found"
  }

  return $record.StringData(1)
}

function ConvertTo-NormalizedPathValue {
  param(
    [Parameter(Mandatory = $true)]
    [string]$PathValue
  )

  return $PathValue.Trim().Trim('"').TrimEnd('\\').ToLowerInvariant()
}

function Test-PathContainsEntry {
  param(
    [Parameter(Mandatory = $true)]
    [string]$PathValue,

    [Parameter(Mandatory = $true)]
    [string]$ExpectedEntry
  )

  if ([string]::IsNullOrWhiteSpace($PathValue)) {
    return $false
  }

  $normalizedExpected = ConvertTo-NormalizedPathValue -PathValue $ExpectedEntry
  foreach ($entry in $PathValue.Split(';')) {
    if ([string]::IsNullOrWhiteSpace($entry)) {
      continue
    }

    if ((ConvertTo-NormalizedPathValue -PathValue $entry) -eq $normalizedExpected) {
      return $true
    }
  }

  return $false
}

function Get-PathEntryList {
  param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('User', 'Machine')]
    [string]$Target
  )

  $pathValue = [Environment]::GetEnvironmentVariable('Path', $Target)
  if ([string]::IsNullOrWhiteSpace($pathValue)) {
    return @()
  }

  return @(
    $pathValue.Split(';') |
      ForEach-Object { $_.Trim() } |
      Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
  )
}

function Resolve-QrrunInstallDir {
  param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('User', 'Machine')]
    [string]$Target,

    [string[]]$PreferredDirs = @()
  )

  foreach ($entry in (Get-PathEntryList -Target $Target)) {
    $exePath = Join-Path $entry 'qrrun.exe'
    if (Test-Path -LiteralPath $exePath -PathType Leaf) {
      return $entry
    }
  }

  foreach ($dir in $PreferredDirs) {
    if ([string]::IsNullOrWhiteSpace($dir)) {
      continue
    }

    $exePath = Join-Path $dir 'qrrun.exe'
    if (Test-Path -LiteralPath $exePath -PathType Leaf) {
      return $dir
    }
  }

  return $null
}

function Assert-PathEntryState {
  param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('User', 'Machine')]
    [string]$Target,

    [Parameter(Mandatory = $true)]
    [string]$ExpectedEntry,

    [Parameter(Mandatory = $true)]
    [bool]$ShouldExist
  )

  $pathValue = [Environment]::GetEnvironmentVariable('Path', $Target)
  $exists = Test-PathContainsEntry -PathValue $pathValue -ExpectedEntry $ExpectedEntry

  if ($ShouldExist -and -not $exists) {
    throw "Expected PATH entry '$ExpectedEntry' in $Target scope was not found"
  }

  if (-not $ShouldExist -and $exists) {
    throw "PATH entry '$ExpectedEntry' in $Target scope should be removed"
  }
}

function Assert-FileState {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Path,

    [Parameter(Mandatory = $true)]
    [bool]$ShouldExist
  )

  $exists = Test-Path -LiteralPath $Path -PathType Leaf
  if ($ShouldExist -and -not $exists) {
    throw "Expected file was not found: $Path"
  }

  if (-not $ShouldExist -and $exists) {
    throw "File should be removed but still exists: $Path"
  }
}

function Assert-QrrunExecutable {
  param(
    [Parameter(Mandatory = $true)]
    [string]$ExePath
  )

  Assert-FileState -Path $ExePath -ShouldExist $true

  $output = & $ExePath --version 2>&1
  if ($LASTEXITCODE -ne 0) {
    throw "Executable check failed for '$ExePath'. Exit code: $LASTEXITCODE"
  }

  if ($output -notmatch 'commit:') {
    throw "Unexpected version output from '$ExePath': $output"
  }
}

function Write-MsiLogOnFailure {
  param(
    [Parameter(Mandatory = $true)]
    [string]$LogPath
  )

  if (-not (Test-Path -LiteralPath $LogPath -PathType Leaf)) {
    Write-Output "msiexec log file was not found: $LogPath"
    return
  }

  Write-Output "----- Begin msiexec log: $LogPath -----"
  Get-Content -LiteralPath $LogPath
  Write-Output "----- End msiexec log: $LogPath -----"
}

function Invoke-MsiExec {
  param(
    [Parameter(Mandatory = $true)]
    [string]$OperationName,

    [Parameter(Mandatory = $true)]
    [string]$Arguments,

    [int[]]$AllowedExitCodes = @(0, 3010)
  )

  $logPath = Join-Path $script:LogRoot "${OperationName}.log"
  $fullArguments = "$Arguments /L*v `"$logPath`""

  Write-Output "Running: msiexec.exe $fullArguments"
  $process = Start-Process -FilePath 'msiexec.exe' -ArgumentList $fullArguments -NoNewWindow -Wait -PassThru
  if ($AllowedExitCodes -notcontains $process.ExitCode) {
    Write-MsiLogOnFailure -LogPath $logPath
    throw "msiexec failed in '$OperationName' with exit code $($process.ExitCode). See $logPath"
  }

  return $logPath
}

function Invoke-BestEffortUninstall {
  param(
    [Parameter(Mandatory = $true)]
    [string]$ScopeArguments,

    [Parameter(Mandatory = $true)]
    [string]$OperationName
  )

  Invoke-MsiExec -OperationName $OperationName -Arguments "/x `"$script:ResolvedMsiPath`" /qn /norestart $ScopeArguments" -AllowedExitCodes @(0, 3010, 1605, 1614) | Out-Null
}

function Test-InstallExecutionSupported {
  param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('amd64', 'arm64')]
    [string]$TargetGoArch
  )

  if ($TargetGoArch -ne 'amd64') {
    Write-Information "Skipping install flow tests for $TargetGoArch MSI on this runner" -InformationAction Continue
    return $false
  }

  return $true
}

function Test-UserScope {
  $scopeArgs = 'ALLUSERS=2 MSIINSTALLPERUSER=1'
  $preferredInstallDirs = @(
    (Join-Path (Join-Path $env:LOCALAPPDATA 'Programs') 'qrrun'),
    (Join-Path $env:LOCALAPPDATA 'qrrun')
  )

  Write-Output 'Running user-scope install, repair, reinstall, and uninstall checks'
  Invoke-BestEffortUninstall -ScopeArguments $scopeArgs -OperationName 'cleanup-user-before'

  Invoke-MsiExec -OperationName 'install-user' -Arguments "/i `"$script:ResolvedMsiPath`" /qn /norestart $scopeArgs" | Out-Null
  $installDir = Resolve-QrrunInstallDir -Target User -PreferredDirs $preferredInstallDirs
  if ($null -eq $installDir) {
    throw "qrrun.exe was not found after user-scope install"
  }

  $exePath = Join-Path $installDir 'qrrun.exe'
  Assert-QrrunExecutable -ExePath $exePath
  Assert-PathEntryState -Target User -ExpectedEntry $installDir -ShouldExist $true

  Remove-Item -LiteralPath $exePath -Force
  Assert-FileState -Path $exePath -ShouldExist $false

  Invoke-MsiExec -OperationName 'repair-user' -Arguments "/fa `"$script:ResolvedMsiPath`" /qn /norestart $scopeArgs REINSTALL=ALL REINSTALLMODE=vomus" | Out-Null
  Assert-QrrunExecutable -ExePath $exePath

  Invoke-MsiExec -OperationName 'reinstall-user' -Arguments "/i `"$script:ResolvedMsiPath`" /qn /norestart $scopeArgs" | Out-Null
  Assert-QrrunExecutable -ExePath $exePath

  Invoke-MsiExec -OperationName 'uninstall-user' -Arguments "/x `"$script:ResolvedMsiPath`" /qn /norestart $scopeArgs" -AllowedExitCodes @(0, 3010, 1605, 1614) | Out-Null
  Assert-FileState -Path $exePath -ShouldExist $false
  Assert-PathEntryState -Target User -ExpectedEntry $installDir -ShouldExist $false
}

function Test-MachineScope {
  $scopeArgs = 'ALLUSERS=1'
  $preferredInstallDirs = @(
    (Join-Path $env:ProgramFiles 'qrrun'),
    (Join-Path ${env:ProgramFiles(x86)} 'qrrun')
  )

  Write-Output 'Running machine-scope install, repair, reinstall, and uninstall checks'
  Invoke-BestEffortUninstall -ScopeArguments $scopeArgs -OperationName 'cleanup-machine-before'

  Invoke-MsiExec -OperationName 'install-machine' -Arguments "/i `"$script:ResolvedMsiPath`" /qn /norestart $scopeArgs" | Out-Null
  $installDir = Resolve-QrrunInstallDir -Target Machine -PreferredDirs $preferredInstallDirs
  if ($null -eq $installDir) {
    throw "qrrun.exe was not found after machine-scope install"
  }

  $exePath = Join-Path $installDir 'qrrun.exe'
  Assert-QrrunExecutable -ExePath $exePath
  Assert-PathEntryState -Target Machine -ExpectedEntry $installDir -ShouldExist $true

  Remove-Item -LiteralPath $exePath -Force
  Assert-FileState -Path $exePath -ShouldExist $false

  Invoke-MsiExec -OperationName 'repair-machine' -Arguments "/fa `"$script:ResolvedMsiPath`" /qn /norestart $scopeArgs REINSTALL=ALL REINSTALLMODE=vomus" | Out-Null
  Assert-QrrunExecutable -ExePath $exePath

  Invoke-MsiExec -OperationName 'reinstall-machine' -Arguments "/i `"$script:ResolvedMsiPath`" /qn /norestart $scopeArgs" | Out-Null
  Assert-QrrunExecutable -ExePath $exePath

  Invoke-MsiExec -OperationName 'uninstall-machine' -Arguments "/x `"$script:ResolvedMsiPath`" /qn /norestart $scopeArgs" -AllowedExitCodes @(0, 3010, 1605, 1614) | Out-Null
  Assert-FileState -Path $exePath -ShouldExist $false
  Assert-PathEntryState -Target Machine -ExpectedEntry $installDir -ShouldExist $false
}

$script:ResolvedMsiPath = (Resolve-Path -Path $MsiPath).Path
$script:LogRoot = Join-Path (Split-Path -Parent $script:ResolvedMsiPath) 'msi-validation-logs'
New-Item -ItemType Directory -Path $script:LogRoot -Force | Out-Null

Write-Output "Validating MSI metadata: $script:ResolvedMsiPath"
$installer = New-Object -ComObject WindowsInstaller.Installer
$database = $installer.OpenDatabase($script:ResolvedMsiPath, 0)

$productName = Get-MsiPropertyValue -Database $database -PropertyName 'ProductName'
$productVersion = Get-MsiPropertyValue -Database $database -PropertyName 'ProductVersion'
$upgradeCode = Get-MsiPropertyValue -Database $database -PropertyName 'UpgradeCode'

if ($productName -ne 'qrrun') {
  throw "Unexpected ProductName '$productName'"
}

if ($productVersion -notmatch '^\d+\.\d+\.\d+\.\d+$') {
  throw "Unexpected ProductVersion format '$productVersion'"
}

$expectedUpgradeCode = 'B1A2C8E2-3E4B-4F93-ABF7-D39C45FB0C6D'
if ([string]::IsNullOrWhiteSpace([string]$upgradeCode)) {
  throw 'UpgradeCode property is empty'
}

$normalizedUpgradeCode = ([string]$upgradeCode).Trim().TrimStart('{').TrimEnd('}').ToUpperInvariant()
if ($normalizedUpgradeCode -ne $expectedUpgradeCode) {
  throw "Unexpected UpgradeCode '$upgradeCode'"
}

Write-Output "MSI metadata validated (ProductVersion=$productVersion)"

if (Test-InstallExecutionSupported -TargetGoArch $GoArch) {
  Test-UserScope
  Test-MachineScope
}

Write-Output "MSI validation completed successfully. Logs: $script:LogRoot"
