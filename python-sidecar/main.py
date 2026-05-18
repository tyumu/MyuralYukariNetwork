"""
Python MemU Sidecar API

gRPC IPC server for MemU memory operations.
All memory operations and health checks are exposed over gRPC only.
"""

import asyncio
import json
import sys
import tempfile
import time
from pathlib import Path
from typing import Any

from google.protobuf.json_format import MessageToDict
import grpc

# Ensure generated grpc stubs are importable.
GRPC_GEN_DIR = Path(__file__).resolve().parent / "src" / "grpc_gen"
if str(GRPC_GEN_DIR) not in sys.path:
    sys.path.insert(0, str(GRPC_GEN_DIR))

import memory_pb2
import memory_pb2_grpc

from src.contracts import MemorizeReq, MemorizeResp, RecallItem, RecallReq, RecallResp
from src.config import AppConfig, build_memory_service
from src.logger import setup_logger
from src.memu_payloads import build_conversation_payload, build_retrieve_queries

# Load config
try:
    config = AppConfig.from_env()
    logger = setup_logger(config.log_level, config.dev_mode)
except Exception as e:
    print(f"Failed to load config: {e}")
    raise

HEALTH_CHECK_TTL_SECONDS = 20

_memory_service = None
_memory_service_error = None
_health_cache = None
_recall_method_lock = asyncio.Lock()


def get_memory_service():
    """Return a cached MemU service instance, initializing it on demand."""
    global _memory_service
    global _memory_service_error

    if _memory_service is not None:
        return _memory_service

    try:
        service = build_memory_service(config)
    except Exception as e:
        _memory_service_error = str(e)
        logger.error(f"Failed to initialize memory service: {e}")
        return None

    _memory_service = service
    _memory_service_error = None
    logger.info("Memory service initialized")
    return service


def _unix_socket_path(endpoint: str) -> str | None:
    if endpoint.startswith("unix://"):
        return endpoint[len("unix://") :]
    if endpoint.startswith("unix:"):
        return endpoint[len("unix:") :]
    return None


def _cleanup_unix_socket(endpoint: str) -> None:
    socket_path = _unix_socket_path(endpoint)
    if not socket_path:
        return

    sock = Path(socket_path)
    sock.parent.mkdir(parents=True, exist_ok=True)
    if sock.exists():
        sock.unlink(missing_ok=True)


async def _handle_memorize(req: MemorizeReq) -> MemorizeResp:
    """Store text in memory via MemU."""
    try:
        memory_service = get_memory_service()
        if memory_service is None:
            return MemorizeResp(
                user_id=req.user_id,
                item_id="",
                category=req.category,
                success=False,
                saved=False,
                error="Memory service unavailable",
            )

        logger.info(
            f"Memorize request: user_id={req.user_id}, category={req.category}, text_len={len(req.text)}"
        )

        # MemU expects a resource URL. Persist the chat turn as a conversation JSON file.
        temp_path: Path | None = None
        try:
            conversation_payload = build_conversation_payload(req.text, req.assistant_text)

            with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False, encoding="utf-8") as f:
                json.dump(conversation_payload, f, ensure_ascii=False)
                temp_path = Path(f.name)

            result = await memory_service.memorize(
                resource_url=temp_path.as_posix(),
                modality="conversation",
                user={"user_id": req.user_id},
            )
        finally:
            if temp_path and temp_path.exists():
                temp_path.unlink(missing_ok=True)

        items = result.get("items", []) if isinstance(result, dict) else []
        if len(items) == 0:
            logger.info("Memorize skipped: no items created")
            return MemorizeResp(
                user_id=req.user_id,
                item_id="",
                category=req.category,
                success=True,
                saved=False,
                skip_reason="no_memory_items_created",
            )

        item_id = items[0].get("id", "")
        logger.info(f"Memorize success: item_id={item_id}, items_created={len(items)}")
        return MemorizeResp(
            user_id=req.user_id,
            item_id=item_id,
            category=req.category,
            success=True,
            saved=True,
        )
    except Exception as e:
        logger.error(f"Memorize failed: {e}")
        return MemorizeResp(
            user_id=req.user_id,
            item_id="",
            category=req.category,
            success=False,
            saved=False,
            error=str(e),
        )


async def _handle_recall(req: RecallReq) -> RecallResp:
    """Retrieve relevant memories via MemU."""
    try:
        memory_service = get_memory_service()
        if memory_service is None:
            return RecallResp(
                user_id=req.user_id,
                items=[],
                success=False,
                error="Memory service unavailable",
            )

        method = req.method if req.method in ("rag", "llm") else "rag"

        logger.info(
            f"Recall request: user_id={req.user_id}, query={req.query[:50]}..., method={method}, top_k={req.top_k}"
        )

        # MemU retrieve expects a query list and optional scope filters.
        where = dict(req.where or {})
        where["user_id"] = req.user_id
        result = await _retrieve_with_method_override(
            memory_service=memory_service,
            method=method,
            queries=build_retrieve_queries(req.query),
            where=where,
        )

        items = [
            RecallItem(
                id=r.get("id", ""),
                content=r.get("content", r.get("summary", "")),
                category=r.get("category", r.get("memory_type", "")),
                salience=r.get("salience", 0.0),
            )
            for r in (result.get("items", []) if isinstance(result, dict) else [])
        ][: req.top_k]

        logger.info(f"Recall success: retrieved {len(items)} items")
        return RecallResp(user_id=req.user_id, items=items, success=True)
    except Exception as e:
        logger.error(f"Recall failed: {e}")
        return RecallResp(
            user_id=req.user_id,
            items=[],
            success=False,
            error=str(e),
        )


async def _retrieve_with_method_override(
    memory_service: Any,
    method: str,
    queries: list[dict[str, Any]],
    where: dict[str, Any],
) -> dict[str, Any]:
    """Run MemU retrieve using per-request method while preserving shared service state."""
    retrieve_config = getattr(memory_service, "retrieve_config", None)
    if retrieve_config is None or not hasattr(retrieve_config, "method"):
        # Backward-compatible path for mocked or legacy services.
        return await memory_service.retrieve(queries=queries, where=where)

    async with _recall_method_lock:
        original_method = getattr(retrieve_config, "method", "rag")
        retrieve_config.method = method
        try:
            return await memory_service.retrieve(queries=queries, where=where)
        finally:
            retrieve_config.method = original_method


async def _health_payload() -> tuple[bool, dict[str, Any]]:
    global _health_cache

    now = time.monotonic()
    cached = _health_cache
    if cached and (now - cached.get("checked_at", 0) < HEALTH_CHECK_TTL_SECONDS):
        return bool(cached.get("ok")), dict(cached.get("payload", {}))

    memory_service = get_memory_service()
    if memory_service is None:
        payload = {
            "status": "error",
            "service": "memu-sidecar",
            "version": "1.1.0",
            "memory_service": "error",
            "error": _memory_service_error or "Memory service unavailable",
        }
        _health_cache = {"checked_at": now, "ok": False, "payload": payload}
        return False, payload

    if config.sidecar_health_strict:
        try:
            # Strict mode validates embedding runtime compatibility.
            embed_client = memory_service._get_llm_client("embedding")
            await embed_client.embed(["health-check"])
        except Exception as exc:
            payload = {
                "status": "error",
                "service": "memu-sidecar",
                "version": "1.1.0",
                "memory_service": "error",
                "error": str(exc),
            }
            _health_cache = {"checked_at": now, "ok": False, "payload": payload}
            return False, payload

    payload = {
        "status": "ok",
        "service": "memu-sidecar",
        "version": "1.1.0",
        "memory_service": "active",
        "embedding_check": "enabled" if config.sidecar_health_strict else "skipped",
        "memory_grpc_endpoint": config.memory_grpc_endpoint,
    }
    _health_cache = {"checked_at": now, "ok": True, "payload": payload}
    return True, payload


class MemoryGrpcService(memory_pb2_grpc.MemoryServiceServicer):
    async def Memorize(self, request, context):
        resp = await _handle_memorize(
            MemorizeReq(
                user_id=request.user_id or "default",
                text=request.text,
                assistant_text=request.assistant_text,
                memory_type=request.memory_type or "event",
                category=request.category or "conversation",
            )
        )
        return memory_pb2.MemorizeResponse(
            user_id=resp.user_id,
            item_id=resp.item_id,
            category=resp.category,
            success=resp.success,
            saved=resp.saved,
            skip_reason=resp.skip_reason or "",
            error=resp.error or "",
        )

    async def Recall(self, request, context):
        where = None
        if request.HasField("where"):
            where = MessageToDict(request.where)

        top_k = request.top_k if request.top_k > 0 else config.retrieval_top_k
        method = request.method if request.method in ("rag", "llm") else "rag"

        resp = await _handle_recall(
            RecallReq(
                user_id=request.user_id or "default",
                query=request.query,
                top_k=top_k,
                where=where,
                method=method,
            )
        )

        return memory_pb2.RecallResponse(
            user_id=resp.user_id,
            items=[
                memory_pb2.RecallItem(
                    id=item.id,
                    content=item.content,
                    category=item.category,
                    salience=float(item.salience),
                )
                for item in resp.items
            ],
            success=resp.success,
            error=resp.error or "",
        )

    async def Health(self, request, context):
        del request
        healthy, payload = await _health_payload()
        return memory_pb2.HealthResponse(
            healthy=healthy,
            status=payload.get("status", "unknown"),
            error=payload.get("error", ""),
        )


async def serve() -> None:
    endpoint = config.memory_grpc_endpoint
    _cleanup_unix_socket(endpoint)

    grpc_server = grpc.aio.server()
    memory_pb2_grpc.add_MemoryServiceServicer_to_server(MemoryGrpcService(), grpc_server)
    bound_port = grpc_server.add_insecure_port(endpoint)
    if bound_port == 0:
        raise RuntimeError(f"failed to bind gRPC endpoint: {endpoint}")

    await grpc_server.start()
    logger.info(f"gRPC sidecar listening on {endpoint}")
    logger.info("MemU Sidecar gRPC server started")

    try:
        await grpc_server.wait_for_termination()
    finally:
        await grpc_server.stop(grace=2)
        _cleanup_unix_socket(endpoint)
        logger.info("MemU Sidecar gRPC server stopped")


if __name__ == "__main__":
    logger.info(f"Starting MemU Sidecar gRPC IPC on {config.memory_grpc_endpoint}")
    try:
        asyncio.run(serve())
    except KeyboardInterrupt:
        logger.info("MemU Sidecar interrupted and shutting down")
