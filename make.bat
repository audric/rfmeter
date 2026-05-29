@echo off
REM Windows convenience wrapper around build.ps1 — avoids the PowerShell
REM execution-policy prompt. Usage from cmd or by double-clicking:
REM     make            build rfmeter.exe (default)
REM     make windows    same as above
REM     make run        build and run
REM     make test       go test -race ./...
REM     make clean      remove build artifacts
REM (any build.ps1 target works; arguments are forwarded as-is)
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0build.ps1" %*
exit /b %ERRORLEVEL%
