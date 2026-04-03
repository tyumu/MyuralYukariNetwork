from __future__ import annotations

import argparse
from memory_chat_app.sidecar import create_memu_sidecar_app


def main() -> None:
    parser = argparse.ArgumentParser(description="Minimal MemU sidecar server")
    parser.add_argument("--mode", choices=["sidecar"], default="sidecar")
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=8090)
    args = parser.parse_args()

    import uvicorn

    sidecar_api = create_memu_sidecar_app()
    uvicorn.run(sidecar_api, host=args.host, port=args.port)


if __name__ == "__main__":
    main()
