# アーキテクチャ

Tauri (React) + Go + Python (MemU sidecar) + llama.cpp の 4 層構成です。

## システム構成

```text
Tauri Frontend (tauri_app, port 1420)
  -> HTTP JSON
Go Backend (go-backend, port 8000)
  |- POST /chat
  |- GET  /status (30s cache)
  |- memory client -> Python sidecar (gRPC: 127.0.0.1:50051 on Windows dev)
  |- llm client    -> llama.cpp /v1 (port 11434)

Python Sidecar (python-sidecar, gRPC)
  |- gRPC MemoryService (Memorize/Recall/Health)
  -> MemU -> PostgreSQL

llama.cpp (llama-server, OpenAI compatible)
  |- GET  /v1/models
  |- POST /v1/chat/completions
```

## コンポーネント責務

### Tauri Frontend (`tauri_app/src`)

- `App.jsx`: チャット画面全体
- `components/ChatBox.jsx`: 入力と送信
- `components/MessageList.jsx`: メッセージ表示
- `components/DevLogger.jsx`: `/status` の可視化
- `api/index.js`: Go API クライアント (`/chat`, `/status`)

### Go Backend (`go-backend`)

- `cmd/server/main.go`: ルート登録、CORS、`/status` キャッシュ
- `internal/api/chat_handler.go`: `/chat` エントリーポイント
- `internal/api/chat_flow.go`: recall -> generate -> memorize のフロー
- `internal/api/status_handler.go`: memory/llama ヘルス集約
- `internal/memory/client.go`: sidecar 通信
- `internal/ollama/client.go`: llama.cpp OpenAI 互換 API 通信
- `internal/config/config.go`: 環境変数ロード/検証

### Python Sidecar (`python-sidecar`)

- `main.py`: gRPC サーバー定義
- `src/config/__init__.py`: MemU 設定と初期化
- `src/contracts.py`: Pydantic 契約
- `src/grpc_gen/*`: `contracts/memory.proto` から生成した gRPC スタブ
- `src/logger/__init__.py`: sidecar ログバッファ

## チャット処理シーケンス

1. Frontend が `POST /chat` に送信
2. Go が sidecar gRPC `Recall` を IPC で呼ぶ (`top_k=RETRIEVAL_TOP_K`)
3. Go が recall 結果を system prompt に組み込む
4. Go が llama.cpp `POST /v1/chat/completions` で生成
5. Go が sidecar gRPC `Memorize` を IPC で呼び、会話保存を試みる
6. Go が `metadata` 付き `ChatResponse` を返す

`metadata` には少なくとも以下を含みます。

- `memory_context`
- `memory_recall` (`status: ok|empty|error`)
- `memory_write` (`status: ok|skipped|error`, `saved`, `skip_reason`)
- `model`
- `timestamp`

## ヘルスとステータス

### Go `/status`

- memory: sidecar gRPC `Health` が `healthy=true` なら `ok`
- llama_cpp: `/v1/models` が `200` なら `ok`
- 総合 `status`: `healthy | degraded | error`
- レスポンスは 30 秒キャッシュ

### Sidecar gRPC `Health`

- 通常: `healthy=true`, `status="ok"`
- 初期化失敗や strict 埋め込みチェック失敗: `healthy=false`, `error` に詳細
- 20 秒キャッシュ

## データ契約の管理

契約の基準は次の 3 点です。

1. `contracts/memory.proto`
2. `go-backend/shared/schemas.go`
3. `python-sidecar/src/contracts.py`

## 起動オーケストレーション

`scripts/start-dev.ps1` は以下を実行します。

1. `env.ps1` を読み込み
2. llama.cpp 起動または既存プロセス追跡
3. sidecar の gRPC health を確認し、未起動なら起動（既存起動時は health 確認のみ）
4. Go backend 起動または既存プロセス追跡
5. `myural_yukari_tauri` の stale プロセスを停止
6. Tauri を `npm run tauri:dev` で起動
7. 追跡対象 PID を `%TEMP%/MyuralYukariNetwork-dev/processes.json` に保存

停止は `scripts/stop-dev.ps1` で追跡 PID をまとめて停止します。
追跡 state がない場合でも、主要プロセスに対する best-effort cleanup を行います。

## 現時点の制約

- Go `/chat` はメモリ保存を同期実行するため、保存遅延が応答時間に影響します
- sidecar gRPC `Recall` の `method` (`rag|llm`) はリクエスト単位で適用され、該当 retrieval ワークフローが選択されます
- Windows 主体運用のため、`Makefile` コマンドは補助扱いです
