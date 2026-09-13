<#
.SYNOPSIS
    Replays the published codelabs on Windows, and records what a person following them would hit.

.DESCRIPTION
    //test/windows:codelab_plan reduces docs/codelabs/*.md to an ordered list of steps: files to
    write, commands to run, directories to change into. This replays that list with the Windows
    release, the way a person reading https://please.build/codelabs.html on Windows would.

    It answers a different question from run_native_tests.ps1 and run_native_probes.ps1. Those ask
    whether Please works on Windows. This asks whether the documentation does, and most of what it
    finds is not a bug in Please: bash syntax PowerShell does not accept, Unix tools that are not
    there, and plugins whose tools have no Windows release. That is the finding, and it is recorded
    rather than worked around. Nothing here edits a codelab to make it pass.

    Each command is handed to pwsh exactly as the codelab writes it, as an encoded command so that
    no quoting of this script's stands between the text and the parser. The shell is the subject
    under test, not plumbing: a reader types these lines into PowerShell, so that is where they run.

    Every step ends as one of:

      PASS     it exited zero, and contained the sidecar's assert text if there was one
      FAIL     it did not
      KNOWN    it failed, and codelab_known_failures.txt says so, with a reason
      SKIPPED  the sidecar says it cannot run here, or it needs a tool this machine lacks
      BLOCKED  an earlier step in the same codelab failed, so this one was never reached

    BLOCKED is never counted as a failure. Without it one missing plugin tool in go_intro would
    manufacture a dozen more failures, and the one entry that matters would drown.

    What a codelab shows a command printing is compared and reported, but never fails a step. The
    codelabs' output is full of timings and a randomly chosen greeting, and asserting on it would
    produce flakes that discredit the whole check.

.EXAMPLE
    # On Linux, before pushing: parses the plan, resolves the known failures, prints what would run.
    pwsh ./test/windows/run_codelabs.ps1 -DryRun -Plan plz-out/gen/test/windows/codelab_plan.json `
        -KnownFailures test/windows/codelab_known_failures.txt
#>
param(
    # codelab_plan.json, as built by //test/windows:codelab_plan.
    [Parameter(Mandatory)][string]$Plan,
    # A directory holding the release zip. Not needed for -DryRun.
    [string]$Release = '',
    [string]$Logs = '',
    # One "codelab_id" or "codelab_id::step-key" per line, with a comment above each saying why.
    [string]$KnownFailures = '',
    # Run only these codelabs. For reproducing one locally.
    [string[]]$Only = @(),
    # Per command, unless the sidecar gives one. A plugin download and a Go toolchain can both
    # land in a single step.
    [int]$TimeoutSeconds = 600,
    # Execute nothing: check the bookkeeping and print the plan.
    [switch]$DryRun
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# The codelabs chain commands with &&, which Windows PowerShell 5.1 rejects outright. A person on
# Windows today has pwsh 7, and so does windows-latest; running under 5.1 would report findings
# about a shell nobody should be using.
if ($PSVersionTable.PSVersion.Major -lt 7) {
    throw "run_codelabs.ps1 needs PowerShell 7 (pwsh); this is $($PSVersionTable.PSVersion)"
}

# Taken once, up front: each codelab points TEMP somewhere of its own, and GetTempPath follows it.
$temp = if ($env:RUNNER_TEMP) { $env:RUNNER_TEMP } else { [IO.Path]::GetTempPath() }
if (-not $Logs) { $Logs = Join-Path $temp 'logs' }
New-Item -ItemType Directory -Force -Path $Logs | Out-Null
$problems = [Collections.Generic.List[string]]::new()

function Write-Summary([string] $Text) {
    if ($env:GITHUB_STEP_SUMMARY) { Add-Content -Path $env:GITHUB_STEP_SUMMARY -Value $Text }
    else { Write-Host $Text }
}

# The same format, and the same rules, as run_native_tests.ps1's.
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

function Get-LogName([string] $Key) { return ($Key -replace '[^A-Za-z0-9_-]+', '_') + '.log' }

# --- what a step needs, decided on this machine ----------------------------------------------

# Presence alone is not the question. windows-latest has a docker, but it runs Windows containers,
# and every image in the codelabs is Linux; and it has a kubectl, with no cluster behind it. Either
# would pass a Get-Command check and then fail for a reason that says nothing about the codelab.
$needCache = @{}
function Test-Need([string] $Need) {
    if ($needCache.ContainsKey($Need)) { return $needCache[$Need] }
    $result = switch ($Need) {
        'docker' {
            if (-not (Get-Command docker -EA SilentlyContinue)) { 'docker is not installed' }
            else {
                $os = (& docker info --format '{{.OSType}}' 2>$null | Out-String).Trim()
                if ($os -ne 'linux') { "docker runs $(if ($os) { $os } else { 'no' }) containers, and the codelab's images are Linux" }
                else { '' }
            }
        }
        'kubectl' {
            if (-not (Get-Command kubectl -EA SilentlyContinue)) { 'kubectl is not installed' }
            else {
                & kubectl cluster-info --request-timeout=5s *> $null
                if ($LASTEXITCODE -ne 0) { 'kubectl has no cluster to talk to' } else { '' }
            }
        }
        default { $null }
    }
    $needCache[$Need] = $result
    return $result
}

# --- running one command --------------------------------------------------------------------

function Invoke-Command-Step($Step, [string] $WorkDir, [string] $LogPath) {
    $timeout = if ($Step.PSObject.Properties['timeout'] -and $Step.timeout) { $Step.timeout } else { $TimeoutSeconds }
    $encoded = [Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes($Step.command))
    $proc = Start-Process -FilePath $script:Pwsh `
        -ArgumentList @('-NoProfile', '-NonInteractive', '-OutputFormat', 'Text', '-EncodedCommand', $encoded) `
        -WorkingDirectory $WorkDir -NoNewWindow -PassThru `
        -RedirectStandardOutput "$LogPath.out" -RedirectStandardError "$LogPath.err" `
        -RedirectStandardInput $script:EmptyInput
    $timedOut = -not $proc.WaitForExit($timeout * 1000)
    if ($timedOut) {
        # The whole tree: a plz that has started a build action leaves children behind, and a
        # child holding plz-out open stops the next codelab's directory being removed.
        try { $proc.Kill($true) } catch { }
        $proc.WaitForExit()
    }
    # ExitCode is not always populated until the object is refreshed, and a null would read as a
    # pass. See run_native_probes.ps1.
    $proc.Refresh()
    $output = @(Get-Content -LiteralPath "$LogPath.out", "$LogPath.err" -EA SilentlyContinue)
    Set-Content -LiteralPath $LogPath -Value (@("> $($Step.command)", "  in $WorkDir", '') + $output)
    Remove-Item -LiteralPath "$LogPath.out", "$LogPath.err" -EA SilentlyContinue
    return [pscustomobject]@{
        ExitCode = if ($timedOut) { $null } elseif ($null -ne $proc.ExitCode) { $proc.ExitCode } else { 0 }
        TimedOut = $timedOut
        Timeout  = $timeout
        Output   = $output
    }
}

# Merges a .plzconfig fragment the way a reader following the codelab edits the file: a key the
# section already has is replaced, a new key goes into its section, and a new section is appended.
#
# Appending the fragment verbatim was tried first, and it manufactured a failure on the first
# native run: plz init plugin go already writes GoTool, the codelab's fragment sets it again, and a
# plugin section refuses a repeated key where core config quietly takes the last one. A key repeated
# on purpose to extend a list would be replaced here rather than added to; no codelab fragment has
# one.
function Merge-PlzConfig([string] $Existing, [string] $Fragment) {
    $lines = [Collections.Generic.List[string]]::new()
    if ($Existing) { $lines.AddRange([string[]]($Existing.TrimEnd("`r", "`n") -split "`r?`n")) }

    # Section names are case-insensitive in this format; a subsection's quoted name is not.
    function Get-SectionKey([string] $Header) {
        if ($Header -notmatch '^\s*\[\s*([^\s"\]]+)\s*(?:"([^"]*)")?\s*\]') { return $null }
        return "$($Matches[1].ToLowerInvariant())|$($Matches[2])"
    }
    # Where a section starts, and the index after its last non-blank line.
    function Find-Section([string] $Key) {
        for ($i = 0; $i -lt $lines.Count; $i++) {
            # -cne: PowerShell compares case-insensitively by default, and the subsection half
            # of the key must not be.
            if ((Get-SectionKey $lines[$i]) -cne $Key) { continue }
            $end = $i + 1
            for ($j = $i + 1; $j -lt $lines.Count -and -not $lines[$j].TrimStart().StartsWith('['); $j++) {
                if ($lines[$j].Trim()) { $end = $j + 1 }
            }
            return @($i, $end)
        }
        return $null
    }

    $section = $null
    foreach ($raw in ($Fragment -split "`r?`n")) {
        $line = $raw.Trim()
        if (-not $line -or $line.StartsWith(';') -or $line.StartsWith('#')) { continue }
        $key = Get-SectionKey $line
        if ($key) {
            $section = $key
            if (-not (Find-Section $key)) {
                if ($lines.Count -gt 0 -and $lines[$lines.Count - 1].Trim()) { $lines.Add('') }
                $lines.Add($line)
            }
            continue
        }
        if (-not $section -or $line -notmatch '^([^=;#]+?)\s*=') { continue }
        $name = $Matches[1].Trim()
        $start, $end = Find-Section $section
        $replaced = $false
        for ($i = $start + 1; $i -lt $end; $i++) {
            if ($lines[$i] -match '^\s*([^=;#]+?)\s*=' -and $Matches[1].Trim() -ieq $name) {
                $lines[$i] = $line
                $replaced = $true
                break
            }
        }
        if (-not $replaced) { $lines.Insert($end, $line) }
    }
    return ($lines -join "`n") + "`n"
}

# How much of what the codelab shows this command printing actually appeared. Advisory only.
function Compare-Expected($Step, $Output) {
    if (-not $Step.PSObject.Properties['expected_output'] -or -not $Step.expected_output) { return '' }
    $got = ($Output | ForEach-Object { $_.Trim() }) -join "`n"
    $missing = @($Step.expected_output | Where-Object { $_.Trim() -and -not $got.Contains($_.Trim()) })
    if ($missing.Count -eq 0) { return '' }
    return "$($missing.Count) of $(@($Step.expected_output).Count) lines the codelab shows did not appear (advisory)"
}

# --- setup ----------------------------------------------------------------------------------

$planDoc = Get-Content -Raw -LiteralPath $Plan | ConvertFrom-Json
$known = Read-KnownFailures $KnownFailures

# An entry naming nothing is also caught on Linux, by //test/windows/codelab_script/script:script_test.
# Checked again here so that a plan and a failures list from different commits cannot pass quietly.
$planNames = @{}
foreach ($c in $planDoc.codelabs) {
    $planNames[$c.id] = $true
    foreach ($s in $c.steps) { $planNames[$s.key] = $true }
}
foreach ($k in $known.Keys) {
    if (-not $planNames.ContainsKey($k)) {
        $problems.Add("$k is in $KnownFailures but names nothing in the plan")
    }
}

if (-not $DryRun) {
    if (-not $Release) { throw '-Release is required unless -DryRun is given' }
    $zip = Get-ChildItem -Path $Release -Filter 'please_*.zip' | Select-Object -First 1
    if (-not $zip) { throw "No please_*.zip in $Release" }
    $install = Join-Path $temp 'codelab-install'
    if (Test-Path $install) { Remove-Item -Recurse -Force $install }
    Expand-Archive -Path $zip.FullName -DestinationPath $install
    $pleaseDir = Join-Path $install 'please'
    if (-not (Test-Path (Join-Path $pleaseDir 'plz.cmd'))) { throw "No plz.cmd in $($zip.Name)" }

    # On the PATH, not invoked by path. The codelabs say `plz`, package/Install.md tells a Windows
    # user to put this directory on their PATH, and doing the same here is also the only thing
    # anywhere that runs plz.cmd natively. If it mangles arguments, that is a finding.
    $env:PATH = "$pleaseDir$([IO.Path]::PathSeparator)$env:PATH"
    $script:Pwsh = (Get-Process -Id $PID).Path
    $script:EmptyInput = Join-Path $temp 'codelab-empty-stdin'
    Set-Content -LiteralPath $script:EmptyInput -Value $null -NoNewline

    # Through the PATH, as every codelab step will be. If plz.cmd cannot even report a version,
    # say so in one line rather than as a stack trace, and let the codelabs show how far it gets.
    Write-Host '::group::plz --version'
    try {
        & plz --version 2>&1 | Write-Host
        if ($LASTEXITCODE -ne 0) { $problems.Add("plz --version exited $LASTEXITCODE through the PATH") }
    } catch {
        $problems.Add("plz --version could not run through the PATH: $_")
    }
    Write-Host '::endgroup::'
}

$utf8 = [Text.UTF8Encoding]::new($false)
$rows = [Collections.Generic.List[object]]::new()
$details = [Collections.Generic.List[string]]::new()

# --- the codelabs ---------------------------------------------------------------------------

foreach ($codelab in $planDoc.codelabs) {
    if ($Only.Count -gt 0 -and $codelab.id -notin $Only) { continue }
    $counts = [ordered]@{ PASS = 0; FAIL = 0; KNOWN = 0; SKIPPED = 0; BLOCKED = 0 }
    $total = if ($codelab.blocks.PSObject.Properties['total']) { $codelab.blocks.total } else { 0 }

    if ($codelab.PSObject.Properties['not_runnable'] -and $codelab.not_runnable) {
        $rows.Add([pscustomobject]@{ Id = $codelab.id; Blocks = $total; Steps = 0; Counts = $counts; Note = "not runnable: $($codelab.not_runnable)" })
        continue
    }

    Write-Host "::group::$($codelab.id) - $($codelab.title)"
    $root = Join-Path $temp "codelabs\$($codelab.id)"
    $home_ = Join-Path $temp "codelabs\$($codelab.id)-home"

    if (-not $DryRun) {
        foreach ($d in $root, $home_) {
            if (Test-Path $d) { Remove-Item -Recurse -Force $d }
        }
        New-Item -ItemType Directory -Force -Path $root, "$home_\AppData\Local", "$home_\Temp" | Out-Null
        # A home of its own, beside the working tree rather than in it, so ~/.please and the
        # caches land somewhere a `tree -a` does not see. LOCALAPPDATA is not optional: Please's
        # content-addressed cache has produced false passes twice in this port, and here it would
        # let one codelab replay what an earlier one built. Nothing above $root holds a .plzconfig,
        # which is what keeps plz init from stopping to ask whether to continue.
        $env:HOME = $home_
        $env:USERPROFILE = $home_
        $env:LOCALAPPDATA = "$home_\AppData\Local"
        $env:TEMP = "$home_\Temp"
        $env:TMP = "$home_\Temp"
    }

    $cwd = $root
    $blockedBy = ''
    $ran = 0

    foreach ($step in $codelab.steps) {
        $key = $step.key
        $log = Join-Path $Logs (Get-LogName $key)
        $outcome = ''
        $note = ''

        if ($step.kind -eq 'skip') {
            $outcome = 'SKIPPED'
            $note = "$($step.reason): $($step.detail)"
        } elseif ($blockedBy) {
            $outcome = 'BLOCKED'
            $note = "after $blockedBy"
        } elseif ($DryRun) {
            $what = switch ($step.kind) {
                'run' { $step.command }
                'file' { "$($step.mode) $($step.path)" }
                'chdir' { "cd $($step.dir)" }
            }
            Write-Host ("  {0,-60} {1,-5} {2}" -f $key, $step.kind, $what)
            if ($step.kind -eq 'run') { $ran++ }
            continue
        } else {
            switch ($step.kind) {
                'chdir' {
                    $target = Join-Path $cwd $step.dir
                    if (Test-Path -LiteralPath $target -PathType Container) {
                        $cwd = (Resolve-Path -LiteralPath $target).Path
                        $outcome = 'PASS'
                    } else {
                        # A reader whose earlier step did not create it is stuck here too.
                        $outcome = 'FAIL'
                        $note = "no directory $($step.dir) in $cwd"
                    }
                }
                'file' {
                    $path = Join-Path $cwd $step.path
                    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $path) | Out-Null
                    # Not Set-Content: its encoding differs between PowerShell versions, and a
                    # .plzconfig carrying a byte-order mark does not parse.
                    if ($step.mode -eq 'merge') {
                        $existing = if (Test-Path -LiteralPath $path) { [IO.File]::ReadAllText($path) } else { '' }
                        [IO.File]::WriteAllText($path, (Merge-PlzConfig $existing $step.content), $utf8)
                    } else {
                        [IO.File]::WriteAllText($path, "$($step.content)`n", $utf8)
                    }
                    $outcome = 'PASS'
                }
                'run' {
                    $missing = @()
                    if ($step.PSObject.Properties['needs'] -and $step.needs) {
                        foreach ($need in $step.needs) {
                            $why = Test-Need $need
                            if ($null -eq $why) {
                                $problems.Add("${key}: unknown need '$need'; run_codelabs.ps1 has no way to check for it")
                                $missing += "unknown need $need"
                            } elseif ($why) {
                                $missing += $why
                            }
                        }
                    }
                    if ($missing.Count -gt 0) {
                        $outcome = 'SKIPPED'
                        $note = "tool-missing: $($missing -join '; ')"
                        break
                    }
                    Write-Host "> $($step.command)"
                    $ran++
                    $r = Invoke-Command-Step $step $cwd $log
                    $r.Output | Select-Object -Last 40 | Write-Host
                    $assert = if ($step.PSObject.Properties['assert']) { $step.assert } else { '' }
                    if ($r.TimedOut) {
                        $outcome = 'FAIL'
                        $note = "timed out after $($r.Timeout)s"
                    } elseif ($r.ExitCode -ne 0) {
                        $outcome = 'FAIL'
                        $note = "exited $($r.ExitCode)"
                    } elseif ($assert -and -not (($r.Output -join "`n").Contains($assert))) {
                        $outcome = 'FAIL'
                        $note = "exited 0 but did not print '$assert'"
                    } else {
                        $outcome = 'PASS'
                        $note = Compare-Expected $step $r.Output
                    }
                }
            }
        }

        if ($outcome -eq 'FAIL' -and ($known.ContainsKey($key) -or $known.ContainsKey($codelab.id))) {
            $outcome = 'KNOWN'
        }
        if ($outcome -eq 'PASS' -and $known.ContainsKey($key)) {
            $problems.Add("$key is in $KnownFailures but passed; remove it")
        }
        $nonBlocking = $step.PSObject.Properties['non_blocking'] -and $step.non_blocking
        if ($outcome -in 'FAIL', 'KNOWN' -and -not $nonBlocking) {
            $blockedBy = $key
        }
        if ($outcome -eq 'FAIL') {
            $problems.Add("$key $note")
        }

        $counts[$outcome]++
        if ($outcome -ne 'PASS' -or $note) {
            $details.Add("$outcome $key$(if ($note) { " - $note" })")
        }
        Write-Host "$outcome $key$(if ($note) { " - $note" })"
    }

    # The analogue of run_native_tests.ps1's "the bundle produced no runnable tests". A codelab that
    # ran nothing and was not declared unrunnable has had its commands lost somewhere between the
    # Markdown and here, and reporting it as a clean pass would be the worst possible answer.
    if ($ran -eq 0 -and $counts.BLOCKED -eq 0 -and $counts.SKIPPED -eq 0) {
        $problems.Add("$($codelab.id) ran no commands, and codelab_steps.conf does not say it has none to run")
    }
    if ($known.ContainsKey($codelab.id) -and $counts.KNOWN -eq 0 -and -not $DryRun) {
        $problems.Add("$($codelab.id) is in $KnownFailures but nothing in it failed; remove it")
    }
    Write-Host '::endgroup::'
    $rows.Add([pscustomobject]@{ Id = $codelab.id; Blocks = $total; Steps = @($codelab.steps).Count; Counts = $counts; Note = '' })
}

# --- report ---------------------------------------------------------------------------------

Write-Summary "## Codelabs on Windows`n"
if ($DryRun) { Write-Summary "Dry run: nothing was executed.`n" }
Write-Summary '| Codelab | Blocks | Steps | Passed | Failed | Known | Skipped | Blocked | |'
Write-Summary '|---|---:|---:|---:|---:|---:|---:|---:|---|'
foreach ($r in $rows) {
    $c = $r.Counts
    Write-Summary "| $($r.Id) | $($r.Blocks) | $($r.Steps) | $($c.PASS) | $($c.FAIL) | $($c.KNOWN) | $($c.SKIPPED) | $($c.BLOCKED) | $($r.Note) |"
}
if ($details.Count -gt 0) {
    # Printed as the step key first so a line can go straight into codelab_known_failures.txt.
    Write-Summary "`n<details><summary>Every step that did not simply pass</summary>`n"
    Write-Summary '```'
    foreach ($d in $details) { Write-Summary $d }
    Write-Summary '```'
    Write-Summary '</details>'
}

if ($problems.Count -gt 0) {
    Write-Summary "`n### Problems`n"
    foreach ($p in $problems) { Write-Summary "- $p" }
    Write-Host "`n$($problems.Count) problem(s):"
    foreach ($p in $problems) { Write-Host "  $p" }
    exit 1
}
Write-Host "`nNo unexpected results."
