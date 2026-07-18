from __future__ import annotations

import math
import re
from typing import Protocol

from model.requirement_analysis_models import (
    ANALYSIS_STATUS_NEED_CLARIFICATION,
    ANALYSIS_STATUS_READY,
    NEXT_ACTION_ASK_USER,
    NEXT_ACTION_ROUTE_WORKFLOW,
    ExtractionResult,
    RequirementAnalysisResult,
    SLOT_VALUE_TYPE_INTEGER,
    SLOT_VALUE_TYPE_NUMBER,
    SLOT_VALUE_TYPE_STRING,
    SlotDefinition,
    WorkflowDefinition,
)
from service.question_policy import QuestionPolicy


class RequirementExtractor(Protocol):
    """定义自然语言结构化提取器必须实现的接口。"""

    def extract(self, user_text: str, workflows: list[WorkflowDefinition]) -> ExtractionResult:
        """从用户原话中提取意图、置信度和参数，不执行任何业务动作。"""


class RequirementAnalysisService:
    """组合动态语言理解与确定性参数校验，生成下一问或工作流路由建议。"""

    def __init__(
        self,
        extractor: RequirementExtractor,
        workflows: list[WorkflowDefinition],
        confidence_threshold: float = 0.65,
        question_policy: QuestionPolicy | None = None,
    ) -> None:
        self.extractor = extractor
        self.workflows = {workflow.intent: workflow for workflow in workflows}
        self.confidence_threshold = confidence_threshold
        self.question_policy = question_policy or QuestionPolicy()

    def analyze(
        self,
        user_text: str,
        current_intent: str | None = None,
        existing_slots: dict | None = None,
        last_question: str = "",
    ) -> RequirementAnalysisResult:
        """拆解一轮输入；追问场景可传入当前意图和之前已经收集的参数。"""

        text = user_text.strip()
        if not text:
            raise ValueError("用户输入不能为空")
        active_workflow = self.workflows.get(current_intent) if current_intent else None
        candidate_workflows = [active_workflow] if active_workflow else list(self.workflows.values())
        extraction = self.extractor.extract(text, candidate_workflows)
        confidence = min(max(float(extraction.confidence), 0.0), 1.0)
        workflow = active_workflow or self.workflows.get(extraction.intent)
        if workflow is None or (active_workflow is None and confidence < self.confidence_threshold):
            return RequirementAnalysisResult(
                intent="unknown",
                confidence=confidence,
                slots={},
                missing_slots=[],
                status=ANALYSIS_STATUS_NEED_CLARIFICATION,
                next_action=NEXT_ACTION_ASK_USER,
                next_question="为了准确理解您的需求，请问您是想了解理财产品、查询业务费用，还是联系人工客服？",
                suggested_options=["了解理财产品", "查询业务费用", "联系人工客服"],
                usage=extraction.usage,
            )

        definitions = {definition.name: definition for definition in workflow.slots}
        combined_slots = dict(existing_slots or {})
        combined_slots.update(extraction.slots)
        slots = {}
        for key, value in combined_slots.items():
            definition = definitions.get(key)
            if definition is None:
                continue
            valid, normalized = _normalize_slot_value(definition, value)
            if valid:
                slots[key] = normalized
        missing_definitions = [
            definition
            for definition in workflow.slots
            if definition.required and definition.name not in slots
        ]
        if missing_definitions:
            selected = self.question_policy.select(
                workflow,
                [definition.name for definition in missing_definitions],
                text,
                last_question,
            )
            return RequirementAnalysisResult(
                intent=workflow.intent,
                confidence=confidence,
                slots=slots,
                missing_slots=[definition.name for definition in missing_definitions],
                status=ANALYSIS_STATUS_NEED_CLARIFICATION,
                next_action=NEXT_ACTION_ASK_USER,
                next_question=selected.question,
                suggested_options=list(selected.options),
                usage=extraction.usage,
            )

        return RequirementAnalysisResult(
            intent=workflow.intent,
            confidence=confidence,
            slots=slots,
            missing_slots=[],
            status=ANALYSIS_STATUS_READY,
            next_action=NEXT_ACTION_ROUTE_WORKFLOW,
            next_question="",
            suggested_options=[],
            usage=extraction.usage,
        )


def _normalize_slot_value(definition: SlotDefinition, value) -> tuple[bool, object]:
    """校验并规范化模型或跨服务状态中的参数值。"""

    if value in (None, "", []):
        return False, value
    if definition.value_type == SLOT_VALUE_TYPE_STRING:
        normalized = value.strip() if isinstance(value, str) else value
        valid_type = isinstance(normalized, str) and bool(normalized)
    elif definition.value_type == SLOT_VALUE_TYPE_NUMBER:
        normalized = _parse_positive_number(value)
        valid_type = normalized is not None
    elif definition.value_type == SLOT_VALUE_TYPE_INTEGER:
        normalized = _parse_positive_integer(value)
        valid_type = normalized is not None
    else:
        return False, value
    if not valid_type or (definition.allowed_values and normalized not in definition.allowed_values):
        return False, value
    return True, normalized


def _parse_positive_integer(value) -> int | None:
    """接受protobuf产生的3.0及“3个月”“半年”“1年”等期限表达。"""

    if isinstance(value, bool):
        return None
    if isinstance(value, (int, float)):
        number = float(value)
    elif isinstance(value, str):
        text = value.strip().replace(" ", "")
        if text == "半年":
            return 6
        year_match = re.fullmatch(r"(\d+(?:\.\d+)?)年", text)
        if year_match:
            number = float(year_match.group(1)) * 12
        else:
            month_match = re.fullmatch(r"(\d+(?:\.\d+)?)(?:个?月)?", text)
            if not month_match:
                return None
            number = float(month_match.group(1))
    else:
        return None
    if not math.isfinite(number) or number <= 0 or not number.is_integer():
        return None
    return int(number)


def _parse_positive_number(value) -> int | float | None:
    """接受普通数字和“20万”“200000元”等常见人民币金额表达。"""

    if isinstance(value, bool):
        return None
    if isinstance(value, (int, float)):
        number = float(value)
    elif isinstance(value, str):
        text = value.strip().replace(",", "").replace("，", "").replace(" ", "")
        match = re.fullmatch(
            r"(?:人民币)?([0-9]+(?:\.[0-9]+)?)([千万元亿]?)(?:人民币)?(?:元)?(?:左右)?",
            text,
        )
        if not match:
            return None
        multiplier = {"": 1, "千": 1_000, "万": 10_000, "万元": 10_000, "亿": 100_000_000}[match.group(2)]
        number = float(match.group(1)) * multiplier
    else:
        return None
    if not math.isfinite(number) or number <= 0:
        return None
    return int(number) if number.is_integer() else number
