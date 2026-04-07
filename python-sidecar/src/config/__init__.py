"""Configuration for Python MemU Sidecar API."""

import os
import sys
from pathlib import Path
from dataclasses import dataclass
from typing import TYPE_CHECKING
from dotenv import load_dotenv

# Resolve repository paths from this file location.
THIS_FILE = Path(__file__).resolve()
SIDECAR_ROOT = THIS_FILE.parents[2]
REPO_ROOT = SIDECAR_ROOT.parent

# Add memU src to path so we can import memu.app
MEMU_SRC = REPO_ROOT / "memU" / "src"
if str(MEMU_SRC) not in sys.path:
    sys.path.insert(0, str(MEMU_SRC))

# Load env vars from both sidecar-local and repository-root .env files.
load_dotenv(SIDECAR_ROOT / ".env")
load_dotenv(REPO_ROOT / ".env")

if TYPE_CHECKING:
    from memu.app import MemoryService


@dataclass(frozen=True)
class AppConfig:
    """Configuration loaded from environment."""
    
    memory_grpc_endpoint: str
    sidecar_health_strict: bool
    dev_mode: bool
    log_level: str

    # LLM settings
    llm_base_url: str
    llm_api_key: str
    chat_model: str
    embed_model: str

    # Database (MemU)
    postgres_dsn: str
    retrieval_top_k: int

    @classmethod
    def from_env(cls) -> "AppConfig":
        """Load configuration from environment variables."""
        chat_model = os.getenv("CHAT_MODEL", "unsloth/gemma-4-E4B-it-GGUF")
        return cls(
            memory_grpc_endpoint=os.getenv("MEMORY_GRPC_ENDPOINT", _default_memory_grpc_endpoint()),
            sidecar_health_strict=os.getenv("SIDECAR_HEALTH_STRICT", "false").lower() in ("1", "true", "yes"),
            dev_mode=os.getenv("DEV_MODE", "true").lower() in ("1", "true", "yes"),
            log_level=os.getenv("LOG_LEVEL", "info"),
            
            llm_base_url=os.getenv("LLM_BASE_URL", "http://localhost:11434/v1"),
            llm_api_key=os.getenv("LLM_API_KEY", ""),
            chat_model=chat_model,
            embed_model=os.getenv("EMBED_MODEL", "nomic-embed-text:latest-num-gpu0"),
            
            postgres_dsn=os.getenv("POSTGRES_DSN", "postgresql://user:password@localhost/memu"),
            retrieval_top_k=int(os.getenv("RETRIEVAL_TOP_K", "5")),
        )


def _default_memory_grpc_endpoint() -> str:
    if os.name == "nt":
        return "127.0.0.1:50051"
    return "unix:///tmp/myural_yukari_memory.sock"


def build_memory_service(config: AppConfig) -> "MemoryService":
    """Build and return a configured MemoryService instance."""
    from memu.app import MemoryService
    
    memory_categories = [
        {"name": "conversation", "description": "Conversation turns and short-term context."},
        {"name": "user_profile", "description": "Stable user traits and preferences."},
    ]
    
    service = MemoryService(
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
            "memory_categories": memory_categories,
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
    
    return service
