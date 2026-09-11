<#
.SYNOPSIS
    Runs the cross-built Windows test binaries natively, out of the bundle.

.DESCRIPTION
    The counterpart of wine_go_test for a real Windows machine. //test/windows:native_test_bundle
    packages the same test binaries and the same data that run under Wine on Linux; this runs
    them here, where the answers actually count. See docs/design/windows/05-testing-strategy.md
    for what Wine cannot show and why that matters.

    Everything this sets up mirrors what wine_go_test sets up, minus Wine: the test's own
    directory as the working directory, $DATA pointing at its data, and the bundled busybox on
    the PATH for the tests that run build actions.
#>
param(
    # The extracted bundle: manifest.txt, shell/, tests/<name>/.
    [Parameter(Mandatory)][string]$Bundle,
    [string]$Logs = "$env:RUNNER_TEMP\logs",
    # One "name" or "name::TestCase" per line, with a comment above each saying why. A known
    # failure that starts passing is also a failure, which is what stops this becoming a
    # dumping ground.
    [string]$KnownFailures = '',
    # Run only these entries. For reproducing one failure locally.
    [string[]]$Only = @(),
    [int]$TimeoutSeconds = 600
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Read-KnownFailures([string] $Path) {
    $known = @{}
    if (-not $Path -or -not (Test-Path $Path)) { return $known }
    foreach ($line in Get-Content $Path) {
        $trimmed = $line.Trim()
        if (-not $trimmed -or $trimmed.StartsWith('#')) { continue }
        $known[$trimmed] = $true
    }
    return $known
}

# Go's -test.v marks each case with a line like "--- FAIL: TestFoo (0.05s)". Subtests come
# through the same way, indented, which is why the pattern allows leading whitespace.
function Get-Cases([string] $Path) {
    $cases = @()
    foreach ($line in Get-Content -LiteralPath $Path -ErrorAction SilentlyContinue) {
        if ($line -match '^\s*--- (PASS|FAIL|SKIP): (\S+)') {
            $cases += [pscustomobject]@{ Result = $Matches[1]; Name = $Matches[2] }
        }
    }
    return $cases
}

function Write-Summary([string] $Text) {
    if ($env:GITHUB_STEP_SUMMARY) { Add-Content -Path $env:GITHUB_STEP_SUMMARY -Value $Text }
    else { Write-Host $Text }
}

$bundleDir = (Resolve-Path $Bundle).Path
$manifest = Join-Path $bundleDir 'manifest.txt'
if (-not (Test-Path $manifest)) { throw "No manifest.txt in $bundleDir; is that the bundle?" }
New-Item -ItemType Directory -Force -Path $Logs | Out-Null

$known = Read-KnownFailures $KnownFailures
$names = Get-Content $manifest | Where-Object { $_.Trim() }
if ($Only.Count -gt 0) { $names = $names | Where-Object { $Only -contains $_ } }

$rows = @()
$problems = @()

foreach ($name in $names) {
    $dir = Join-Path $bundleDir "tests\$name"
    if (-not (Test-Path $dir)) {
        # The manifest is written at parse time and the directories at build time, so this
        # means a test was dropped between the two rather than that it failed.
        $problems += "$name is in manifest.txt but has no directory in the bundle"
        continue
    }

    Write-Host "::group::$name"
    Push-Location $dir
    try {
        # Please rewrites every backslash in every environment value on Windows
        # (BuildEnv.normalisePathSeparators), so hand these over already normalised. Getting it
        # wrong produces failures that look like port bugs and are not.
        $here = $dir -replace '\\', '/'
        foreach ($v in 'TEST_DIR', 'TMP_DIR', 'TMPDIR', 'HOME', 'USERPROFILE', 'TEMP', 'TMP') {
            Set-Item -Path "env:$v" -Value $here
        }
        # Not something Please itself redirects, but without it a test that runs a build shares
        # the machine's directory cache, and a cached artifact from another run is exactly how
        # this port has produced a false pass before.
        $env:LOCALAPPDATA = $here

        $dataFile = Join-Path $dir 'DATA.txt'
        $env:DATA = if (Test-Path $dataFile) { (Get-Content -Raw $dataFile).Trim() } else { '' }

        if (Test-Path (Join-Path $dir 'NEEDS_SHELL')) {
            # Its own directory on the PATH rather than the working directory, which is how an
            # install has it and what Go's exec will agree to run.
            $env:PATH = (Join-Path $bundleDir 'shell') + [IO.Path]::PathSeparator + $env:PATH
        }

        $out = Join-Path $Logs "$name.out"
        $err = Join-Path $Logs "$name.err"
        $proc = Start-Process -FilePath (Join-Path $dir 'test.exe') `
            -ArgumentList '-test.v' -NoNewWindow -PassThru `
            -RedirectStandardOutput $out -RedirectStandardError $err
        if (-not $proc.WaitForExit($TimeoutSeconds * 1000)) {
            # A hung test would otherwise hold the job open for hours. Kill the tree: these
            # binaries start children of their own.
            Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
            $problems += "$name timed out after ${TimeoutSeconds}s"
            $rows += [pscustomobject]@{ Test = $name; Pass = 0; Fail = 0; Skip = 0; Status = 'TIMEOUT' }
            continue
        }
        # Start-Process does not always populate ExitCode until the object is refreshed, and
        # a null here would read as non-zero and fail a test that passed.
        $proc.Refresh()
        $code = if ($null -ne $proc.ExitCode) { $proc.ExitCode } else { 0 }

        # Panics go to stderr and belong with the output they interrupted.
        $log = Join-Path $Logs "$name.log"
        Get-Content -LiteralPath $out, $err -ErrorAction SilentlyContinue | Set-Content -LiteralPath $log
        Get-Content -LiteralPath $log | Write-Host

        $cases = Get-Cases $log
        $failed = @($cases | Where-Object { $_.Result -eq 'FAIL' })
        $passed = @($cases | Where-Object { $_.Result -eq 'PASS' })
        $skipped = @($cases | Where-Object { $_.Result -eq 'SKIP' })

        foreach ($case in $failed) {
            $key = "$name::$($case.Name)"
            if ($known.ContainsKey($key) -or $known.ContainsKey($name)) { continue }
            $problems += $key
            Write-Host "::error title=$name::$($case.Name) failed"
        }
        foreach ($case in $passed) {
            $key = "$name::$($case.Name)"
            if ($known.ContainsKey($key)) {
                $problems += "$key is in $KnownFailures but passed; remove it"
            }
        }
        # A binary that dies without reporting a single case - a panic in TestMain, a missing
        # DLL - would otherwise look like a clean run with nothing in it.
        if ($code -ne 0 -and $failed.Count -eq 0 -and -not $known.ContainsKey($name)) {
            $problems += "$name exited $code with no failing case; see $name.log"
            Write-Host "::error title=$name::exited $code without reporting a failure"
        }

        $status = if ($failed.Count -gt 0 -or $code -ne 0) { 'FAIL' } else { 'ok' }
        $rows += [pscustomobject]@{
            Test = $name; Pass = $passed.Count; Fail = $failed.Count
            Skip = $skipped.Count; Status = $status
        }
    } finally {
        Pop-Location
        Write-Host '::endgroup::'
    }
}

Write-Summary "## Windows unit tests`n"
Write-Summary '| Test | Pass | Fail | Skip | |'
Write-Summary '|---|---:|---:|---:|---|'
foreach ($row in $rows) {
    Write-Summary "| $($row.Test) | $($row.Pass) | $($row.Fail) | $($row.Skip) | $($row.Status) |"
}
if ($rows.Count -gt 0) {
    $totals = $rows | Measure-Object -Property Pass, Fail, Skip -Sum
    Write-Summary "`n$($rows.Count) binaries, $($totals[0].Sum) passed, $($totals[1].Sum) failed, $($totals[2].Sum) skipped."
} else {
    Write-Summary "`nNo test binaries ran at all."
    $problems += 'the bundle produced no runnable tests'
}

if ($problems.Count -gt 0) {
    Write-Summary "`n### Unexpected`n"
    foreach ($p in $problems) { Write-Summary "- $p" }
    Write-Host "`n$($problems.Count) unexpected result(s):"
    foreach ($p in $problems) { Write-Host "  $p" }
    exit 1
}
Write-Host "`nAll $($rows.Count) test binaries behaved as expected."
