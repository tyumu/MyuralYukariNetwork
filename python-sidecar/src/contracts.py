"""Pydantic models used by the gRPC sidecar handlers."""

from typing import Literal, Optional

from pydantic import BaseModel, Field


class MemorizeRequest(BaseModel):
    user_id: str = Field(default="default")
    text: str
    assistant_text: str = Field(default="")
    memory_type: str = Field(default="event")
    category: str = Field(default="conversation")


class MemorizeResponse(BaseModel):
    user_id: str
    item_id: str
    category: str
    success: bool
    saved: bool
    skip_reason: Optional[str] = None
    error: Optional[str] = None


class RecallRequest(BaseModel):
    user_id: str = Field(default="default")
    query: str
    top_k: int = Field(default=5, ge=1, le=50)
    where: Optional[dict[str, object]] = None
    method: Literal["rag", "llm"] = Field(default="rag")


class RecallItem(BaseModel):
    id: str
    content: str
    category: str
    salience: float


class RecallResponse(BaseModel):
    user_id: str
    items: list[RecallItem]
    success: bool
    error: Optional[str] = None


# Backward-compatible aliases used by existing imports.
MemorizeReq = MemorizeRequest
MemorizeResp = MemorizeResponse
RecallReq = RecallRequest
RecallResp = RecallResponse