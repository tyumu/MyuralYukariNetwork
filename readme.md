# みゅーらる ゆかり ねっとわーく (MYN)

## 【コンセプト】

「十人十色の結月ゆかり」を育成・保存・共有する。
ユーザーとの対話を通じて、デフォルトのゆかりさんが独自の性格と記憶を獲得していき「〇〇（マスター名）式ゆかり」へと分岐（Fork）していく。

## 開発ドキュメント入口

- セットアップ手順: [SETUP.md](SETUP.md)
- 開発クイックリファレンス: [DEV_GUIDE.md](DEV_GUIDE.md)
- システム構成: [ARCHITECTURE.md](ARCHITECTURE.md)
- API 仕様: [API_SPEC.md](API_SPEC.md)
- 開発ルール: [DEVELOPMENT_RULES.md](DEVELOPMENT_RULES.md)

## 最短起動（Windows）

```powershell
cd <MyuralYukariNetwork のクローン先>
. .\env.ps1
.\scripts\start-dev.ps1
```

停止:

```powershell
.\scripts\stop-dev.ps1
```

## サービス構成

1. llama.cpp (`11434`)
2. Python Sidecar (gRPC: `MEMORY_GRPC_ENDPOINT`, Windows 既定 `127.0.0.1:50051`)
3. Go Backend (`8000`)
4. Tauri Frontend (`1420`)