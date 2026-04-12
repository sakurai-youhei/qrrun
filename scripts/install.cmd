@echo off
setlocal EnableExtensions

REM qrrun MSI installer for Windows.
REM Usage: scripts\install.cmd [version] [user|machine]
REM if version is omitted, the latest release tag is used.

set "REPO=sakurai-youhei/qrrun"
set "VERSION=%~1"
set "SCOPE=%~2"
set "QRRUN_INSTALL_NO_PAUSE=%QRRUN_INSTALL_NO_PAUSE%"

if "%~1"=="--help" goto :usage
if "%~1"=="-h" goto :usage

if "%SCOPE%"=="" set "SCOPE=user"
if /I not "%SCOPE%"=="user" if /I not "%SCOPE%"=="machine" (
  echo Invalid scope: %SCOPE%
  echo Scope must be user or machine.
  exit /b 1
)

if "%VERSION%"=="" (
  call :resolve_latest_version
  if errorlevel 1 exit /b 1
)
call :validate_version "%VERSION%"
if errorlevel 1 exit /b 1

set "GOARCH=amd64"
if /I "%PROCESSOR_ARCHITECTURE%"=="ARM64" set "GOARCH=arm64"
if /I "%PROCESSOR_ARCHITEW6432%"=="ARM64" set "GOARCH=arm64"

set "MSI_URL=https://github.com/%REPO%/releases/download/%VERSION%/"
set "MSI_URL=%MSI_URL%qrrun_%VERSION%_windows_%GOARCH%.msi"

echo Installing qrrun %VERSION% (%GOARCH%, %SCOPE%)
echo Source: %MSI_URL%

call :install_from_url
if %ERRORLEVEL% EQU 0 (
  echo Installation completed successfully.
  exit /b 0
)

echo Direct URL install failed with code %ERRORLEVEL%.
echo Falling back to local MSI download.
set "MSI_FILE=%TEMP%\qrrun_%VERSION%_%GOARCH%.msi"
curl -fL "%MSI_URL%" -o "%MSI_FILE%"
if errorlevel 1 (
  echo Failed to download MSI from %MSI_URL%
  exit /b 1
)

call :install_from_file
set "INSTALL_EXIT=%ERRORLEVEL%"
if exist "%MSI_FILE%" del /q "%MSI_FILE%" >NUL 2>&1
if errorlevel 1 echo Warning: failed to delete temporary MSI file: %MSI_FILE%

if %INSTALL_EXIT% NEQ 0 (
  echo Installation failed with code %INSTALL_EXIT%.
  exit /b %INSTALL_EXIT%
)

echo Installation completed successfully.
exit /b 0

REM Install from direct URL.
:install_from_url
if /I "%SCOPE%"=="machine" (
  msiexec /i "%MSI_URL%" /passive /norestart ALLUSERS=1
) else (
  msiexec /i "%MSI_URL%" /passive /norestart ALLUSERS=2 MSIINSTALLPERUSER=1
)
exit /b %ERRORLEVEL%

REM Install from downloaded local MSI file.
:install_from_file
if /I "%SCOPE%"=="machine" (
  msiexec /i "%MSI_FILE%" /passive /norestart ALLUSERS=1
) else (
  msiexec /i "%MSI_FILE%" /passive /norestart ALLUSERS=2 MSIINSTALLPERUSER=1
)
exit /b %ERRORLEVEL%

REM Resolve the latest release version from GitHub redirect response.
:resolve_latest_version
set "LATEST_URL="
set "LATEST_RELEASE_URL=https://github.com/sakurai-youhei/qrrun/releases/latest"
for /f "usebackq tokens=* delims=" %%U in (`curl -fsSL -o NUL -w "%%{url_effective}" "%LATEST_RELEASE_URL%"`) do (
  set "LATEST_URL=%%U"
)
if not defined LATEST_URL (
  echo Failed to resolve latest release URL.
  exit /b 1
)

if "%LATEST_URL%"=="" (
  echo Failed to resolve latest release URL.
  exit /b 1
)

set "VERSION="
for %%S in ("%LATEST_URL:/=" "%") do set "VERSION=%%~S"
if "%VERSION%"=="" (
  echo Failed to parse latest version from URL: %LATEST_URL%
  exit /b 1
)
exit /b 0

REM Validate version token to avoid unsafe shell characters.
:validate_version
set "CHECK_VERSION=%~1"
if "%CHECK_VERSION%"=="" (
  echo Version must not be empty.
  exit /b 1
)

echo(%CHECK_VERSION:~0,1%| findstr /R "[A-Za-z0-9]" >NUL
if errorlevel 1 (
  echo Invalid version format: %CHECK_VERSION%
  echo Version must start with an alphanumeric character.
  exit /b 1
)

set "SANITIZED=%CHECK_VERSION%"
set "SANITIZED=%SANITIZED:&=%"
set "SANITIZED=%SANITIZED:|=%"
set "SANITIZED=%SANITIZED:<=%"
set "SANITIZED=%SANITIZED:>=%"
set "SANITIZED=%SANITIZED: =%"
set "SANITIZED=%SANITIZED:(=%"
set "SANITIZED=%SANITIZED:)=%"
set "SANITIZED=%SANITIZED:!=%"
set "SANITIZED=%SANITIZED:`=%"
set "SANITIZED=%SANITIZED:'=%"
if /I not "%SANITIZED%"=="%CHECK_VERSION%" (
  echo Invalid version format: %CHECK_VERSION%
  echo Version contains unsafe special characters.
  exit /b 1
)

if "%CHECK_VERSION%"=="." (
  echo Invalid version format: %CHECK_VERSION%
  exit /b 1
)
if "%CHECK_VERSION%"=="-" (
  echo Invalid version format: %CHECK_VERSION%
  exit /b 1
)
if "%CHECK_VERSION%"=="_" (
  echo Invalid version format: %CHECK_VERSION%
  exit /b 1
)

REM Keep this guard for compatibility with tools that rely on ERRORLEVEL.
if errorlevel 1 (
  echo Invalid version format: %CHECK_VERSION%
  exit /b 1
)
exit /b 0

REM Show command usage and examples.
:usage
echo Install qrrun on Windows using MSI release assets.
echo.
echo Usage:
echo   scripts\install.cmd [version] [user^|machine]
echo.
echo Examples:
echo   scripts\install.cmd
echo   scripts\install.cmd v0.1.0-beta.1
echo   scripts\install.cmd v0.1.0 machine
echo.
echo Notes:
echo   - If version is omitted, latest release is used.
echo   - user installs for current user and updates user PATH.
echo   - machine installs system-wide and updates system PATH.
if /I "%QRRUN_INSTALL_NO_PAUSE%"=="1" goto :no_pause
pause

REM End of usage flow.
:no_pause
exit /b 0
