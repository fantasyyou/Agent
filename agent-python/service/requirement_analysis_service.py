from __future__ import annotations

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
        slots = {key: value for key, value in combined_slots.items() if key in definitions and _valid_slot_value(definitions[key], value)}
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


def _valid_slot_value(definition: SlotDefinition, value) -> bool:
    """按工作流定义校验模型提取值，避免仅凭字段存在就进入执行流程。"""

    if value in (None, "", []):
        return False
    if definition.value_type == SLOT_VALUE_TYPE_STRING:
        valid_type = isinstance(value, str)
    elif definition.value_type == SLOT_VALUE_TYPE_NUMBER:
        valid_type = isinstance(value, (int, float)) and not isinstance(value, bool) and value > 0
    elif definition.value_type == SLOT_VALUE_TYPE_INTEGER:
        valid_type = isinstance(value, int) and not isinstance(value, bool) and value > 0
    else:
        return False
    return valid_type and (not definition.allowed_values or value in definition.allowed_values)
