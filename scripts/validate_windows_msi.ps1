param(
  [Parameter(Mandatory = $true)]
  [string]$MsiPath,

  [ValidateSet('amd64', 'arm64')]
  [string]$GoArch = 'amd64'
)

$ErrorActionPreference = 'Stop'

function ConvertTo-NormalizedGuid {
  param(
    [Parameter(Mandatory = $true)]
    [string]$GuidValue
  )

  return $GuidValue.Trim().TrimStart('{').TrimEnd('}').ToUpperInvariant()
}

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

  $hostArch = $env:PROCESSOR_ARCHITECTURE.ToLowerInvariant()
  if ($TargetGoArch -eq 'arm64' -and $hostArch -ne 'arm64') {
    Write-Output "Skipping install flow tests for arm64 MSI on host architecture '$hostArch'"
    return $false
  }

  return $true
}

function Test-UserScope {
  $scopeArgs = 'ALLUSERS=2 MSIINSTALLPERUSER=1'
  $installDir = Join-Path (Join-Path $env:LOCALAPPDATA 'Programs') 'qrrun'
  $exePath = Join-Path $installDir 'qrrun.exe'

  Write-Output 'Running user-scope install, repair, reinstall, and uninstall checks'
  Invoke-BestEffortUninstall -ScopeArguments $scopeArgs -OperationName 'cleanup-user-before'

  Invoke-MsiExec -OperationName 'install-user' -Arguments "/i `"$script:ResolvedMsiPath`" /qn /norestart $scopeArgs" | Out-Null
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
  $installDir = Join-Path $env:ProgramFiles 'qrrun'
  $exePath = Join-Path $installDir 'qrrun.exe'

  Write-Output 'Running machine-scope install, repair, reinstall, and uninstall checks'
  Invoke-BestEffortUninstall -ScopeArguments $scopeArgs -OperationName 'cleanup-machine-before'

  Invoke-MsiExec -OperationName 'install-machine' -Arguments "/i `"$script:ResolvedMsiPath`" /qn /norestart $scopeArgs" | Out-Null
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
if ((ConvertTo-NormalizedGuid -GuidValue $upgradeCode) -ne $expectedUpgradeCode) {
  throw "Unexpected UpgradeCode '$upgradeCode'"
}

Write-Output "MSI metadata validated (ProductVersion=$productVersion)"

if (Test-InstallExecutionSupported -TargetGoArch $GoArch) {
  Test-UserScope
  Test-MachineScope
}

Write-Output "MSI validation completed successfully. Logs: $script:LogRoot"
