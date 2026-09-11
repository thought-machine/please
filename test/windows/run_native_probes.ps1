<#
.SYNOPSIS
    Runs the Windows release against a test repo, and probes the things Wine cannot show.

.DESCRIPTION
    The unit tests in the bundle cover Please's own code. This covers the release as an
    artifact, and the failure classes docs/design/windows/05-testing-strategy.md lists as
    invisible under Wine: files held open on teardown, and path length.

    Case-insensitivity and the symlink copy fallback are deliberately not here. They belong in
    Go tests in src/fs, where they ride the bundle and are written in the language the fix will
    be written in.
#>
param(
    # A directory holding the release zip.
    [Parameter(Mandatory)][string]$Release,
    [string]$Logs = "$env:RUNNER_TEMP\logs",
    # How many times to build and clean in a row. One build does not find a sharing violation.
    [int]$Rebuilds = 5
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path "$PSScriptRoot\..\..").Path
New-Item -ItemType Directory -Force -Path $Logs | Out-Null
$problems = @()

function Write-Summary([string] $Text) {
    if ($env:GITHUB_STEP_SUMMARY) { Add-Content -Path $env:GITHUB_STEP_SUMMARY -Value $Text }
    else { Write-Host $Text }
}

function Invoke-Plz([string] $WorkDir, [string[]] $PlzArgs, [string] $LogName) {
    $log = Join-Path $Logs $LogName
    Push-Location $WorkDir
    try {
        # Not the repository checkout: .plzconfig_windows_amd64 names MinGW tools that are not
        # on this machine, and a native plz reads it. The fixtures carry their own config.
        $proc = Start-Process -FilePath $script:PleaseExe -ArgumentList $PlzArgs `
            -NoNewWindow -PassThru -RedirectStandardOutput "$log.out" -RedirectStandardError "$log.err"
        $proc.WaitForExit()
        $proc.Refresh()
        Get-Content -LiteralPath "$log.out", "$log.err" -EA SilentlyContinue | Set-Content -LiteralPath $log
        Get-Content -LiteralPath $log | Write-Host
        # Start-Process does not always populate ExitCode until the object is refreshed, and
        # a null here would read as a failure.
        if ($null -ne $proc.ExitCode) { return $proc.ExitCode }
        return 0
    } finally { Pop-Location }
}

# --- the release itself -------------------------------------------------------------------

$zip = Get-ChildItem -Path $Release -Filter 'please_*.zip' | Select-Object -First 1
if (-not $zip) { throw "No please_*.zip in $Release" }
$install = Join-Path $env:RUNNER_TEMP 'install'
if (Test-Path $install) { Remove-Item -Recurse -Force $install }
Expand-Archive -Path $zip.FullName -DestinationPath $install
$script:PleaseExe = Join-Path $install 'please\please.exe'
if (-not (Test-Path $script:PleaseExe)) { throw "No please.exe in $($zip.Name)" }

Write-Host "::group::plz --version"
$version = & $script:PleaseExe --version 2>&1 | Out-String
Write-Host $version
Write-Host '::endgroup::'
Write-Summary "## Windows release`n`n``$($version.Trim())`` from ``$($zip.Name)```n"

# --- a real build, compared byte for byte ---------------------------------------------------

$work = Join-Path $env:RUNNER_TEMP 'smoke'
if (Test-Path $work) { Remove-Item -Recurse -Force $work }
Copy-Item -Recurse (Join-Path $repoRoot 'test\windows\smoke_repo') $work

Write-Host "::group::build //:pipeline"
$code = Invoke-Plz $work @('build', '//:pipeline') 'smoke_build.log'
Write-Host '::endgroup::'
if ($code -ne 0) {
    $problems += "building //:pipeline exited $code"
} else {
    # Compared line by line rather than as bytes: the build action's output is whatever busybox
    # wrote, and the expectation came out of git, so only the content is meant to match.
    $got = Get-Content (Join-Path $work 'plz-out\gen\sorted.txt')
    $want = Get-Content (Join-Path $work 'expected_sorted.txt')
    if (Compare-Object $got $want) {
        $problems += "//:pipeline produced $($got -join ',') rather than $($want -join ',')"
    }
}

# --- files held open on teardown ------------------------------------------------------------

# Windows refuses to delete or rename a file another process has open, and Wine is more
# permissive. This is the single most likely source of real-Windows-only failures, and it hits
# where Please works hardest: plz-out/tmp teardown and RemoveAll. One build never finds it;
# repetition under a live virus scanner sometimes does.
Write-Host "::group::$Rebuilds builds with a clean between each"
for ($i = 1; $i -le $Rebuilds; $i++) {
    $code = Invoke-Plz $work @('clean') "clean_$i.log"
    if ($code -ne 0) { $problems += "plz clean exited $code on run $i" }
    $code = Invoke-Plz $work @('build', '//:pipeline') "rebuild_$i.log"
    if ($code -ne 0) { $problems += "rebuild $i exited $code" }
}
Write-Host '::endgroup::'

# --- long paths -----------------------------------------------------------------------------

# MAX_PATH is 260 unless long-path support is on and the binary opted in by manifest. Go
# prefixes absolute paths with \\?\ by itself, so the interesting failure is not in Please but
# in what it hands busybox as a command line, which gets no such treatment - which is exactly
# the pipe-and-redirect action this fixture builds.
$padding = 'w' * 60
$deep = Join-Path $env:RUNNER_TEMP "long\$padding\$padding\$padding"
if (Test-Path (Join-Path $env:RUNNER_TEMP 'long')) {
    Remove-Item -Recurse -Force (Join-Path $env:RUNNER_TEMP 'long')
}
New-Item -ItemType Directory -Force -Path $deep | Out-Null
$deepRepo = Join-Path $deep 'repo'
Copy-Item -Recurse (Join-Path $repoRoot 'test\windows\smoke_repo') $deepRepo
Write-Host "::group::build at a $($deepRepo.Length)-character path"
$code = Invoke-Plz $deepRepo @('build', '//:pipeline') 'long_path.log'
Write-Host '::endgroup::'
if ($code -ne 0) {
    # Recorded rather than fatal on the first pass: whether this is expected to work depends on
    # LongPathsEnabled, which the workflow prints.
    $problems += "building at a $($deepRepo.Length)-character path exited $code"
}

# --- report ---------------------------------------------------------------------------------

if ($problems.Count -gt 0) {
    Write-Summary "`n### Probe failures`n"
    foreach ($p in $problems) { Write-Summary "- $p" }
    Write-Host "`n$($problems.Count) probe failure(s):"
    foreach ($p in $problems) { Write-Host "  $p" }
    exit 1
}
Write-Summary "`nThe release built a repo, survived $Rebuilds clean-and-rebuild cycles, and built at a long path."
Write-Host "`nAll probes passed."
