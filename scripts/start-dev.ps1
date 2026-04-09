param()

$ErrorActionPreference = 'Stop'

Set-StrictMode -Version Latest

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot '..')
. (Join-Path $RepoRoot 'env.ps1')

$DevStateDir = Join-Path $env:TEMP 'MyuralYukariNetwork-dev'
New-Item -ItemType Directory -Force -Path $DevStateDir | Out-Null
$ProcessStateFile = Join-Path $DevStateDir 'processes.json'

$script:StartedProcesses = @()

function Add-TrackedProcess {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,
        [Parameter(Mandatory = $true)]
        [int]$ProcessId
    )

    foreach ($entry in $script:StartedProcesses) {
        if ($entry.Pid -eq $ProcessId) {
            return
        }
    }

    $script:StartedProcesses += [pscustomobject]@{
        Name = $Name
        Pid  = $ProcessId
    }

    try {
        Write-ProcessState
    }
    catch {
        Write-Host "[dev-state] could not persist process state: $($_.Exception.Message)" -ForegroundColor DarkYellow
    }
}

function Get-ListeningPid {
    param(
        [Parameter(Mandatory = $true)]
        [int]$Port
    )

    $conn = Get-NetTCPConnection -State Listen -LocalPort $Port -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($null -eq $conn) {
        return $null
    }
    return [int]$conn.OwningProcess
}

function Test-UrlReady {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Url,
        [int]$RequestTimeoutSeconds = 5
    )

    try {
        $response = Invoke-WebRequest -UseBasicParsing -Uri $Url -TimeoutSec $RequestTimeoutSeconds
        return $response.StatusCode -ge 200 -and $response.StatusCode -lt 300
    } catch {
        return $false
    }
}

function Get-ProcessExitDiagnostic {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Label,
        [Parameter(Mandatory = $true)]
        [int]$ProcessId,
        [Parameter(Mandatory = $true)]
        [string]$Url,
        [string]$FailureLogPath = ''
    )

    $message = "$Label process (PID=$ProcessId) exited before becoming ready at $Url"
    if (-not [string]::IsNullOrWhiteSpace($FailureLogPath) -and (Test-Path $FailureLogPath)) {
        $stderrTail = @(Get-Content $FailureLogPath -Tail 20 -ErrorAction SilentlyContinue)
        if ($stderrTail.Count -gt 0) {
            $tailText = ($stderrTail -join "`n")
            $message = "$message`n--- stderr tail ($FailureLogPath) ---`n$tailText"

            $tailTextLower = $tailText.ToLowerInvariant()
            if ($tailTextLower.Contains('rate limit') -or $tailTextLower.Contains('429')) {
                $message = "$message`nHint: Hugging Face rate limit hit. Set LLAMA_MODEL_PATH to a local .gguf, or configure Hugging Face authentication."
            }
        } else {
            $message = "$message`nCheck log: $FailureLogPath"
        }
    }

    return $message
}

function Get-TimeoutDiagnostic {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Label,
        [Parameter(Mandatory = $true)]
        [string]$Target,
        [string]$FailureLogPath = ''
    )

    $message = "Timed out waiting for $Label at $Target"
    if (-not [string]::IsNullOrWhiteSpace($FailureLogPath) -and (Test-Path $FailureLogPath)) {
        $stderrTail = @(Get-Content $FailureLogPath -Tail 20 -ErrorAction SilentlyContinue)
        if ($stderrTail.Count -gt 0) {
            $tailText = ($stderrTail -join "`n")
            $message = "$message`n--- stderr tail ($FailureLogPath) ---`n$tailText"

            $tailTextLower = $tailText.ToLowerInvariant()
            if ($tailTextLower.Contains('rate limit') -or $tailTextLower.Contains('429')) {
                $message = "$message`nHint: Hugging Face rate limit hit. Set LLAMA_MODEL_PATH to a local .gguf, or configure Hugging Face authentication."
            }
        } else {
            $message = "$message`nCheck log: $FailureLogPath"
        }
    }

    return $message
}

function Wait-ForUrl {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Url,
        [Parameter(Mandatory = $true)]
        [string]$Label,
        [int]$TimeoutSeconds = 180,
        [int]$RequestTimeoutSeconds = 5,
        [int]$ProcessId = 0,
        [string]$FailureLogPath = ''
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        if ($ProcessId -gt 0) {
            $proc = Get-Process -Id $ProcessId -ErrorAction SilentlyContinue
            if ($null -eq $proc) {
                throw (Get-ProcessExitDiagnostic -Label $Label -ProcessId $ProcessId -Url $Url -FailureLogPath $FailureLogPath)
            }
        }

        if (Test-UrlReady -Url $Url -RequestTimeoutSeconds $RequestTimeoutSeconds) {
            Write-Host "[$Label] ready at $Url" -ForegroundColor Green
            return
        }

        Start-Sleep -Milliseconds 1000
    }

    throw (Get-TimeoutDiagnostic -Label $Label -Target $Url -FailureLogPath $FailureLogPath)
}

function Test-GrpcMemoryHealth {
    param(
        [Parameter(Mandatory = $true)]
        [string]$PythonExe,
        [Parameter(Mandatory = $true)]
        [string]$Endpoint,
        [int]$TimeoutSeconds = 5
    )

    $healthScript = @'
import asyncio
import sys
from pathlib import Path

import grpc

repo_root = Path(sys.argv[1]).resolve()
endpoint = sys.argv[2]
timeout_s = float(sys.argv[3])

grpc_gen = repo_root / "python-sidecar" / "src" / "grpc_gen"
if str(grpc_gen) not in sys.path:
    sys.path.insert(0, str(grpc_gen))

import memory_pb2
import memory_pb2_grpc

async def main() -> int:
    try:
        async with grpc.aio.insecure_channel(endpoint) as channel:
            stub = memory_pb2_grpc.MemoryServiceStub(channel)
            response = await asyncio.wait_for(stub.Health(memory_pb2.HealthRequest()), timeout=timeout_s)
            return 0 if response.healthy else 1
    except Exception:
        return 2

raise SystemExit(asyncio.run(main()))
'@

    & $PythonExe -c $healthScript $RepoRoot $Endpoint $TimeoutSeconds | Out-Null
    return ($LASTEXITCODE -eq 0)
}

function Wait-ForGrpcMemoryHealth {
    param(
        [Parameter(Mandatory = $true)]
        [string]$PythonExe,
        [Parameter(Mandatory = $true)]
        [string]$Endpoint,
        [int]$ProcessId = 0,
        [int]$TimeoutSeconds = 180,
        [int]$RequestTimeoutSeconds = 5,
        [string]$FailureLogPath = ''
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        if ($ProcessId -gt 0) {
            $proc = Get-Process -Id $ProcessId -ErrorAction SilentlyContinue
            if ($null -eq $proc) {
                throw (Get-ProcessExitDiagnostic -Label 'Python sidecar' -ProcessId $ProcessId -Url "grpc://$Endpoint" -FailureLogPath $FailureLogPath)
            }
        }

        if (Test-GrpcMemoryHealth -PythonExe $PythonExe -Endpoint $Endpoint -TimeoutSeconds $RequestTimeoutSeconds) {
            Write-Host "[Python sidecar] gRPC healthy at $Endpoint" -ForegroundColor Green
            return
        }

        Start-Sleep -Milliseconds 1000
    }

    throw (Get-TimeoutDiagnostic -Label 'Python sidecar gRPC health' -Target "grpc://$Endpoint" -FailureLogPath $FailureLogPath)
}

function Start-LoggedProcess {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,
        [Parameter(Mandatory = $true)]
        [string]$FilePath,
        [string[]]$ArgumentList = @(),
        [Parameter(Mandatory = $true)]
        [string]$WorkingDirectory
    )

    $stdoutPath = Join-Path $DevStateDir "$Name.stdout.log"
    $stderrPath = Join-Path $DevStateDir "$Name.stderr.log"

    $processFilePath = $FilePath
    $processArgumentList = $ArgumentList

    if ($FilePath.ToLowerInvariant().EndsWith('.cmd') -or $FilePath.ToLowerInvariant().EndsWith('.bat')) {
        $batchPathArgument = '"{0}"' -f $FilePath
        $processFilePath = 'cmd.exe'
        $processArgumentList = @('/c', $batchPathArgument) + $ArgumentList
    }

    $process = Start-Process -FilePath $processFilePath -ArgumentList $processArgumentList -WorkingDirectory $WorkingDirectory -PassThru -RedirectStandardOutput $stdoutPath -RedirectStandardError $stderrPath

    Add-TrackedProcess -Name $Name -ProcessId $process.Id

    Write-Host "[$Name] started (PID=$($process.Id))" -ForegroundColor Cyan

    return $process
}

function Resolve-CommandPath {
    param(
        [Parameter(Mandatory = $true)]
        [string[]]$Candidates
    )

    foreach ($candidate in $Candidates) {
        $command = Get-Command $candidate -ErrorAction SilentlyContinue | Select-Object -First 1
        if ($null -ne $command -and ($command.Source -or $command.Path)) {
            return $command.Source ?? $command.Path
        }
    }

    return $null
}

function Get-EnvOrDefault {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,
        [AllowEmptyString()]
        [string]$DefaultValue
    )

    $value = [Environment]::GetEnvironmentVariable($Name)
    if ([string]::IsNullOrWhiteSpace($value)) {
        return $DefaultValue
    }

    return $value
}

function Get-UrlPort {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Url,
        [int]$DefaultPort = 0
    )

    try {
        $uri = [System.Uri]$Url
        if ($uri.Port -gt 0) {
            return [int]$uri.Port
        }
    } catch {
    }

    return $DefaultPort
}

function Write-ProcessState {
    $script:StartedProcesses | ConvertTo-Json -Depth 4 | Set-Content -Encoding UTF8 $ProcessStateFile
}

function Stop-ProcessByName {
    param(
        [Parameter(Mandatory = $true)]
        [string]$ProcessName
    )

    $processes = Get-Process -Name $ProcessName -ErrorAction SilentlyContinue
    foreach ($proc in @($processes)) {
        try {
            Stop-Process -Id $proc.Id -Force -ErrorAction Stop
            Write-Host "[$ProcessName] stopped stale process (PID=$($proc.Id))" -ForegroundColor DarkGray
        } catch {
        }
    }
}

function Invoke-ServiceStep {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,
        [Parameter(Mandatory = $true)]
        [string]$Url,
        [Parameter(Mandatory = $true)]
        [int]$Port,
        [Parameter(Mandatory = $true)]
        [scriptblock]$StartAction,
        [int]$TimeoutSeconds = 180,
        [int]$RequestTimeoutSeconds = 5,
        [string]$FailureLogPath = ''
    )

    if (Test-UrlReady -Url $Url -RequestTimeoutSeconds $RequestTimeoutSeconds) {
        Write-Host "[$Name] already running" -ForegroundColor Yellow
        $existingPid = Get-ListeningPid -Port $Port
        if ($null -ne $existingPid) {
            Add-TrackedProcess -Name $Name -ProcessId $existingPid
            Write-Host "[$Name] tracked existing process (PID=$existingPid)" -ForegroundColor DarkGray
        }
        return
    }

    $startedProcess = & $StartAction | Select-Object -First 1

    $startedProcessId = 0
    if ($null -ne $startedProcess) {
        if ($startedProcess -is [System.Diagnostics.Process]) {
            $startedProcessId = [int]$startedProcess.Id
        } elseif ($startedProcess.PSObject.Properties.Name -contains 'Id') {
            try {
                $startedProcessId = [int]$startedProcess.Id
            } catch {
            }
        }
    }

    Wait-ForUrl -Url $Url -Label $Name -TimeoutSeconds $TimeoutSeconds -RequestTimeoutSeconds $RequestTimeoutSeconds -ProcessId $startedProcessId -FailureLogPath $FailureLogPath
}

function Resolve-SidecarPython {
    if ($env:SIDECAR_PYTHON_EXE) {
        $resolvedOverride = Resolve-CommandPath -Candidates @($env:SIDECAR_PYTHON_EXE)
        if ($resolvedOverride) {
            return $resolvedOverride
        }
        return $env:SIDECAR_PYTHON_EXE
    }

    $candidates = @(
        (Join-Path $RepoRoot '.venv\Scripts\python.exe'),
        (Join-Path $RepoRoot 'memU\.venv\Scripts\python.exe')
    )

    foreach ($candidate in $candidates) {
        if (Test-Path $candidate) {
            return $candidate
        }
    }

    $resolvedPython = Resolve-CommandPath -Candidates @('python', 'python.exe', 'py')
    if ($resolvedPython) {
        return $resolvedPython
    }

    return 'python'
}

try {
    Write-Host 'Starting dev services in order: llama.cpp -> Python sidecar -> Go backend -> Tauri frontend' -ForegroundColor White

    $llamaCommand = if ($env:LLAMA_SERVER_EXE) { $env:LLAMA_SERVER_EXE } else { (Resolve-CommandPath -Candidates @('llama-server', 'llama-server.exe')) }
    if (-not $llamaCommand) { throw 'llama-server was not found on PATH. Set LLAMA_SERVER_EXE or install llama.cpp.' }

    $llmBaseUrl = (Get-EnvOrDefault -Name 'LLM_BASE_URL' -DefaultValue 'http://127.0.0.1:11434/v1').TrimEnd('/')
    $llamaModelsUrl = "$llmBaseUrl/models"
    $llamaPort = Get-UrlPort -Url $llmBaseUrl -DefaultPort 11434

    $llamaNgl = Get-EnvOrDefault -Name 'LLAMA_NGL' -DefaultValue '99'
    $llamaContext = Get-EnvOrDefault -Name 'LLAMA_CONTEXT' -DefaultValue '8192'
    $llamaPooling = Get-EnvOrDefault -Name 'LLAMA_POOLING' -DefaultValue 'mean'

    $llamaArgs = @(
        '--port', "$llamaPort",
        '--embedding',
        '--pooling', $llamaPooling,
        '-ngl', $llamaNgl,
        '-c', $llamaContext
    )

    $llamaModelPath = Get-EnvOrDefault -Name 'LLAMA_MODEL_PATH' -DefaultValue ''
    if (-not [string]::IsNullOrWhiteSpace($llamaModelPath)) {
        $llamaArgs = @('-m', $llamaModelPath) + $llamaArgs
    } else {
        $llamaHfRepo = Get-EnvOrDefault -Name 'LLAMA_HF_REPO' -DefaultValue 'unsloth/gemma-4-E4B-it-GGUF'
        $llamaHfFile = Get-EnvOrDefault -Name 'LLAMA_HF_FILE' -DefaultValue 'gemma-4-E4B-it-UD-Q4_K_XL.gguf'
        $llamaArgs = @('--hf-repo', $llamaHfRepo, '--hf-file', $llamaHfFile) + $llamaArgs
    }

    Invoke-ServiceStep -Name 'llama.cpp' -Url $llamaModelsUrl -Port $llamaPort -TimeoutSeconds 300 -RequestTimeoutSeconds 10 -FailureLogPath (Join-Path $DevStateDir 'llama-server.stderr.log') -StartAction {
        Start-LoggedProcess -Name 'llama-server' -FilePath $llamaCommand -ArgumentList $llamaArgs -WorkingDirectory $RepoRoot
    }

    $sidecarPythonExe = Resolve-SidecarPython
    if (-not (Test-Path $sidecarPythonExe)) {
        $resolvedSidecarPython = Resolve-CommandPath -Candidates @($sidecarPythonExe)
        if ($resolvedSidecarPython) {
            $sidecarPythonExe = $resolvedSidecarPython
        } else {
            throw "Python executable not found: $sidecarPythonExe"
        }
    }

    $memoryGrpcEndpoint = Get-EnvOrDefault -Name 'MEMORY_GRPC_ENDPOINT' -DefaultValue '127.0.0.1:50051'
    if (Test-GrpcMemoryHealth -PythonExe $sidecarPythonExe -Endpoint $memoryGrpcEndpoint -TimeoutSeconds 10) {
        Write-Host "[Python sidecar] already running (gRPC healthy at $memoryGrpcEndpoint)" -ForegroundColor Yellow
    } else {
        $sidecarProcess = Start-LoggedProcess -Name 'python-sidecar' -FilePath $sidecarPythonExe -ArgumentList @('main.py') -WorkingDirectory (Join-Path $RepoRoot 'python-sidecar')
        Wait-ForGrpcMemoryHealth -PythonExe $sidecarPythonExe -Endpoint $memoryGrpcEndpoint -ProcessId $sidecarProcess.Id -TimeoutSeconds 180 -RequestTimeoutSeconds 10 -FailureLogPath (Join-Path $DevStateDir 'python-sidecar.stderr.log')
    }

    $goCommand = Resolve-CommandPath -Candidates @('go', 'go.exe')
    if (-not $goCommand) { throw 'go was not found on PATH. Install Go or add it to PATH.' }

    $serverHost = Get-EnvOrDefault -Name 'SERVER_HOST' -DefaultValue '127.0.0.1'
    $serverPort = [int](Get-EnvOrDefault -Name 'SERVER_PORT' -DefaultValue '8000')
    $serverStatusUrl = "http://${serverHost}:$serverPort/status"
    Invoke-ServiceStep -Name 'Go backend' -Url $serverStatusUrl -Port $serverPort -TimeoutSeconds 180 -RequestTimeoutSeconds 10 -FailureLogPath (Join-Path $DevStateDir 'go-backend.stderr.log') -StartAction {
        Start-LoggedProcess -Name 'go-backend' -FilePath $goCommand -ArgumentList @('run', './cmd/server/main.go') -WorkingDirectory (Join-Path $RepoRoot 'go-backend')
    }

    $npmCommand = Resolve-CommandPath -Candidates @('npm.cmd', 'npm')
    if (-not $npmCommand) { throw 'npm was not found on PATH. Install Node.js or add it to PATH.' }

    # Tauri binary can remain running after previous sessions and lock target/debug executable.
    Stop-ProcessByName -ProcessName 'myural_yukari_tauri'

    Invoke-ServiceStep -Name 'Tauri frontend' -Url 'http://localhost:1420' -Port 1420 -TimeoutSeconds 180 -RequestTimeoutSeconds 10 -FailureLogPath (Join-Path $DevStateDir 'tauri-app.stderr.log') -StartAction {
        Start-LoggedProcess -Name 'tauri-app' -FilePath $npmCommand -ArgumentList @('run', 'tauri:dev') -WorkingDirectory (Join-Path $RepoRoot 'tauri_app')
    }

    Write-Host "`nAll requested services are ready." -ForegroundColor Green
    Write-Host "Logs: $DevStateDir" -ForegroundColor DarkGray
    Write-Host "State: $ProcessStateFile" -ForegroundColor DarkGray
}
catch {
    Write-Host "Startup failed: $($_.Exception.Message)" -ForegroundColor Red

    foreach ($proc in $script:StartedProcesses) {
        try {
            Stop-Process -Id $proc.Pid -Force -ErrorAction SilentlyContinue
        } catch {
        }
    }

    if (Test-Path $ProcessStateFile) {
        Remove-Item $ProcessStateFile -Force -ErrorAction SilentlyContinue
    }

    throw
}
