# 開発クイックリファレンス

このドキュメントは、現在の実装と起動フローに合わせた最短参照です。

## 主要ディレクトリ

```text
MyuralYukariNetwork/
|- go-backend/                 # Go API サーバー
|  |- cmd/server/main.go       # エントリーポイント
|  |- internal/
|  |  |- api/                  # chat/status ハンドラ群
|  |  |  |- handler.go
|  |  |  |- chat_handler.go
|  |  |  |- chat_flow.go
|  |  |  |- status_handler.go
|  |  |- config/config.go
|  |  |- memory/client.go
|  |  |- ollama/client.go
|  |- shared/schemas.go        # Go 側契約
|
|- python-sidecar/             # MemU sidecar (gRPC IPC)
|  |- main.py
|  |- src/
|  |  |- contracts.py          # Python 側契約
|  |  |- config/__init__.py
|  |  |- logger/__init__.py
|
|- contracts/
|  |- memory.proto             # gRPC / Protobuf 契約
|
|- scripts/
|  |- start-dev.ps1            # 4サービス自動起動
|  |- stop-dev.ps1             # 追跡PIDの停止
|
|- tauri_app/                  # Tauri + React フロント
|  |- src/main.jsx
|  |- src/App.jsx
|  |- src/api/index.js
|  |- package.json
|
|- env.ps1
|- env.sh
|- SETUP.md
|- ARCHITECTURE.md
|- API_SPEC.md
```

## 最短起動 (Windows)

```powershell
cd <MyuralYukariNetwork のクローン先>
. .\env.ps1
.\scripts\start-dev.ps1
```

停止:

```powershell
.\scripts\stop-dev.ps1
```

`stop-dev.ps1` は追跡 state が見つからない場合でも、主要プロセスに対して best-effort cleanup を行います。

## env の役割

- [.env](.env) は Go backend と Python sidecar が起動時に読む共通設定です
- [env.ps1](env.ps1) は Windows PowerShell で手動起動するときの補助です
- [env.sh](env.sh) は macOS / Linux で手動起動するときの補助です
- [scripts/start-dev.ps1](scripts/start-dev.ps1) は [env.ps1](env.ps1) を読み、必要なら llama.cpp や sidecar の上書き値も参照します
- [.env.example](.env.example) は新規作成時のひな形で、実運用では [.env](.env) にコピーして使います

## env 変数一覧

### 共通設定

これらは [.env](.env) に書く値で、[env.ps1](env.ps1) と [env.sh](env.sh) にも同じ意味で置けます。

- `SERVER_PORT`
- `SERVER_HOST`
- `LOG_LEVEL`
- `DEV_MODE`
- `FRONTEND_ORIGIN`
- `MEMORY_GRPC_ENDPOINT`
- `LLM_BASE_URL`
- `LLM_API_KEY`
- `CHAT_MODEL`
- `EMBED_MODEL`
- `POSTGRES_DSN`
- `RETRIEVAL_TOP_K`
- `VITE_API_BASE_URL`
- `SIDECAR_HEALTH_STRICT`

### start-dev.ps1 / env.ps1 / env.sh 専用の上書き

これらは通常の .env ではなく、`start-dev.ps1` で起動するローカル開発時の補助値です。

- `LLAMA_SERVER_EXE`
- `LLAMA_HF_REPO`
- `LLAMA_HF_FILE`
- `LLAMA_MODEL_PATH`
- `LLAMA_NGL`
- `LLAMA_CONTEXT`
- `LLAMA_POOLING`
- `SIDECAR_PYTHON_EXE`

### 使い分けの目安

- すべての起動方法で共通に効かせたい値は [.env](.env) に置く
- Windows の手動起動は [env.ps1](env.ps1) を使う
- macOS / Linux の手動起動は [env.sh](env.sh) を使う
- llama.cpp の場所やモデルファイルの指定を変えたいときだけ start-dev 専用の上書きを使う

`start-dev.ps1` は次を順番に起動/確認します。

1. llama.cpp (`LLM_BASE_URL/models`)
2. Python sidecar (gRPC `Health` via `MEMORY_GRPC_ENDPOINT`)
3. Go backend (`http://<SERVER_HOST>:<SERVER_PORT>/status`)
4. Tauri frontend (`http://localhost:1420`)

Go backend と sidecar の本番経路は `MEMORY_GRPC_ENDPOINT` を使う gRPC です
(Windows 開発既定: `127.0.0.1:50051`、Linux/macOS helper 既定: `unix:///tmp/myural_yukari_memory.sock`)。

## 手動起動 (Windows)

### 1. llama.cpp

```powershell
llama-server --hf-repo unsloth/gemma-4-E4B-it-GGUF --hf-file gemma-4-E4B-it-UD-Q4_K_XL.gguf --port 11434 --embedding --pooling mean -ngl 99 -c 8192
```

### 2. Python sidecar

```powershell
cd python-sidecar
..\memU\.venv\Scripts\python.exe main.py
```

### 3. Go backend

```powershell
cd go-backend
go run ./cmd/server/main.go
```

### 4. Tauri frontend

```powershell
cd tauri_app
npm install
npm run tauri:dev
```

## よく使う確認コマンド

### ヘルス確認

```powershell
Invoke-WebRequest -UseBasicParsing http://localhost:11434/v1/models
Invoke-WebRequest -UseBasicParsing http://localhost:8000/status
```

### チャット疎通

```powershell
$body = @{ user_id='test_user'; message='こんにちは'; session_id='' } | ConvertTo-Json
Invoke-WebRequest -UseBasicParsing -Uri http://localhost:8000/chat -Method POST -ContentType 'application/json' -Body $body -TimeoutSec 220
```

## 主要環境変数

- `SERVER_HOST` (default: `localhost`)
- `SERVER_PORT` (default: `8000`)
- `FRONTEND_ORIGIN` (default: `http://localhost:1420`)
- `MEMORY_GRPC_ENDPOINT` (Windows dev default: `127.0.0.1:50051`)
- `LLM_BASE_URL` (default: `http://localhost:11434/v1`)
- `CHAT_MODEL` (default: `unsloth/gemma-4-E4B-it-GGUF`)
- `EMBED_MODEL` (default: `nomic-embed-text:latest-num-gpu0`)
- `POSTGRES_DSN`
- `RETRIEVAL_TOP_K` (default: `5`, range `1..50`)
- `SIDECAR_HEALTH_STRICT` (`true` で埋め込み実行までチェック)

`start-dev.ps1` 用の任意上書き:

- `LLAMA_SERVER_EXE`
- `LLAMA_HF_REPO`
- `LLAMA_HF_FILE`
- `LLAMA_MODEL_PATH`
- `LLAMA_NGL`
- `LLAMA_CONTEXT`
- `LLAMA_POOLING`
- `SIDECAR_PYTHON_EXE`

## テストと検証

Go:

```powershell
cd go-backend
go test ./...
```

契約更新時の確認:

1. `contracts/memory.proto` を更新
2. `go-backend/internal/memory/memorypb/*` を再生成
3. `python-sidecar/src/grpc_gen/*` を再生成

## 実装メモ

- Go `/status` は 30 秒キャッシュです
- Go `/chat` のメモリ保存は生成後に同期実行されます
- sidecar gRPC `Recall` の `method` (`rag|llm`) はリクエスト単位で適用されます（空/不正値は `rag`）
- sidecar gRPC `Memorize` は `success=true` でも `saved=false` のスキップを返す場合があります
- フロントの API ベース URL は `VITE_API_BASE_URL` (default: `http://localhost:8000`)
