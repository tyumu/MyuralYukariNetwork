import unittest

from pydantic import ValidationError

from src.contracts import MemorizeResp, RecallReq


class ContractsSmokeTests(unittest.TestCase):
    def test_recall_default_top_k(self) -> None:
        req = RecallReq(user_id="u1", query="hello")
        self.assertEqual(req.top_k, 5)

    def test_memorize_response_requires_saved(self) -> None:
        with self.assertRaises(ValidationError):
            MemorizeResp(
                user_id="u1",
                item_id="",
                category="conversation",
                success=True,
            )

    def test_memorize_skip_payload_is_valid(self) -> None:
        resp = MemorizeResp(
            user_id="u1",
            item_id="",
            category="conversation",
            success=True,
            saved=False,
            skip_reason="no_memory_items_created",
        )
        self.assertFalse(resp.saved)
        self.assertEqual(resp.skip_reason, "no_memory_items_created")


if __name__ == "__main__":
    unittest.main()
