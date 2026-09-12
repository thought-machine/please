<#
.SYNOPSIS
    Downloads a precompiled copy of Please and installs it.

.DESCRIPTION
    The Windows counterpart of get_plz.sh, served from the same bucket and run the same way:

        irm https://get.please.build/get_plz.ps1 | iex

    Kept deliberately parallel to that script rather than clever, so the two can be read side by
    side. The differences are all forced: the release is a .zip rather than a tarball, because
    Windows has no guaranteed tar; the short name is a plz.cmd shim rather than a symlink,
    because symlinks need Developer Mode; and the install is linked up a level by hard-linking
    or copying, for the same reason.
#>

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$UrlBase = 'https://get.please.build'

if ($env:PROCESSOR_ARCHITECTURE -eq 'AMD64' -or $env:PROCESSOR_ARCHITEW6432 -eq 'AMD64') {
    $Arch = 'amd64'
} else {
    Write-Error "Please does not support the $env:PROCESSOR_ARCHITECTURE architecture on Windows."
    exit 1
}

$Version = (Invoke-WebRequest -UseBasicParsing "$UrlBase/latest_version").Content.Trim()
$Location = Join-Path $env:USERPROFILE '.please'
$Dir = Join-Path $Location $Version
$Zip = Join-Path ([System.IO.Path]::GetTempPath()) "please_$Version.zip"

Write-Host "Downloading Please $Version..." -ForegroundColor Green
if (Test-Path $Dir) { Remove-Item -Recurse -Force $Dir }
New-Item -ItemType Directory -Force -Path $Dir | Out-Null
Invoke-WebRequest -UseBasicParsing "$UrlBase/windows_${Arch}/$Version/please_$Version.zip" -OutFile $Zip

# The zip holds everything under a please/ directory, which is the layer the tarball strips with
# --strip-components=1. Expand-Archive has no equivalent, so unpack and move up.
$Staging = Join-Path ([System.IO.Path]::GetTempPath()) "please_$Version"
if (Test-Path $Staging) { Remove-Item -Recurse -Force $Staging }
Expand-Archive -Path $Zip -DestinationPath $Staging
Move-Item (Join-Path $Staging 'please\*') $Dir
Remove-Item -Recurse -Force $Staging, $Zip

# Link it all back up a directory. Symlinks need Developer Mode on Windows, so hard-link where
# we can and copy where we can't; this is the same choice the self-updater and pleasew.ps1 make.
foreach ($file in Get-ChildItem -File $Dir) {
    $link = Join-Path $Location $file.Name
    if (Test-Path $link) { Remove-Item -Force $link }
    try {
        New-Item -ItemType HardLink -Path $link -Target $file.FullName -ErrorAction Stop | Out-Null
    } catch {
        Copy-Item -Force $file.FullName $link
    }
}

Write-Host "Please installed to $Location" -ForegroundColor Green
Write-Host "Add it to your PATH to use plz from anywhere:"
Write-Host "    [Environment]::SetEnvironmentVariable('Path', `"`$env:Path;$Location`", 'User')"
