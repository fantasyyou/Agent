from __future__ import annotations

import unittest

from client.deepseek_requirement_extractor import DeepSeekRequirementExtractor
from model.requirement_analysis_models import (
    ANALYSIS_STATUS_NEED_CLARIFICATION,
    ANALYSIS_STATUS_READY,
    NEXT_ACTION_ASK_USER,
    NEXT_ACTION_ROUTE_WORKFLOW,
    ExtractionResult,
)
from service.requirement_analysis_service import RequirementAnalysisService
from service.workflow_catalog import default_workflows
from service.question_policy import QuestionPolicy


class FakeExtractor:
    """返回预设提取结果，确保单元测试不调用真实大模型。"""

    def __init__(self, result: ExtractionResult) -> None:
        self.result = result

    def extract(self, user_text, workflows):
        return self.result


class RequirementAnalysisServiceTest(unittest.TestCase):
    def test_missing_slots_returns_first_fixed_question(self) -> None:
        extractor = FakeExtractor(ExtractionResult(
            intent="product_recommendation",
            confidence=0.92,
            slots={"investment_amount": 50000},
        ))
        result = RequirementAnalysisService(extractor, default_workflows()).analyze("我想拿五万元买点理财")

        self.assertEqual(ANALYSIS_STATUS_NEED_CLARIFICATION, result.status)
        self.assertEqual(NEXT_ACTION_ASK_USER, result.next_action)
        self.assertEqual(["risk_tolerance", "investment_period_months"], result.missing_slots)
        self.assertIn("风险承受能力", result.next_question)
        self.assertEqual(["稳健型", "平衡型", "进取型", "暂不确定"], result.suggested_options)

    def test_complete_slots_can_route_workflow(self) -> None:
        extractor = FakeExtractor(ExtractionResult(
            intent="product_recommendation",
            confidence=0.96,
            slots={
                "risk_tolerance": "conservative",
                "investment_amount": 100000,
                "investment_period_months": 6,
            },
        ))
        result = RequirementAnalysisService(extractor, default_workflows()).analyze("十万元半年不用，我偏稳健")

        self.assertEqual(ANALYSIS_STATUS_READY, result.status)
        self.assertEqual(NEXT_ACTION_ROUTE_WORKFLOW, result.next_action)
        self.assertEqual([], result.missing_slots)
        self.assertEqual("", result.next_question)

    def test_low_confidence_asks_user_to_choose_intent(self) -> None:
        extractor = FakeExtractor(ExtractionResult(intent="fee_query", confidence=0.4, slots={}))
        result = RequirementAnalysisService(extractor, default_workflows()).analyze("这个怎么办")

        self.assertEqual("unknown", result.intent)
        self.assertEqual(NEXT_ACTION_ASK_USER, result.next_action)
        self.assertEqual(["了解理财产品", "查询业务费用", "联系人工客服"], result.suggested_options)

    def test_unknown_slots_are_removed_by_allowlist(self) -> None:
        extractor = FakeExtractor(ExtractionResult(
            intent="fee_query",
            confidence=0.9,
            slots={"business_type": "跨行转账", "admin": True},
        ))
        result = RequirementAnalysisService(extractor, default_workflows()).analyze("跨行转账怎么收费")

        self.assertEqual({"business_type": "跨行转账"}, result.slots)
        self.assertEqual(ANALYSIS_STATUS_READY, result.status)

    def test_invalid_enum_and_number_values_remain_missing(self) -> None:
        extractor = FakeExtractor(ExtractionResult(
            intent="product_recommendation",
            confidence=0.9,
            slots={
                "risk_tolerance": "随便",
                "investment_amount": -1,
                "investment_period_months": "半年",
            },
        ))
        result = RequirementAnalysisService(extractor, default_workflows()).analyze("随便买一点")

        self.assertEqual({}, result.slots)
        self.assertEqual(
            ["risk_tolerance", "investment_amount", "investment_period_months"],
            result.missing_slots,
        )

    def test_follow_up_merges_existing_slots(self) -> None:
        extractor = FakeExtractor(ExtractionResult(
            intent="product_recommendation",
            confidence=0.95,
            slots={"risk_tolerance": "conservative"},
        ))
        result = RequirementAnalysisService(extractor, default_workflows()).analyze(
            "我是稳健型",
            current_intent="product_recommendation",
            existing_slots={"investment_amount": 50000},
        )

        self.assertEqual("conservative", result.slots["risk_tolerance"])
        self.assertEqual(50000, result.slots["investment_amount"])
        self.assertEqual(["investment_period_months"], result.missing_slots)
        self.assertIn("多长时间", result.next_question)

    def test_empty_input_is_rejected_before_model_call(self) -> None:
        extractor = FakeExtractor(ExtractionResult(intent="human_service", confidence=1, slots={}))
        with self.assertRaisesRegex(ValueError, "用户输入不能为空"):
            RequirementAnalysisService(extractor, default_workflows()).analyze("   ")


class DeepSeekRequirementExtractorTest(unittest.TestCase):
    def test_parse_plain_json(self) -> None:
        result = DeepSeekRequirementExtractor.parse_response(
            '{"intent":"fee_query","confidence":0.88,"slots":{"business_type":"跨行转账"}}'
        )
        self.assertEqual("fee_query", result.intent)
        self.assertEqual("跨行转账", result.slots["business_type"])

    def test_parse_markdown_json_fence(self) -> None:
        result = DeepSeekRequirementExtractor.parse_response(
            '```json\n{"intent":"human_service","confidence":0.99,"slots":{}}\n```'
        )
        self.assertEqual("human_service", result.intent)

    def test_invalid_slots_type_is_rejected(self) -> None:
        with self.assertRaisesRegex(ValueError, "slots 无效"):
            DeepSeekRequirementExtractor.parse_response(
                '{"intent":"fee_query","confidence":0.8,"slots":[]}'
            )


class QuestionPolicyTest(unittest.TestCase):
    def test_last_question_is_not_repeated_when_another_required_slot_exists(self) -> None:
        workflow = next(item for item in default_workflows() if item.intent == "product_recommendation")
        selected = QuestionPolicy().select(
            workflow,
            ["investment_amount", "investment_period_months"],
            "还没有想好",
            "这笔资金预计多长时间内不会使用？",
        )
        self.assertEqual("investment_amount", selected.name)


if __name__ == "__main__":
    unittest.main()
