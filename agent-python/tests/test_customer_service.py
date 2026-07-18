from __future__ import annotations

import unittest

from model.agent_models import AnswerRequest, AnswerResponse, ModelUsage
from model.requirement_analysis_models import ExtractionResult
from service.customer_service import CustomerService
from service.prompt_service import PromptService
from service.requirement_analysis_service import RequirementAnalysisService
from service.workflow_catalog import default_workflows


class QueueExtractor:
    def __init__(self, results):
        self.results = list(results)

    def extract(self, user_text, workflows):
        return self.results.pop(0)


class FakeModelClient:
    def __init__(self):
        self.calls = 0

    def complete(self, system_prompt, user_prompt):
        self.calls += 1
        return AnswerResponse(
            answer="已收集您的需求，后续需要查询机构产品库并完成适当性校验。",
            usage=ModelUsage("deepseek", "deepseek-chat", 10, 0, 5, 15, 20),
        )


class CustomerServiceMultiTurnTest(unittest.TestCase):
    def test_product_workflow_collects_state_across_three_turns(self) -> None:
        extractor = QueueExtractor([
            ExtractionResult("product_recommendation", 0.95, {"risk_tolerance": "conservative"}),
            ExtractionResult("product_recommendation", 0.96, {"investment_period_months": 6}),
            ExtractionResult("product_recommendation", 0.97, {"investment_amount": 100000}),
        ])
        model_client = FakeModelClient()
        service = CustomerService(
            PromptService(), model_client, RequirementAnalysisService(extractor, default_workflows())
        )

        first = service.answer(AnswerRequest("r1", "u1", "s1", "我想买稳健型产品"))
        self.assertIn("多长时间", first.answer)
        self.assertEqual("conservative", first.dialogue_state["slots"]["risk_tolerance"])

        second = service.answer(AnswerRequest(
            "r2", "u1", "s1", "半年", dialogue_state=first.dialogue_state
        ))
        self.assertIn("金额", second.answer)
        self.assertEqual(6, second.dialogue_state["slots"]["investment_period_months"])

        third = service.answer(AnswerRequest(
            "r3", "u1", "s1", "十万元", dialogue_state=second.dialogue_state
        ))
        self.assertTrue(third.clear_dialogue_state)
        self.assertEqual({}, third.dialogue_state)
        self.assertEqual(1, model_client.calls)


if __name__ == "__main__":
    unittest.main()
