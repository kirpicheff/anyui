param(
    [int]$WarmupSec = 5,
    [int]$SampleSec = 5
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path

function Get-ProcessTreeIds {
    param([int]$RootPid)
    $all = Get-CimInstance Win32_Process | Select-Object ProcessId, ParentProcessId
    $result = New-Object System.Collections.Generic.HashSet[int]
    $queue = New-Object System.Collections.Generic.Queue[int]
    [void]$result.Add($RootPid)
    $queue.Enqueue($RootPid)

    while ($queue.Count -gt 0) {
        $curr = $queue.Dequeue()
        foreach ($p in $all) {
            if ($p.ParentProcessId -eq $curr -and -not $result.Contains([int]$p.ProcessId)) {
                [void]$result.Add([int]$p.ProcessId)
                $queue.Enqueue([int]$p.ProcessId)
            }
        }
    }
    return ,([int[]]$result)
}

function Get-TreeMetrics {
    param([int]$RootPid)
    $pids = Get-ProcessTreeIds -RootPid $RootPid
    $wsBytes = [int64]0
    $privBytes = [int64]0
    $cpuSec = [double]0
    $count = 0

    foreach ($id in $pids) {
        try {
            $proc = Get-Process -Id $id -ErrorAction Stop
            $wsBytes += $proc.WorkingSet64
            $privBytes += $proc.PrivateMemorySize64
            $cpuSec += $proc.TotalProcessorTime.TotalSeconds
            $count++
        } catch {
            # Process may have exited
        }
    }

    return [PSCustomObject]@{
        ProcessCount = $count
        WorkingSetMB = [math]::Round($wsBytes / 1MB, 2)
        PrivateMB    = [math]::Round($privBytes / 1MB, 2)
        CpuSec       = $cpuSec
    }
}

$targets = @(
    @{ Name = "AnyUI"; Dir = "anyui"; CGO = "0"; Tags = "" },
    @{ Name = "Gio";   Dir = "gio";   CGO = "0"; Tags = "" },
    @{ Name = "Fyne";  Dir = "fyne";  CGO = "1"; Tags = "" },
    @{ Name = "Wails"; Dir = "wails"; CGO = "0"; Tags = "desktop,production" }
)

$cores = [Environment]::ProcessorCount
$results = @()

foreach ($t in $targets) {
    $dir = Join-Path $root $t.Dir
    $exe = Join-Path $dir "bench_$($t.Dir).exe"
    $startupFile = Join-Path $dir "startup.txt"
    if (Test-Path $startupFile) { Remove-Item $startupFile -Force }

    Write-Host "=== Building $($t.Name) (CGO=$($t.CGO)) ===" -ForegroundColor Cyan

    Push-Location $dir
    try {
        $env:CGO_ENABLED = $t.CGO
        $buildArgs = @("build", "-trimpath", "-ldflags=-s -w -H windowsgui")
        if ($t.Tags -ne "") {
            $buildArgs += @("-tags", $t.Tags)
        }
        $buildArgs += @("-o", $exe, ".")
        & go @buildArgs
        if ($LASTEXITCODE -ne 0) {
            throw "Build failed for $($t.Name)"
        }
    } finally {
        Pop-Location
    }

    $sizeMB = [math]::Round((Get-Item $exe).Length / 1MB, 2)
    Write-Host "Built $exe ($sizeMB MB). Running for $($WarmupSec + $SampleSec)s..."

    $env:BENCH_STARTUP_FILE = $startupFile
    $proc = Start-Process -FilePath $exe -PassThru
    try {
        Start-Sleep -Seconds $WarmupSec
        $m1 = Get-TreeMetrics -RootPid $proc.Id
        $sw = [System.Diagnostics.Stopwatch]::StartNew()
        Start-Sleep -Seconds $SampleSec
        $sw.Stop()
        $m2 = Get-TreeMetrics -RootPid $proc.Id

        $cpuDelta = [math]::Max(0, $m2.CpuSec - $m1.CpuSec)
        $idleCpuPct = [math]::Round(($cpuDelta / ($sw.Elapsed.TotalSeconds * $cores)) * 100, 2)

        $startupMs = "N/A"
        if (Test-Path $startupFile) {
            $startupMs = (Get-Content $startupFile -Raw).Trim()
            Remove-Item $startupFile -Force
        }

        $results += [PSCustomObject]@{
            Framework    = $t.Name
            CGO          = $t.CGO
            BinaryMB     = $sizeMB
            StartupMs    = $startupMs
            Processes    = $m2.ProcessCount
            WorkingSetMB = $m2.WorkingSetMB
            PrivateMB    = $m2.PrivateMB
            IdleCpuPct   = $idleCpuPct
        }
    } finally {
        Remove-Item Env:BENCH_STARTUP_FILE -ErrorAction SilentlyContinue
        & cmd /c "taskkill /F /T /PID $($proc.Id) >nul 2>&1"
    }
}

Write-Host ""
Write-Host "=== Benchmark Results (Windows, $cores logical CPUs) ===" -ForegroundColor Green
$results | Format-Table -AutoSize

$wv2Size = "N/A"
$wv2Root = "C:\Program Files (x86)\Microsoft\EdgeWebView\Application"
if (Test-Path $wv2Root) {
    $latest = Get-ChildItem $wv2Root -Directory | Where-Object { $_.Name -match '^\d+\.' } | Sort-Object Name -Descending | Select-Object -First 1
    if ($latest) {
        $bytes = (Get-ChildItem $latest.FullName -Recurse -File | Measure-Object -Property Length -Sum).Sum
        $wv2Size = "{0:N1} MB (v{1})" -f ($bytes / 1MB), $latest.Name
    }
}

$osInfo = Get-CimInstance Win32_OperatingSystem
$cpuInfo = Get-CimInstance Win32_Processor | Select-Object -First 1
$csInfo = Get-CimInstance Win32_ComputerSystem
$ramGB = [math]::Round($csInfo.TotalPhysicalMemory / 1GB, 1)
$goVer = (& go version).Trim()

$md = @()
$md += "# UI Framework Benchmarks (Windows)"
$md += ""
$md += "**Test Machine:** $($osInfo.Caption) ($($osInfo.OSArchitecture)), $($cpuInfo.Name.Trim()) ($cores logical CPUs), $ramGB GB RAM, ``$goVer``"
$md += ""
$md += "Minimal window (`480x320`) with a label, text input, and click counter button."
$md += "Measured after ${WarmupSec}s warmup over a ${SampleSec}s idle window (including child processes such as WebView2)."
$md += ""
$md += "| Framework | CGO | Binary Size | Startup Time | Processes | Working Set (RAM) | Private Memory | Idle CPU |"
$md += "| --- | :---: | ---: | ---: | ---: | ---: | ---: | ---: |"
foreach ($r in $results) {
    $md += "| $($r.Framework) | $($r.CGO) | $($r.BinaryMB) MB | $($r.StartupMs) ms | $($r.Processes) | $($r.WorkingSetMB) MB | $($r.PrivateMB) MB | $($r.IdleCpuPct)% |"
}
$md += ""
$md += "- **WebView2 Runtime (required by Wails):** $wv2Size on disk."
$md += "- **Startup Time measurement:** time from `main()` start to first rendered frame (`AnyUI`, `Gio`), foreground lifecycle (`Fyne`), or `OnDomReady` (`Wails`)."

$outPath = Join-Path $root "RESULTS.md"
$md -join "`r`n" | Set-Content -Path $outPath -Encoding UTF8
Write-Host "Saved report to $outPath"
