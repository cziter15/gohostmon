@echo off
setlocal

REM Platforms and architectures to build for
set platforms=windows linux darwin
set archs=amd64

REM Loop through platforms and architectures
for %%P in (%platforms%) do (
    for %%A in (%archs%) do (
        call :buildBinary %%P %%A
    )
)

exit /b

:buildBinary
REM Build binary for given OS and architecture
REM Parameters:
REM   %1 - GOOS (platform)
REM   %2 - GOARCH (architecture)

set GOOS=%1
set GOARCH=%2

echo Building binary for %GOOS%/%GOARCH%...

REM Check if the target is Windows, and append .exe extension
set OUTPUT_BIN=bin\%GOARCH%-%GOOS%-gohostmon

if "%GOOS%"=="windows" (
    set OUTPUT_BIN=%OUTPUT_BIN%.exe
)

go build -ldflags="-s -w" -o %OUTPUT_BIN%

if %errorlevel% neq 0 (
    echo Failed to build %GOOS% %GOARCH% binary. Exiting.
    exit /b 1
) else (
    echo Successfully built %GOOS% %GOARCH% binary at %OUTPUT_BIN%.
)

exit /b 0
