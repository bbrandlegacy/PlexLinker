[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$preconfirmed = $env:PLEXLINKER_SETUP_CONFIRMED -eq '1'

if (-not $preconfirmed) {
    Write-Host 'PlexLinker setup will install aria2 if missing, install PlexLinker under Program Files, and update the machine PATH.'
    Write-Host 'No administrator password or reusable credential will be stored.'
    $answer = Read-Host 'Proceed with these changes? [y/N]'
    if ($answer -notmatch '^(?i:y|yes)$') {
        Write-Host 'Setup cancelled.'
        exit 1
    }
    $env:PLEXLINKER_SETUP_CONFIRMED = '1'
}

$identity = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = [Security.Principal.WindowsPrincipal]::new($identity)
$isAdmin = $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host 'PlexLinker setup requires administrator approval. Windows will display a UAC prompt.'
    $arguments = @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', ('"{0}"' -f $PSCommandPath))
    $process = Start-Process powershell.exe -Verb RunAs -ArgumentList $arguments -Wait -PassThru
    exit $process.ExitCode
}

$source = $env:PLEXLINKER_SOURCE_EXE
if ([string]::IsNullOrWhiteSpace($source)) {
    $source = Join-Path $PSScriptRoot 'plexlinker-v2.exe'
}
if (-not (Test-Path -LiteralPath $source -PathType Leaf)) {
    throw "PlexLinker binary not found: $source"
}

Write-Host 'PlexLinker Windows setup'
Write-Host "  Source: $source"
Write-Host '  Install: C:\Program Files\PlexLinker\plexlinker.exe'
Write-Host '  Dependency: aria2 (installed with WinGet only when missing)'
Write-Host 'Administrator authentication is handled by UAC and is never read or stored by this script.'

if (-not (Get-Command aria2c.exe -ErrorAction SilentlyContinue)) {
    if (Get-Command winget.exe -ErrorAction SilentlyContinue) {
        & winget.exe install --id aria2.aria2 --exact --accept-package-agreements --accept-source-agreements --silent
        if ($LASTEXITCODE -ne 0) {
            throw "WinGet failed to install aria2 (exit code $LASTEXITCODE)."
        }
    } elseif (Get-Command choco.exe -ErrorAction SilentlyContinue) {
        & choco.exe install aria2 --yes --no-progress
        if ($LASTEXITCODE -ne 0) {
            throw "Chocolatey failed to install aria2 (exit code $LASTEXITCODE)."
        }
    } else {
        throw 'aria2c is missing and neither WinGet nor Chocolatey is available.'
    }
}

$installDirectory = Join-Path $env:ProgramFiles 'PlexLinker'
$installedBinary = Join-Path $installDirectory 'plexlinker.exe'
New-Item -ItemType Directory -Path $installDirectory -Force | Out-Null
Copy-Item -LiteralPath $source -Destination $installedBinary -Force

$machinePath = [Environment]::GetEnvironmentVariable('Path', 'Machine')
if (($machinePath -split ';') -notcontains $installDirectory) {
    [Environment]::SetEnvironmentVariable('Path', ($machinePath.TrimEnd(';') + ';' + $installDirectory), 'Machine')
}
$env:Path = "$installDirectory;$env:Path"
$env:Path = [Environment]::GetEnvironmentVariable('Path', 'Machine') + ';' + [Environment]::GetEnvironmentVariable('Path', 'User')

if (-not (Get-Command aria2c.exe -ErrorAction SilentlyContinue)) {
    throw 'aria2 was installed but aria2c is not visible on PATH. Open a new terminal and rerun setup.'
}

$configRoot = Join-Path $env:APPDATA 'PlexLinker'
New-Item -ItemType Directory -Path $configRoot -Force | Out-Null
New-Item -ItemType File -Path (Join-Path $configRoot 'setup-complete') -Force | Out-Null

& $installedBinary --version
if ($LASTEXITCODE -ne 0) {
    throw 'The installed PlexLinker binary did not pass its version check.'
}
Write-Host 'Setup complete. Open a new CMD or PowerShell window and run: plexlinker'
if (-not $preconfirmed) {
    Read-Host 'Press Enter to close'
}
