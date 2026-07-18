from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

from model.agent_models import ModelUsage

# ANALYSIS_STATUS_NEED_CLARIFICATION 表示信息不足，需要继续向用户提问。
ANALYSIS_STATUS_NEED_CLARIFICATION = "need_clarification"
# ANALYSIS_STATUS_READY 表示必填参数完整，可以交给后续工作流处理。
ANALYSIS_STATUS_READY = "ready"

# NEXT_ACTION_ASK_USER 表示下一步只能向用户收集或确认信息。
NEXT_ACTION_ASK_USER = "ask_user"
# NEXT_ACTION_ROUTE_WORKFLOW 表示下一步可以路由到确定性业务工作流。
NEXT_ACTION_ROUTE_WORKFLOW = "route_workflow"

# SLOT_VALUE_TYPE_STRING 表示参数值必须是非空字符串。
SLOT_VALUE_TYPE_STRING = "string"
# SLOT_VALUE_TYPE_NUMBER 表示参数值必须是大于零的数字。
SLOT_VALUE_TYPE_NUMBER = "number"
# SLOT_VALUE_TYPE_INTEGER 表示参数值必须是大于零的整数。
SLOT_VALUE_TYPE_INTEGER = "integer"


@dataclass(frozen=True)
class SlotDefinition:
    """定义一个业务参数以及缺失时使用的固定追问方式。"""

    name: str  # 参数的稳定英文标识，供程序和模型共同引用。
    description: str  # 参数业务含义，用于指导模型正确抽取。
    required: bool  # 是否必须取得该参数后才能进入后续工作流。
    question: str  # 参数缺失时向用户提出的问题。
    options: list[str] = field(default_factory=list)  # 可选的固定选项；为空表示允许自由输入。
    value_type: str = SLOT_VALUE_TYPE_STRING  # 参数值类型，可选值为 string、number、integer。
    allowed_values: list[Any] = field(default_factory=list)  # 允许的规范值；为空表示不限制具体取值。
    priority: int = 0  # 缺失时的基础提问优先级；最终顺序还会由动态问题策略调整。


@dataclass(frozen=True)
class WorkflowDefinition:
    """定义一个允许识别的客服意图及其参数约束。"""

    intent: str  # 意图的稳定英文标识。
    description: str  # 意图对应的业务场景说明。
    slots: list[SlotDefinition] = field(default_factory=list)  # 该意图允许使用的参数定义，顺序也是追问顺序。


@dataclass(frozen=True)
class ExtractionResult:
    """表示语言模型对用户原话的结构化理解，不包含执行决定。"""

    intent: str  # 模型识别出的意图标识。
    confidence: float  # 意图置信度，范围为 0 到 1。
    slots: dict[str, Any] = field(default_factory=dict)  # 模型从当前输入中提取出的参数值。
    usage: ModelUsage | None = None  # 本次结构化提取消耗的模型计量；纯测试提取器可以不提供。


@dataclass(frozen=True)
class RequirementAnalysisResult:
    """表示确定性规则校验后的需求拆解和下一步建议。"""

    intent: str  # 最终接受的意图；无法可靠判断时为 unknown。
    confidence: float  # 原始意图识别置信度，范围为 0 到 1。
    slots: dict[str, Any]  # 经过白名单过滤的有效参数。
    missing_slots: list[str]  # 仍未取得的必填参数，顺序与工作流定义一致。
    status: str  # 分析状态，可选值为 ANALYSIS_STATUS_NEED_CLARIFICATION、ANALYSIS_STATUS_READY。
    next_action: str  # 下一动作，可选值为 NEXT_ACTION_ASK_USER、NEXT_ACTION_ROUTE_WORKFLOW。
    next_question: str  # 需要追问时的固定问题；无需追问时为空字符串。
    suggested_options: list[str]  # 下一问可展示的固定选项；允许自由输入时为空列表。
    usage: ModelUsage | None = None  # 需求理解阶段对应的模型计量。
