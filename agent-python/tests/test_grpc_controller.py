from __future__ import annotations

import unittest

from google.protobuf import json_format, struct_pb2

from controller.grpc_controller import GRPCCustomerController
from model.agent_models import AnswerResponse, ModelUsage


class FakeCustomerService:
    def __init__(self) -> None:
        self.request = None

    def answer(self, request):
        self.request = request
        return AnswerResponse(
            answer="请继续补充投资期限。",
            usage=ModelUsage("deepseek", "deepseek-chat", 10, 0, 5, 15, 20),
            decision={
                "intent": "product_recommendation",
                "status": "need_clarification",
                "active_slot": "investment_period_months",
                "slot_updates": {"risk_tolerance": {"value": "conservative", "status": "confirmed", "source": "user", "evidence": "稳健", "confidence": 0.9}},
                "next_action": "ask_user",
                "next_question": "请继续补充投资期限。",
                "suggested_options": [],
            },
        )


class FakeContext:
    def abort(self, code, details):
        raise AssertionError(f"unexpected gRPC abort: {code} {details}")


class GRPCCustomerControllerTest(unittest.TestCase):
    def test_nested_dialogue_state_is_converted_from_protobuf_struct(self) -> None:
        service = FakeCustomerService()
        controller = GRPCCustomerController(service)
        request = struct_pb2.Struct()
        request.update({
            "request_id": "req-1",
            "user_id": "user-1",
            "session_id": "session-1",
            "question": "这笔钱暂时用不到",
            "memories": [],
            "dialogue_state": {
                "intent": "product_recommendation",
                "slots": {"risk_tolerance": "conservative"},
                "status": "need_clarification",
                "last_question": "这笔资金准备放多久？",
            },
        })

        response = controller.answer(request, FakeContext())
        payload = json_format.MessageToDict(response, preserving_proto_field_name=True)

        self.assertEqual("product_recommendation", service.request.dialogue_state["intent"])
        self.assertEqual("conservative", service.request.dialogue_state["slots"]["risk_tolerance"])
        self.assertEqual("请继续补充投资期限。", payload["answer"])
        self.assertEqual("product_recommendation", payload["decision"]["intent"])


if __name__ == "__main__":
    unittest.main()
