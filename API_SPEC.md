# API 仕様

この仕様は現在の実装に合わせています。

- Go 契約: `go-backend/shared/schemas.go`
- Python 契約: `python-sidecar/src/contracts.py`
- gRPC 契約: `contracts/memory.proto`

## Go バックエンド API (`http://localhost:8000`)

### `POST /chat`

リクエスト:

```json
{
  "user_id": "user_123",
  "message": "私の好みを覚えてる?",
  "session_id": "session_456"
}
```

成功レスポンス (`200 OK`):

```json
{
  "user_id": "user_123",
  "message": {
    "role": "assistant",
    "content": "はい、覚えています。",
    "id": "msg_1712476800000",
    "timestamp_ms": 1712476800000
  },
  "session_id": "session_456",
  "metadata": {
    "memory_context": "[Retrieved 2 memory items]\\n- [conversation] ...",
    "memory_recall": {
      "status": "ok"
    },
    "memory_write": {
      "status": "ok",
      "saved": true
    },
    "model": "unsloth/gemma-4-E4B-it-GGUF",
    "timestamp": "2026-04-07T10:00:00Z"
  }
}
```

`metadata.memory_recall.status`:

- `ok`: メモリ取得成功
- `empty`: 取得 0 件
- `error`: 取得失敗 (`error` が付与される)

`metadata.memory_write.status`:

- `ok`: メモリ保存成功
- `skipped`: `saved=false` で保存スキップ (`skip_reason` が付く場合あり)
- `error`: 保存失敗 (`error` が付与される)

実装上の処理順:

1. sidecar gRPC `Recall` を IPC で呼ぶ (`top_k` は `RETRIEVAL_TOP_K`)
2. 取得結果を system prompt に整形
3. llama.cpp の `POST /v1/chat/completions` を呼ぶ
4. 生成後に sidecar gRPC `Memorize` を IPC で実行
5. メタデータ付きで返却

補足:

- 現在の Go `/chat` 実装では `RecallRequest.method` は常に `rag` 固定です

エラー系:

- `400`: リクエスト JSON 不正 (`"Invalid request"`)
- `405`: メソッド不正 (`"Method not allowed"`)
- `502`: LLM サービス異常

```json
{ "error": "LLM service unavailable" }
```

- `504`: 生成タイムアウト

```json
{ "error": "Generation timeout" }
```

### `GET /status`

レスポンス (`200 OK`):

```json
{
  "status": "healthy",
  "services": {
    "memory": "ok",
    "llama_cpp": "ok"
  },
  "timestamp_ms": 1712476800000,
  "logs": [
    "INFO 2026-04-07 10:00:00.000: chat request user_id=user_123"
  ]
}
```

`status` は次のいずれか:

- `healthy`: すべて `ok`
- `degraded`: 一部のみ `ok`
- `error`: すべて `error`

補足:

- Go 側で `/status` は 30 秒キャッシュされます
- memory 判定は sidecar gRPC `Health`、llama 判定は `GET /v1/models` を使います

## Go - Sidecar gRPC IPC

Go backend と sidecar の本番経路は gRPC + Protocol Buffers です。

- Windows dev default (`env.ps1`): `127.0.0.1:50051`
- Linux/macOS helper default (`env.sh`): `unix:///tmp/myural_yukari_memory.sock`
- 設定キー: `MEMORY_GRPC_ENDPOINT`

RPC メソッド:

- `memory.v1.MemoryService/Memorize`
- `memory.v1.MemoryService/Recall`
- `memory.v1.MemoryService/Health`

`RecallRequest.method`:

- `rag`: embedding ベースの retrieval ワークフロー
- `llm`: LLM ベースの retrieval ワークフロー
- その他/空文字: `rag` として扱われます

`Health` レスポンス例:

```json
{
  "healthy": true,
  "status": "ok",
  "error": ""
}
```

`Health` 異常例:

```json
{
  "healthy": false,
  "status": "error",
  "error": "Memory service unavailable"
}
```

備考:

- ヘルス結果は sidecar 内で 20 秒キャッシュされます
- `SIDECAR_HEALTH_STRICT=true` の場合、埋め込み実行まで検証します

## llama.cpp API (`http://localhost:11434/v1`)

### `GET /models`

OpenAI 互換のモデル一覧です。`data[].id` を `CHAT_MODEL` に設定します。

```json
{
  "object": "list",
  "data": [
    {
      "id": "unsloth/gemma-4-E4B-it-GGUF",
      "object": "model"
    }
  ]
}
```

### `POST /chat/completions`

リクエスト:

```json
{
  "model": "unsloth/gemma-4-E4B-it-GGUF",
  "messages": [
    {"role": "system", "content": "You are helpful."},
    {"role": "user", "content": "こんにちは"}
  ],
  "stream": false,
  "temperature": 0.3
}
```

レスポンス例:

```json
{
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": "こんにちは。"
      }
    }
  ]
}
```

## タイムアウト設定 (現実装)

- Go `/chat` コンテキスト: `180s`
- Go -> sidecar gRPC 初回接続(dial) timeout: `10s`
- Go -> llama.cpp HTTP client timeout: `5m`
- Go `/status` チェックコンテキスト: `5s`
