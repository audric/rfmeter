<#
.SYNOPSIS
    Build helper for rfmeter on Windows (PowerShell equivalent of the Makefile).

.DESCRIPTION
    Run from the repo root:

        .\build.ps1 [target]

    Targets:
        build / windows   build rfmeter.exe (GUI subsystem, no console window)  [default]
        linux             cross-build the Linux binary ./rfmeter
        run               build and run rfmeter.exe
        test              go test -race ./...
        vet               go vet ./...
        fmt               gofmt -w .
        icon              regenerate the .ico + embedded .syso (needs ImageMagick)
        clean             remove build artifacts

    If scripts are blocked, launch with:

        powershell -ExecutionPolicy Bypass -File .\build.ps1 windows
#>
[CmdletBinding()]
param(
    [ValidateSet('build', 'windows', 'linux', 'run', 'test', 'vet', 'fmt', 'icon', 'clean')]
    [string]$Target = 'build'
)

$ErrorActionPreference = 'Stop'

$Pkg        = './cmd/rfmeter'
$Bin        = 'rfmeter'
$LdFlags    = '-s -w'
$WinLdFlags = "$LdFlags -H=windowsgui"

$IconDir = 'build/icon'
$IconPng = "$IconDir/rfmeter-256.png"
$IconIco = "$IconDir/rfmeter.ico"
$Syso    = 'cmd/rfmeter/rfmeter_windows_amd64.syso'

# Run a command and stop the script if it returns a non-zero exit code.
function Invoke-Step {
    param([scriptblock]$Cmd)
    & $Cmd
    if ($LASTEXITCODE -ne 0) { throw "command failed (exit $LASTEXITCODE)" }
}

function Build-Windows {
    Invoke-Step { go build "-ldflags=$WinLdFlags" -o "$Bin.exe" $Pkg }
    Write-Host "built $Bin.exe" -ForegroundColor Green
}

function Build-Linux {
    $env:GOOS = 'linux'; $env:GOARCH = 'amd64'
    try { Invoke-Step { go build "-ldflags=$LdFlags" -o $Bin $Pkg } }
    finally { Remove-Item Env:GOOS, Env:GOARCH -ErrorAction SilentlyContinue }
    Write-Host "built $Bin" -ForegroundColor Green
}

function Build-Icon {
    Invoke-Step { magick $IconPng -define icon:auto-resize=256,128,64,48,32,16 $IconIco }
    Invoke-Step { go run github.com/akavel/rsrc@latest -ico $IconIco -arch amd64 -o $Syso }
    Write-Host "regenerated $Syso" -ForegroundColor Green
}

switch ($Target) {
    { $_ -in 'build', 'windows' } { Build-Windows }
    'linux'                       { Build-Linux }
    'run'                         { Build-Windows; Invoke-Step { & ".\$Bin.exe" } }
    'test'                        { Invoke-Step { go test -race ./... } }
    'vet'                         { Invoke-Step { go vet ./... } }
    'fmt'                         { Invoke-Step { gofmt -w . } }
    'icon'                        { Build-Icon }
    'clean' {
        Remove-Item -Force -ErrorAction SilentlyContinue "$Bin.exe", $Bin
        Write-Host 'cleaned' -ForegroundColor Green
    }
}
