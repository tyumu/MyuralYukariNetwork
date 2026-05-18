# Windows PowerShell helper script to set environment variables
# Use it from the repo root with:
#   . .\env.ps1
#
# This mirrors .env.example, but is meant for Windows shells and manual startup.

$env:SERVER_PORT = "8000"
$env:SERVER_HOST = "localhost"
$env:LOG_LEVEL = "debug"
$env:DEV_MODE = "true"

$env:MEMORY_GRPC_ENDPOINT = "127.0.0.1:50051"
$env:FRONTEND_ORIGIN = "http://localhost:1420"

# Shared llama.cpp / MemU LLM configuration
$env:LLM_BASE_URL = "http://localhost:11434/v1"
$env:LLM_API_KEY = "ollama"
$env:CHAT_MODEL = "gemma3:12b"
$env:EMBED_MODEL = "nomic-embed-text"
$env:VITE_API_BASE_URL = "http://localhost:8000"

# MemU DB
$env:MEMORY_DB_PROVIDER = "postgres"
$env:MEMORY_DB_DSN = "postgresql://postgres:1210@127.0.0.1:5433/memu_db"
$env:MEMORY_DB_DDL_MODE = "create"
$env:RETRIEVAL_TOP_K = "5"
$env:SIDECAR_HEALTH_STRICT = "false"

Write-Host "✓ Environment variables loaded" -ForegroundColor Green
Write-Host ""
Write-Host "Dev Mode: true" -ForegroundColor Cyan
Write-Host "Log Level: debug" -ForegroundColor Cyan
Write-Host ""
Write-Host "Ready to start services:" -ForegroundColor Yellow
Write-Host "  1. llama.cpp:       start llama-server" -ForegroundColor Gray
Write-Host "  2. Python Sidecar:  cd python-sidecar && python main.py" -ForegroundColor Gray
Write-Host "  3. Go Backend:      cd go-backend && go run ./cmd/server/main.go" -ForegroundColor Gray
Write-Host "  4. Tauri Frontend:  cd tauri_app && npm run tauri:dev" -ForegroundColor Gray

# scripts/start-dev.ps1 用の任意上書き（必要なときだけ設定）
# LLAMA_SERVER_EXE=
# LLAMA_HF_REPO=unsloth/gemma-4-E4B-it-GGUF
# LLAMA_HF_FILE=gemma-4-E4B-it-UD-Q4_K_XL.gguf
# LLAMA_MODEL_PATH=
# LLAMA_NGL=99
# LLAMA_CONTEXT=8192
# LLAMA_POOLING=mean
# SIDECAR_PYTHON_EXE=
