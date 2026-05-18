import unittest
from datetime import datetime

from src.memu_payloads import build_conversation_payload, build_retrieve_queries


class MemuPayloadTests(unittest.TestCase):
    def test_build_conversation_payload_adds_iso_timestamp(self) -> None:
        payload = build_conversation_payload("hello", "hi there")

        self.assertEqual(len(payload["content"]), 2)
        self.assertEqual(payload["content"][0]["role"], "user")
        self.assertEqual(payload["content"][1]["role"], "assistant")
        for message in payload["content"]:
            self.assertIn("created_at", message)
            datetime.fromisoformat(message["created_at"])

    def test_build_conversation_payload_skips_blank_assistant(self) -> None:
        payload = build_conversation_payload("hello", "   ")
        self.assertEqual(len(payload["content"]), 1)

    def test_build_retrieve_queries_uses_text_object(self) -> None:
        self.assertEqual(
            build_retrieve_queries("what do you remember?"),
            [{"role": "user", "content": {"text": "what do you remember?"}}],
        )


if __name__ == "__main__":
    unittest.main()
