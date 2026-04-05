@echo off
setlocal enabledelayedexpansion
echo ==========================================
echo   Veo3 Manager - Production Build v1.0.0
echo ==========================================
echo.

:: Check Wails CLI
where wails >nul 2>nul
if %errorlevel% neq 0 (
    echo [ERROR] Wails CLI not found!
    echo Install: go install github.com/wailsapp/wails/v2/cmd/wails@latest
    exit /b 1
)

:: Parse arguments
set "BUILD_NSIS=0"
set "BUILD_UPX=0"
for %%a in (%*) do (
    if "%%a"=="--nsis" set "BUILD_NSIS=1"
    if "%%a"=="--upx" set "BUILD_UPX=1"
    if "%%a"=="--all" (
        set "BUILD_NSIS=1"
        set "BUILD_UPX=1"
    )
)

:: Step 1: Build Windows exe
echo [1/3] Building Windows exe (production)...
echo       Flags: -clean -trimpath -ldflags "-s -w"
echo       -clean    = Clear old build cache
echo       -trimpath = Remove local paths from binary
echo       -ldflags  = -s (strip symbols) -w (strip DWARF debug)
echo.

if "%BUILD_NSIS%"=="1" (
    where makensis >nul 2>nul
    if %errorlevel% equ 0 (
        echo       + NSIS installer enabled
        wails build -clean -trimpath -ldflags "-s -w" -nsis
    ) else (
        echo [WARN] NSIS not found, building without installer
        echo        Install: choco install nsis  OR  https://nsis.sourceforge.io/Download
        wails build -clean -trimpath -ldflags "-s -w"
    )
) else (
    wails build -clean -trimpath -ldflags "-s -w"
)

if %errorlevel% neq 0 (
    echo [ERROR] Build failed!
    exit /b 1
)
echo [OK] Build successful

:: Step 2: UPX compress (optional)
if "%BUILD_UPX%"=="1" (
    where upx >nul 2>nul
    if %errorlevel% equ 0 (
        echo.
        echo [2/3] Compressing with UPX...
        upx --best --lzma build\bin\Veo3Manager.exe
        echo [OK] Compressed
    ) else (
        echo.
        echo [2/3] UPX not found, skipping compression
        echo       Install: choco install upx  OR  https://github.com/upx/upx/releases
    )
) else (
    echo.
    echo [2/3] UPX skipped (use --upx to enable)
)

:: Step 3: Report
echo.
echo [3/3] Build summary:
echo.
for %%f in (build\bin\Veo3Manager.exe) do (
    set "size=%%~zf"
    set /a "sizeMB=!size! / 1048576"
    echo       Veo3Manager.exe          = !sizeMB! MB
)
if exist build\bin\*installer.exe (
    for %%f in (build\bin\*installer.exe) do (
        set "size=%%~zf"
        set /a "sizeMB=!size! / 1048576"
        echo       %%~nxf = !sizeMB! MB
    )
)

echo.
echo ==========================================
echo   Build complete!
echo.
echo   Usage:
echo     build.bat              Build exe only
echo     build.bat --upx        Build + UPX compress
echo     build.bat --nsis       Build + NSIS installer
echo     build.bat --all        Build + UPX + NSIS
echo ==========================================
