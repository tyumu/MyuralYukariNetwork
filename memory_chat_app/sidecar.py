from __future__ import annotations

from typing import Any, Literal

from pydantic import BaseModel, Field

from .bootstrap import build_memory_service
from .config import AppConfig


class MemorizeRequest(BaseModel):
    """Go連携向けの最小メモリ保存リクエスト。"""

    user_id: str = Field(default="default", description="User identifier for memory scope")
    text: str = Field(..., description="Text content to store")
    memory_type: str = Field(default="event", description="MemU memory_type")
    category: str = Field(default="conversation", description="Target memory category")


class RecallRequest(BaseModel):
    """Go連携向けの最小メモリ検索リクエスト。"""

    user_id: str = Field(default="default", description="User identifier for memory scope")
    query: str = Field(..., description="Natural language query")
    top_k: int = Field(default=5, ge=1, le=50, description="Max results")


class MemorizeResourceRequest(BaseModel):
    """リソースURLをMemUのmemorizeパイプラインで取り込むリクエスト。"""

    user_id: str = Field(default="default", description="User identifier for memory scope")
    resource_url: str = Field(..., description="Resource URL or local file path")
    modality: str = Field(default="conversation", description="conversation/document/image/audio/video")


class RecallAdvancedRequest(BaseModel):
    """詳細検索オプションを受け取るリクエスト。"""

    user_id: str = Field(default="default", description="User identifier for memory scope")
    queries: list[dict[str, Any]] = Field(..., description="MemU retrieve queries payload")
    where: dict[str, Any] | None = Field(default=None, description="Additional scope filters")
    method: Literal["rag", "llm"] = Field(default="rag", description="Retrieve workflow method")
    top_k: int = Field(default=8, ge=1, le=100, description="Max items to keep in response")


class RecallAutoRequest(BaseModel):
    """入力文から自動で検索要否を判定し、必要なら検索を実行するリクエスト。"""

    user_id: str = Field(default="default", description="User identifier for memory scope")
    query: str = Field(..., description="User input text")
    context_queries: list[dict[str, Any]] = Field(default_factory=list, description="Optional prior query context")
    where: dict[str, Any] | None = Field(default=None, description="Additional scope filters")
    method: Literal["rag", "llm"] = Field(default="llm", description="Retrieve workflow method")
    top_k: int = Field(default=8, ge=1, le=100, description="Max items to keep in response")


class CreateCategoryRequest(BaseModel):
    """カテゴリ作成リクエスト。"""

    user_id: str = Field(default="default", description="User identifier for memory scope")
    name: str = Field(..., description="Category name")
    description: str = Field(default="", description="Category description")


def create_memu_sidecar_app():
    """非Pythonバックエンドから呼び出すための軽量MemU APIを作成する。"""
    try:
        from fastapi import Body, FastAPI, HTTPException, Query
    except Exception as exc:
        raise RuntimeError("FastAPI mode requires fastapi and uvicorn") from exc

    cfg = AppConfig.from_env()
    service = build_memory_service(cfg)

    app = FastAPI(
        title="MemU Sidecar API",
        version="1.0.0",
        description="Lightweight local sidecar for memory write/read operations.",
        docs_url=None,
        redoc_url=None,
    )

    async def run_retrieve_with_method(
        *,
        queries: list[dict[str, Any]],
        where: dict[str, Any] | None,
        method: Literal["rag", "llm"],
    ) -> dict[str, Any]:
        """指定methodでretrieveワークフローを実行する。"""
        if not queries:
            raise HTTPException(status_code=400, detail="queries must not be empty")

        ctx = service._get_context()
        store = service._get_database()
        original_query = service._extract_query_text(queries[-1])
        where_filters = service._normalize_where(where)
        context_queries_objs = queries[:-1] if len(queries) > 1 else []

        workflow_name = "retrieve_llm" if method == "llm" else "retrieve_rag"
        state: dict[str, Any] = {
            "method": method,
            "original_query": original_query,
            "context_queries": context_queries_objs,
            "route_intention": service.retrieve_config.route_intention,
            "skip_rewrite": len(queries) == 1,
            "retrieve_category": service.retrieve_config.category.enabled,
            "retrieve_item": service.retrieve_config.item.enabled,
            "retrieve_resource": service.retrieve_config.resource.enabled,
            "sufficiency_check": service.retrieve_config.sufficiency_check,
            "ctx": ctx,
            "store": store,
            "where": where_filters,
        }

        result = await service._run_workflow(workflow_name, state)
        response = result.get("response")
        if response is None:
            return {
                "needs_retrieval": False,
                "original_query": original_query,
                "rewritten_query": original_query,
                "next_step_query": None,
                "categories": [],
                "items": [],
                "resources": [],
            }
        if not isinstance(response, dict):
            raise HTTPException(status_code=500, detail="Invalid retrieve response")
        return response

    def trim_retrieve_response(result: dict[str, Any], top_k: int) -> dict[str, Any]:
        """レスポンス内の配列フィールドをtop_kで制限する。"""
        trimmed = dict(result)
        for key in ("categories", "items", "resources"):
            value = trimmed.get(key)
            if isinstance(value, list):
                trimmed[key] = value[:top_k]
        return trimmed

    # ヘルスチェック用: サーバー生存状態と有効カテゴリ一覧を返す。
    @app.get("/health")
    async def health() -> dict[str, Any]:
        return {
            "status": "ok",
            "categories": [c.name for c in cfg.chat_memory_categories],
        }

    # メモリ保存API: テキストを指定カテゴリ・指定user_idスコープで保存する。
    # Go側は会話後にこのAPIを呼び、永続記憶への書き込みを行う。
    @app.post("/memorize")
    async def memorize(req: MemorizeRequest = Body(...)) -> dict[str, Any]:
        try:
            result = await service.create_memory_item(
                memory_type=req.memory_type,
                memory_content=req.text,
                memory_categories=[req.category],
                user={"user_id": req.user_id},
            )
            return {
                "ok": True,
                "memory_item": result.get("memory_item", {}),
            }
        except Exception as exc:
            raise HTTPException(status_code=500, detail=str(exc)) from exc

    # メモリ検索API: user_idスコープで関連記憶を検索し、上位top_k件を返す。
    # Go側はLLM応答生成前にこのAPIを呼び、文脈として利用する。
    @app.post("/recall")
    async def recall(req: RecallRequest = Body(...)) -> dict[str, Any]:
        try:
            result = await service.retrieve(
                queries=[{"role": "user", "content": req.query}],
                where={"user_id": req.user_id},
            )
            items = result.get("items", [])[: req.top_k]
            return {
                "ok": True,
                "items": items,
            }
        except Exception as exc:
            raise HTTPException(status_code=500, detail=str(exc)) from exc

    # リソース取り込みAPI: 会話JSONや文書/画像をMemUのmemorizeパイプラインで処理する。
    @app.post("/memorize_resource")
    async def memorize_resource(req: MemorizeResourceRequest = Body(...)) -> dict[str, Any]:
        try:
            result = await service.memorize(
                resource_url=req.resource_url,
                modality=req.modality,
                user={"user_id": req.user_id},
            )
            return {
                "ok": True,
                "result": result,
            }
        except Exception as exc:
            raise HTTPException(status_code=500, detail=str(exc)) from exc

    # 高度検索API: queries/whereをほぼ透過で受け取り、retrieve結果を返す。
    @app.post("/recall_advanced")
    async def recall_advanced(req: RecallAdvancedRequest = Body(...)) -> dict[str, Any]:
        try:
            where_filters: dict[str, Any] = {"user_id": req.user_id}
            if req.where:
                where_filters.update(req.where)

            result = await run_retrieve_with_method(
                queries=req.queries,
                where=where_filters,
                method=req.method,
            )

            return {
                "ok": True,
                "result": trim_retrieve_response(result, req.top_k),
            }
        except Exception as exc:
            raise HTTPException(status_code=500, detail=str(exc)) from exc

    # 自動判定検索API: pre_retrieval_decisionで検索要否を判定し、必要時のみretrieveする。
    @app.post("/recall_auto")
    async def recall_auto(req: RecallAutoRequest = Body(...)) -> dict[str, Any]:
        try:
            where_filters: dict[str, Any] = {"user_id": req.user_id}
            if req.where:
                where_filters.update(req.where)

            needs_retrieval, rewritten_query = await service._decide_if_retrieval_needed(
                req.query,
                req.context_queries,
            )

            if not needs_retrieval:
                return {
                    "ok": True,
                    "decision": {
                        "needs_retrieval": False,
                        "original_query": req.query,
                        "rewritten_query": rewritten_query,
                    },
                    "result": {
                        "needs_retrieval": False,
                        "original_query": req.query,
                        "rewritten_query": rewritten_query,
                        "next_step_query": None,
                        "categories": [],
                        "items": [],
                        "resources": [],
                    },
                }

            queries = [*req.context_queries, {"role": "user", "content": rewritten_query}]
            retrieve_result = await run_retrieve_with_method(
                queries=queries,
                where=where_filters,
                method=req.method,
            )

            return {
                "ok": True,
                "decision": {
                    "needs_retrieval": True,
                    "original_query": req.query,
                    "rewritten_query": rewritten_query,
                },
                "result": trim_retrieve_response(retrieve_result, req.top_k),
            }
        except Exception as exc:
            raise HTTPException(status_code=500, detail=str(exc)) from exc

    # カテゴリ一覧API: 現在のuser_idスコープでカテゴリを列挙する。
    @app.get("/categories")
    async def list_categories(
        user_id: str = Query(default="default", description="User identifier for memory scope"),
    ) -> dict[str, Any]:
        try:
            result = await service.list_memory_categories(where={"user_id": user_id})
            return {
                "ok": True,
                "categories": result.get("categories", []),
            }
        except Exception as exc:
            raise HTTPException(status_code=500, detail=str(exc)) from exc

    # カテゴリ作成API: embeddingを生成してカテゴリを作成/再利用し、コンテキストにも反映する。
    @app.post("/categories")
    async def create_category(req: CreateCategoryRequest = Body(...)) -> dict[str, Any]:
        try:
            name = req.name.strip()
            if not name:
                raise HTTPException(status_code=400, detail="category name is required")

            desc = req.description.strip()
            user_scope = {"user_id": req.user_id}
            ctx = service._get_context()
            store = service._get_database()
            await service._ensure_categories_ready(ctx, store, user_scope)

            category_text = f"{name}: {desc}" if desc else name
            embedding = (await service._get_llm_client("embedding").embed([category_text]))[0]
            cat = store.memory_category_repo.get_or_create_category(
                name=name,
                description=desc,
                embedding=embedding,
                user_data=user_scope,
            )

            key = name.lower()
            if key not in ctx.category_name_to_id:
                ctx.category_name_to_id[key] = cat.id
                ctx.category_ids.append(cat.id)

            return {
                "ok": True,
                "category": service._model_dump_without_embeddings(cat),
            }
        except HTTPException:
            raise
        except Exception as exc:
            raise HTTPException(status_code=500, detail=str(exc)) from exc

    return app
