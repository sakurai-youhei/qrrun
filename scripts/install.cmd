@echo off
setlocal EnableExtensions EnableDelayedExpansion

set "REPO=sakurai-youhei/qrrun"
set "VERSION=%~1"
set "SCOPE=%~2"

if "%~1"=="--help" goto :usage
if "%~1"=="-h" goto :usage

if "%SCOPE%"=="" set "SCOPE=user"
if /I not "%SCOPE%"=="user" if /I not "%SCOPE%"=="machine" (
  echo Invalid scope: %SCOPE%
  echo Scope must be user or machine.
  exit /b 1
)

if "%VERSION%"=="" (
  for /f "usebackq delims=" %%U in (`curl -fsSL -o NUL -w "%%{url_effective}" "https://github.com/%REPO%/releases/latest"`) do set "LATEST_URL=%%U"
  if not defined LATEST_URL (
    echo Failed to resolve latest release URL.
    exit /b 1
  )
  for /f "tokens=* delims=/" %%T in ("!LATEST_URL!") do set "VERSION=%%T"
)

set "GOARCH=amd64"
if /I "%PROCESSOR_ARCHITECTURE%"=="ARM64" set "GOARCH=arm64"
if /I "%PROCESSOR_ARCHITEW6432%"=="ARM64" set "GOARCH=arm64"

set "MSI_URL=https://github.com/%REPO%/releases/download/%VERSION%/qrrun_%VERSION%_windows_%GOARCH%.msi"

if /I "%SCOPE%"=="machine" (
  set "MSI_SCOPE_PROPS=ALLUSERS=1"
) else (
  set "MSI_SCOPE_PROPS=ALLUSERS=2 MSIINSTALLPERUSER=1"
)

echo Installing qrrun %VERSION% (%GOARCH%, %SCOPE%)
echo Source: %MSI_URL%

msiexec /i "%MSI_URL%" /passive /norestart %MSI_SCOPE_PROPS%
if %ERRORLEVEL% EQU 0 (
  echo Installation completed successfully.
  exit /b 0
)

echo Direct URL install failed with code %ERRORLEVEL%. Falling back to curl download.
set "MSI_FILE=%TEMP%\qrrun_%VERSION%_%GOARCH%.msi"
curl -fL "%MSI_URL%" -o "%MSI_FILE%"
if errorlevel 1 (
  echo Failed to download MSI from %MSI_URL%
  exit /b 1
)

msiexec /i "%MSI_FILE%" /passive /norestart %MSI_SCOPE_PROPS%
set "INSTALL_EXIT=%ERRORLEVEL%"
del /q "%MSI_FILE%" >NUL 2>&1

if %INSTALL_EXIT% NEQ 0 (
  echo Installation failed with code %INSTALL_EXIT%.
  exit /b %INSTALL_EXIT%
)

echo Installation completed successfully.
exit /b 0

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
exit /b 0
