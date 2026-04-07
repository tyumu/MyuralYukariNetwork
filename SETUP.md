# Setup Guide (Windows / llama.cpp)

このプロジェクトは以下の 4 サービスで動作します。

1. llama.cpp (`llama-server`) : `11434`
2. Python Sidecar (MemU) : gRPC IPC (`MEMORY_GRPC_ENDPOINT`)
3. Go Backend : `8000`
4. Tauri Frontend : `1420` (dev server)

Go Backend と Python Sidecar の間は HTTP ではなく、ローカル gRPC を使います。

- Windows (このリポジトリの dev 既定): `MEMORY_GRPC_ENDPOINT=127.0.0.1:50051`
- Linux/macOS (`env.sh` 既定): `MEMORY_GRPC_ENDPOINT=unix:///tmp/myural_yukari_memory.sock`

## 1. 前提

- Go 1.21+
- Python 3.11+
- Node.js 18+
- PostgreSQL 14+ (MemU 用)
- llama.cpp (`llama-server`, `llama-cli`)

## 2. 環境変数

PowerShell:

```powershell
cd <MyuralYukariNetwork のクローン先>
. .\env.ps1
```

または環境変数を手動で設定する場合は、少なくとも次を揃えてください:

```env
SERVER_PORT=8000
SERVER_HOST=localhost
LOG_LEVEL=debug
DEV_MODE=true
FRONTEND_ORIGIN=http://localhost:1420

MEMORY_GRPC_ENDPOINT=127.0.0.1:50051
LLM_BASE_URL=http://localhost:11434/v1
LLM_API_KEY=
CHAT_MODEL=unsloth/gemma-4-E4B-it-GGUF
EMBED_MODEL=nomic-embed-text:latest-num-gpu0
VITE_API_BASE_URL=http://localhost:8000
POSTGRES_DSN=postgresql://postgres:1210@127.0.0.1:5433/memu_db
RETRIEVAL_TOP_K=5

# Optional overrides for scripts/start-dev.ps1
LLAMA_HF_REPO=unsloth/gemma-4-E4B-it-GGUF
LLAMA_HF_FILE=gemma-4-E4B-it-UD-Q4_K_XL.gguf
LLAMA_MODEL_PATH=
LLAMA_NGL=99
LLAMA_CONTEXT=8192
LLAMA_POOLING=mean
SIDECAR_HEALTH_STRICT=false
SIDECAR_PYTHON_EXE=
```

## 3. llama.cpp 起動

### A. Hugging Face から直接ロード (現在の実績コマンド)

```powershell
llama-server --hf-repo unsloth/gemma-4-E4B-it-GGUF --hf-file gemma-4-E4B-it-UD-Q4_K_XL.gguf --port 11434 --embedding --pooling mean -ngl 99 -c 8192
```

### B. ローカル GGUF からロード

```powershell
cd C:\LLM\llama
.\llama-server.exe -m "C:/LLM/models/your-model.gguf" --port 11434 --embedding --pooling mean -ngl 99 -c 8192
```

起動確認:

```powershell
Invoke-WebRequest -UseBasicParsing http://localhost:11434/v1/models
```

## 4. 開発起動を自動化する

起動順をまとめて実行するには、ルートで次を使います。

```powershell
cd <MyuralYukariNetwork のクローン先>
. .\env.ps1
.\scripts\start-dev.ps1
```

このスクリプトは `llama.cpp -> Python Sidecar -> Go Backend -> Tauri Frontend` の順で起動し、
`/v1/models`、sidecar gRPC `Health`、`/status`、`http://localhost:1420` の readiness を確認してから次へ進みます。
すでに起動済みの対象サービスは既存プロセスとして追跡し、`stop-dev.ps1` からまとめて停止できます。
また、追跡 state がない場合でも `stop-dev.ps1` は best-effort cleanup を試みます。

停止するときは以下を実行します。

```powershell
.\scripts\stop-dev.ps1
```

## 5. PostgreSQL 起動

PostgreSQL を起動し、必要なら DB を作成:

```powershell
psql -h 127.0.0.1 -p 5433 -U postgres -d postgres -c "CREATE DATABASE memu_db;"
```

## 6. Python Sidecar 起動

```powershell
cd <MyuralYukariNetwork のクローン先>\python-sidecar
..\memU\.venv\Scripts\python.exe main.py
```

## 7. Go Backend 起動

```powershell
cd <MyuralYukariNetwork のクローン先>\go-backend
go run ./cmd/server/main.go
```

確認:

```powershell
Invoke-WebRequest -UseBasicParsing http://localhost:8000/status
```

## 8. Tauri Frontend 起動

```powershell
cd <MyuralYukariNetwork のクローン先>\tauri_app
npm install
npm run tauri:dev
```

## 9. 疎通テスト

### Go `/chat` テスト

```powershell
$body = @{ user_id='test_user'; message='こんにちは。短く返答して'; session_id='' } | ConvertTo-Json
Invoke-WebRequest -UseBasicParsing -Uri http://localhost:8000/chat -Method POST -ContentType 'application/json' -Body $body -TimeoutSec 210
```

成功時は `message.content` が返ります。

### 期待ステータス

- `http://localhost:11434/v1/models` : 200
- sidecar gRPC `Health` : `healthy=true`
- `http://localhost:8000/status` : `services.memory=ok`, `services.llama_cpp=ok`

`SIDECAR_HEALTH_STRICT=true` のときは sidecar gRPC `Health` が埋め込み実行まで検証します。`false` の場合はサービス初期化のみ確認します。

`CHAT_MODEL` は `/v1/models` の `id`（例: `unsloth/gemma-4-E4B-it-GGUF`）に合わせます。`EMBED_MODEL` は埋め込み対応モデル（例: `nomic-embed-text:latest-num-gpu0`）を指定してください。

## 10. 停止と片付け

開発終了時は以下の 4 つを止めれば OK:

1. `llama-server` (11434)
2. `python main.py` (gRPC: 127.0.0.1:50051)
3. `go run ./cmd/server/main.go` (8000)
4. `npm run tauri:dev` (1420)

自動起動スクリプトを使った場合は、[scripts/stop-dev.ps1](scripts/stop-dev.ps1) を実行して追跡中のプロセスをまとめて止められます。

ポート確認:

```powershell
$projectPorts = @(5433,8000,11434,1420)
foreach ($pt in $projectPorts) {
  $hit = Get-NetTCPConnection -State Listen -LocalPort $pt -ErrorAction SilentlyContinue | Select-Object -First 1
  if ($hit) { "${pt}: LISTENING (PID=$($hit.OwningProcess))" } else { "${pt}: FREE" }
}
```

Windows 既定の sidecar は TCP (`127.0.0.1:50051`) を使うため、ポート一覧で確認できます。

---

この手順で、llama.cpp 構成でアプリ起動から `/chat` 応答まで確認できます。
