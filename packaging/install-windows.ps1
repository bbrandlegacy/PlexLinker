param(
    [Parameter(Mandatory = $true)]
    [string]$Binary,
    [string]$InstallDirectory = "$env:LOCALAPPDATA\Programs\PlexLinker"
)

$ErrorActionPreference = 'Stop'

if (-not (Test-Path -LiteralPath $Binary -PathType Leaf)) {
    throw "PlexLinker binary not found: $Binary"
}

if (-not (Get-Command aria2c.exe -ErrorAction SilentlyContinue)) {
    throw 'aria2c.exe must be installed and available on PATH before installing PlexLinker.'
}

New-Item -ItemType Directory -Force -Path $InstallDirectory | Out-Null
$destination = Join-Path $InstallDirectory 'plexlinker.exe'
Copy-Item -LiteralPath $Binary -Destination $destination -Force

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$pathEntries = @($userPath -split ';' | Where-Object { $_ })
if ($pathEntries -notcontains $InstallDirectory) {
    $updatedPath = (@($pathEntries) + $InstallDirectory) -join ';'
    [Environment]::SetEnvironmentVariable('Path', $updatedPath, 'User')
}

Write-Host "Installed PlexLinker to $destination" -ForegroundColor Green
Write-Host 'Open a new shell and run: plexlinker.exe'
