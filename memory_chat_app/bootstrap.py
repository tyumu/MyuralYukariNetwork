from __future__ import annotations

import sys
from pathlib import Path

from openai import AsyncOpenAI

from .config import AppConfig, MemoryCategoryDef

PROJECT_ROOT = Path(__file__).resolve().parents[1]
MEMU_SRC = PROJECT_ROOT / "memU" / "src"
if str(MEMU_SRC) not in sys.path:
    sys.path.insert(0, str(MEMU_SRC))

from memu.app import MemoryService


def _to_memu_categories(categories: list[MemoryCategoryDef]) -> list[dict[str, str]]:
    return [{"name": c.name, "description": c.description} for c in categories]


def build_memory_service(config: AppConfig) -> MemoryService:
    return MemoryService(
        llm_profiles={
            "default": {
                "client_backend": "sdk",
                "base_url": config.llm_base_url,
                "api_key": config.llm_api_key,
                "chat_model": config.chat_model,
                "embed_model": config.embed_model,
            }
        },
        memorize_config={
            "memory_categories": _to_memu_categories(config.chat_memory_categories),
        },
        database_config={
            "metadata_store": {
                "provider": "postgres",
                "dsn": config.postgres_dsn,
            }
        },
        retrieve_config={
            "item": {"top_k": config.retrieval_top_k},
        },
    )


def build_chat_client(config: AppConfig) -> AsyncOpenAI:
    return AsyncOpenAI(base_url=config.llm_base_url, api_key=config.llm_api_key)


def create_runtime() -> tuple[AppConfig, MemoryService, AsyncOpenAI]:
    config = AppConfig.from_env()
    service = build_memory_service(config)
    client = build_chat_client(config)
    return config, service, client
