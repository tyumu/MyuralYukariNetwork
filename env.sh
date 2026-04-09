#!/bin/bash
# macOS / Linux helper script to set environment variables
# Use it from the repo root with:
#   source ./env.sh
#
# This mirrors .env.example, but is meant for POSIX shells and manual startup.

export SERVER_PORT="8000"
export SERVER_HOST="localhost"
export LOG_LEVEL="debug"
export DEV_MODE="true"

export MEMORY_GRPC_ENDPOINT="unix:///tmp/myural_yukari_memory.sock"
export FRONTEND_ORIGIN="http://localhost:1420"

# Shared llama.cpp / MemU LLM configuration
export LLM_BASE_URL="http://localhost:11434/v1"
export LLM_API_KEY=""
export CHAT_MODEL="unsloth/gemma-4-E4B-it-GGUF"
export EMBED_MODEL="nomic-embed-text:latest-num-gpu0"
export VITE_API_BASE_URL="http://localhost:8000"

# PostgreSQL connection
export POSTGRES_DSN="postgresql://postgres:1210@127.0.0.1:5433/memu_db"
export RETRIEVAL_TOP_K="5"
export SIDECAR_HEALTH_STRICT="false"

echo "✓ Environment variables loaded"
echo ""
echo "Dev Mode: true"
echo "Log Level: debug"
echo ""
echo "Ready to start services:"
echo "  1. llama.cpp:       start llama-server"
echo "  2. Python Sidecar:  cd python-sidecar && python main.py"
echo "  3. Go Backend:      cd go-backend && go run ./cmd/server/main.go"
echo "  4. Tauri Frontend:  cd tauri_app && npm run tauri:dev"

# scripts/start-dev.ps1 用の任意上書き（必要なときだけ設定）
# LLAMA_SERVER_EXE=
# LLAMA_HF_REPO=unsloth/gemma-4-E4B-it-GGUF
# LLAMA_HF_FILE=gemma-4-E4B-it-UD-Q4_K_XL.gguf
# LLAMA_MODEL_PATH=
# LLAMA_NGL=99
# LLAMA_CONTEXT=8192
# LLAMA_POOLING=mean
# SIDECAR_PYTHON_EXE=