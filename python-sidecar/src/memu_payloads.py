from __future__ import annotations

from datetime import datetime, timezone
from typing import Any


def build_conversation_payload(user_text: str, assistant_text: str = "") -> dict[str, Any]:
    created_at = _now_iso()
    content = [
        {
            "role": "user",
            "content": {"text": user_text},
            "created_at": created_at,
        }
    ]
    if assistant_text.strip():
        content.append(
            {
                "role": "assistant",
                "content": {"text": assistant_text},
                "created_at": created_at,
            }
        )
    return {"content": content}


def build_retrieve_queries(query: str) -> list[dict[str, Any]]:
    return [{"role": "user", "content": {"text": query}}]


def _now_iso() -> str:
    return datetime.now(timezone.utc).isoformat()
