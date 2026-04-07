param()

$ErrorActionPreference = 'Stop'

Set-StrictMode -Version Latest

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot '..')
$DevStateDir = Join-Path $env:TEMP 'MyuralYukariNetwork-dev'
$ProcessStateFile = Join-Path $DevStateDir 'processes.json'

$script:StoppedPidMap = @{}
$script:StoppedAny = $false

function Stop-DevPid {
    param(
        [Parameter(Mandatory = $true)]
        [int]$ProcessId,
        [Parameter(Mandatory = $true)]
        [string]$Label
    )

    if ($ProcessId -le 0) {
        return
    }

    $pidKey = [string]$ProcessId
    if ($script:StoppedPidMap.ContainsKey($pidKey)) {
        return
    }

    $proc = Get-Process -Id $ProcessId -ErrorAction SilentlyContinue
    if ($null -eq $proc) {
        return
    }

    try {
        Stop-Process -Id $ProcessId -Force -ErrorAction Stop
        $script:StoppedPidMap[$pidKey] = $true
        $script:StoppedAny = $true
        Write-Host "Stopped $Label (PID=$ProcessId)" -ForegroundColor Cyan
    }
    catch {
        Write-Host "Could not stop $Label (PID=$ProcessId)" -ForegroundColor Yellow
    }
}

function Invoke-BestEffortCleanup {
    foreach ($name in @('llama-server', 'myural_yukari_tauri')) {
        foreach ($proc in @(Get-Process -Name $name -ErrorAction SilentlyContinue)) {
            Stop-DevPid -ProcessId $proc.Id -Label $name
        }
    }

    $procRows = @(Get-CimInstance Win32_Process -ErrorAction SilentlyContinue)
    foreach ($row in $procRows) {
        $cmd = [string]$row.CommandLine
        if ([string]::IsNullOrWhiteSpace($cmd)) {
            continue
        }

        $cmdLower = $cmd.ToLowerInvariant()
        if ($cmdLower -match 'python-sidecar[\\/].*main\.py') {
            Stop-DevPid -ProcessId ([int]$row.ProcessId) -Label 'python-sidecar'
            continue
        }

        if ($cmdLower -match 'go\s+run\s+.*cmd[\\/]server[\\/]main\.go') {
            Stop-DevPid -ProcessId ([int]$row.ProcessId) -Label 'go-backend'
            continue
        }

        if ($cmdLower -match 'tauri:dev' -and $cmdLower -match 'tauri_app') {
            Stop-DevPid -ProcessId ([int]$row.ProcessId) -Label 'tauri-app'
            continue
        }
    }
}

if (-not (Test-Path $ProcessStateFile)) {
    Write-Host 'No tracked dev processes found. Trying best-effort cleanup...' -ForegroundColor Yellow
    Invoke-BestEffortCleanup
    if ($script:StoppedAny) {
        Write-Host "Best-effort cleanup complete for $RepoRoot" -ForegroundColor Green
    } else {
        Write-Host 'No matching dev processes found.' -ForegroundColor Yellow
    }
    return
}

$trackedProcesses = Get-Content $ProcessStateFile -Raw | ConvertFrom-Json
foreach ($proc in @($trackedProcesses)) {
    $label = [string]$proc.Name
    if ([string]::IsNullOrWhiteSpace($label)) {
        $label = 'tracked-process'
    }
    Stop-DevPid -ProcessId ([int]$proc.Pid) -Label $label
}

Remove-Item $ProcessStateFile -Force -ErrorAction SilentlyContinue
Write-Host "Cleaned tracked dev processes for $RepoRoot" -ForegroundColor Green
