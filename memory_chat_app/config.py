from __future__ import annotations

import json
import os
from dataclasses import dataclass

from dotenv import load_dotenv


@dataclass(frozen=True)
class MemoryCategoryDef:
    name: str
    description: str


@dataclass(frozen=True)
class AppConfig:
    llm_base_url: str
    llm_api_key: str
    chat_model: str
    embed_model: str
    postgres_dsn: str
    chat_memory_categories: list[MemoryCategoryDef]
    retrieval_top_k: int
    memorize_via_pipeline: bool

    @classmethod
    def from_env(cls) -> "AppConfig":
        load_dotenv()

        base_categories = [
            MemoryCategoryDef("conversation", "Conversation turns and short-term context."),
            MemoryCategoryDef("user_profile", "Stable user traits and preferences."),
        ]

        persona_categories = [
            MemoryCategoryDef("ai_persona_preferences", "Preferred assistant style and tone for this user."),
            MemoryCategoryDef("ai_persona_boundaries", "Conversation boundaries and behavioral constraints."),
            MemoryCategoryDef("ai_persona_evolution", "How the assistant persona should adapt over time."),
        ]

        enable_persona = os.getenv("MEMU_ENABLE_PERSONA_CATEGORIES", "false").lower() in {
            "1",
            "true",
            "yes",
            "on",
        }

        categories = list(base_categories)
        if enable_persona:
            categories.extend(persona_categories)

        extra_json = os.getenv("MEMU_EXTRA_CATEGORIES", "").strip()
        if extra_json:
            try:
                raw = json.loads(extra_json)
                if isinstance(raw, list):
                    for item in raw:
                        if not isinstance(item, dict):
                            continue
                        name = str(item.get("name", "")).strip()
                        description = str(item.get("description", "")).strip()
                        if name:
                            categories.append(MemoryCategoryDef(name=name, description=description or "Custom category."))
            except json.JSONDecodeError:
                pass

        return cls(
            llm_base_url=os.getenv("MEMU_LLM_BASE_URL", "http://localhost:11434/v1"),
            llm_api_key=os.getenv("OPENAI_API_KEY", "dummy-key"),
            chat_model=os.getenv("MEMU_CHAT_MODEL", "gemma3:12b"),
            embed_model=os.getenv("MEMU_EMBED_MODEL", "nomic-embed-text:latest-num-gpu0"),
            postgres_dsn=os.getenv(
                "MEMU_POSTGRES_DSN",
                "postgresql+psycopg://postgres:postgres@localhost:5432/memu",
            ),
            chat_memory_categories=categories,
            retrieval_top_k=int(os.getenv("MEMU_RETRIEVE_TOP_K", "8")),
            memorize_via_pipeline=os.getenv("MEMU_MEMORIZE_VIA_PIPELINE", "false").lower() in {
                "1",
                "true",
                "yes",
                "on",
            },
        )
