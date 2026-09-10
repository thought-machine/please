# The Windows counterpart of pleasew: find or download the Please version this repo asks for,
# then hand over to it. Kept deliberately parallel to that script rather than clever, so the
# two can be read side by side.

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$DefaultUrlBase = 'https://get.please.build'

if ($env:PROCESSOR_ARCHITECTURE -eq 'AMD64' -or $env:PROCESSOR_ARCHITEW6432 -eq 'AMD64') {
    $Arch = 'amd64'
} else {
    Write-Error "Please does not support the $env:PROCESSOR_ARCHITECTURE architecture on Windows."
    exit 1
}
$Os = 'windows'

# Check PLZ_CONFIG_PROFILE, or fall back to a --profile argument.
function Get-Profile {
    if ($env:PLZ_CONFIG_PROFILE) { return $env:PLZ_CONFIG_PROFILE }
    for ($i = 0; $i -lt $args.Count; $i++) {
        if ($args[$i] -like '--profile=*') { return $args[$i].Split('=', 2)[1] }
        if ($args[$i] -eq '--profile' -and $i + 1 -lt $args.Count) { return $args[$i + 1] }
    }
    return ''
}

# Find the repo root by walking up until we see a .plzconfig.
function Find-RepoRoot {
    $dir = Get-Location
    while ($dir) {
        if (Test-Path (Join-Path $dir '.plzconfig')) { return $dir.ToString() }
        $parent = Split-Path -Parent $dir
        if ($parent -eq $dir -or -not $parent) { return '' }
        $dir = $parent
    }
    return ''
}

$Profile_ = Get-Profile @args
$RepoRoot = Find-RepoRoot

# Config files in order of precedence, high to low.
$Configs = @()
if ($RepoRoot) {
    $Configs += Join-Path $RepoRoot '.plzconfig.local'
    if ($Profile_) { $Configs += Join-Path $RepoRoot ".plzconfig.$Profile_" }
    $Configs += Join-Path $RepoRoot ".plzconfig_${Os}_${Arch}"
    $Configs += Join-Path $RepoRoot '.plzconfig'
}
$Configs += Join-Path $env:USERPROFILE '.config\please\plzconfig'
$Configs += Join-Path $env:ProgramData 'please\plzconfig'

# Returns the value of the first key matching the pattern, across the config files in order.
function Read-Config([string] $Pattern) {
    foreach ($config in $Configs) {
        if (-not (Test-Path $config)) { continue }
        $match = Select-String -Path $config -Pattern $Pattern -CaseSensitive:$false | Select-Object -First 1
        if ($match) {
            $parts = $match.Line -split '=', 2
            if ($parts.Count -eq 2) { return $parts[1].Trim() }
        }
    }
    return ''
}

$Location = Read-Config '^\s*location'
if ($Location) {
    # It can contain a literal ~, which nothing on Windows expands for us.
    $Location = $Location -replace '^~', $env:USERPROFILE
} else {
    $Location = Join-Path $env:USERPROFILE '.please'
}

# If Please is already here at any version, let it handle any update itself.
$Target = Join-Path $Location 'please.exe'
if (Test-Path $Target) {
    & $Target @args
    exit $LASTEXITCODE
}

$UrlBase = Read-Config '^\s*downloadlocation'
if (-not $UrlBase) { $UrlBase = $DefaultUrlBase }
$UrlBase = $UrlBase.TrimEnd('/')

$Version = Read-Config '^\s*version[^a-z]'
$Version = $Version -replace '^>=', ''
if (-not $Version) {
    Write-Warning "Can't determine version, will use latest."
    $Version = (Invoke-WebRequest -UseBasicParsing "$UrlBase/latest_version").Content.Trim()
}

$Dir = Join-Path $Location $Version
$Zip = Join-Path ([System.IO.Path]::GetTempPath()) "please_$Version.zip"

Write-Host "Downloading Please $Version to $Dir..." -ForegroundColor Green
if (Test-Path $Dir) { Remove-Item -Recurse -Force $Dir }
New-Item -ItemType Directory -Force -Path $Dir | Out-Null
Invoke-WebRequest -UseBasicParsing "$UrlBase/${Os}_${Arch}/$Version/please_$Version.zip" -OutFile $Zip

# The zip holds everything under a please/ directory, which is the layer the tarball strips
# with --strip-components=1. Expand-Archive has no equivalent, so unpack and move up.
$Staging = Join-Path ([System.IO.Path]::GetTempPath()) "please_$Version"
if (Test-Path $Staging) { Remove-Item -Recurse -Force $Staging }
Expand-Archive -Path $Zip -DestinationPath $Staging
Move-Item (Join-Path $Staging 'please\*') $Dir
Remove-Item -Recurse -Force $Staging, $Zip

# Link it all back up a directory. Symlinks need Developer Mode on Windows, so hard-link
# where we can and copy where we can't; this is the same choice the self-updater makes.
foreach ($file in Get-ChildItem -File $Dir) {
    $link = Join-Path $Location $file.Name
    if (Test-Path $link) { Remove-Item -Force $link }
    try {
        New-Item -ItemType HardLink -Path $link -Target $file.FullName -ErrorAction Stop | Out-Null
    } catch {
        Copy-Item -Force $file.FullName $link
    }
}

Write-Host 'Should be good to go now, running plz...' -ForegroundColor Green
& $Target @args
exit $LASTEXITCODE
