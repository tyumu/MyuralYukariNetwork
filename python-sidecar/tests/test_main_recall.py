import unittest
from types import SimpleNamespace
from unittest.mock import AsyncMock, patch

import main
from src.contracts import RecallReq


class RecallQueryShapeTests(unittest.IsolatedAsyncioTestCase):
    async def test_recall_uses_text_object_query(self) -> None:
        memory_service = AsyncMock()
        memory_service.retrieve = AsyncMock(return_value={"items": []})

        req = RecallReq(
            user_id="u1",
            query="What did I say before?",
            top_k=3,
            where={"tenant": "local-dev"},
            method="rag",
        )

        with patch("main.get_memory_service", return_value=memory_service):
            resp = await main._handle_recall(req)

        self.assertTrue(resp.success)
        memory_service.retrieve.assert_awaited_once()

        kwargs = memory_service.retrieve.await_args.kwargs
        self.assertEqual(
            kwargs["queries"],
            [{"role": "user", "content": {"text": "What did I say before?"}}],
        )
        self.assertEqual(kwargs["where"]["user_id"], "u1")
        self.assertEqual(kwargs["where"]["tenant"], "local-dev")

    async def test_recall_applies_requested_method_and_restores_config(self) -> None:
        class DummyMemoryService:
            def __init__(self) -> None:
                self.retrieve_config = SimpleNamespace(method="rag")
                self.seen_methods: list[str] = []

            async def retrieve(self, *, queries, where):
                _ = queries
                _ = where
                self.seen_methods.append(self.retrieve_config.method)
                return {"items": []}

        memory_service = DummyMemoryService()
        req = RecallReq(
            user_id="u1",
            query="Need deep reasoning",
            top_k=3,
            where=None,
            method="llm",
        )

        with patch("main.get_memory_service", return_value=memory_service):
            resp = await main._handle_recall(req)

        self.assertTrue(resp.success)
        self.assertEqual(memory_service.seen_methods, ["llm"])
        self.assertEqual(memory_service.retrieve_config.method, "rag")


if __name__ == "__main__":
    unittest.main()
