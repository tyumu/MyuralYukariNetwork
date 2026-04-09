# Windows PowerShell helper script to set environment variables

$env:SERVER_PORT = "8000"
$env:SERVER_HOST = "localhost"
$env:LOG_LEVEL = "debug"
$env:DEV_MODE = "true"

$env:MEMORY_GRPC_ENDPOINT = "127.0.0.1:50051"
$env:FRONTEND_ORIGIN = "http://localhost:1420"

# Shared llama.cpp / MemU LLM configuration
$env:LLM_BASE_URL = "http://localhost:11434/v1"
$env:LLM_API_KEY = ""
$env:CHAT_MODEL = "unsloth/gemma-4-E4B-it-GGUF"
$env:EMBED_MODEL = "nomic-embed-text:latest-num-gpu0"
$env:VITE_API_BASE_URL = "http://localhost:8000"

# PostgreSQL connection
$env:POSTGRES_DSN = "postgresql://postgres:1210@127.0.0.1:5433/memu_db"
$env:RETRIEVAL_TOP_K = "5"

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
