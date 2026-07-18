from __future__ import annotations

import json
from typing import Any

from client.deepseek_client import DeepSeekClient
from model.requirement_analysis_models import ExtractionResult, WorkflowDefinition


class DeepSeekRequirementExtractor:
    """使用 DeepSeek 将用户原话转换为结构化意图和参数，不执行任何业务动作。"""

    def __init__(self, client: DeepSeekClient) -> None:
        self.client = client

    def extract(self, user_text: str, workflows: list[WorkflowDefinition]) -> ExtractionResult:
        """调用模型完成结构化提取，并严格校验返回字段类型。"""

        catalog = [
            {
                "intent": workflow.intent,
                "description": workflow.description,
                "slots": [
                    {
                        "name": slot.name,
                        "description": slot.description,
                        "value_type": slot.value_type,
                        "allowed_values": slot.allowed_values,
                    }
                    for slot in workflow.slots
                ],
            }
            for workflow in workflows
        ]
        system_prompt = (
            "你是金融客服需求解析器，只负责理解，不得回答问题或执行工具。"
            "只能从给定意图中选择一个；不能判断时使用 unknown。"
            "只输出一个 JSON 对象，不要输出 Markdown。JSON 字段必须为 intent、confidence、slots。"
            "confidence 必须是 0 到 1 的数字，slots 必须是对象，不得编造用户没有表达的参数。"
            "必须严格按照参数的 value_type 和 allowed_values 返回规范值；金额统一换算成人民币元数字，"
            "投资期限统一换算成正整数月数，例如20万返回200000，3个月返回3。"
        )
        user_prompt = (
            f"允许的意图和参数：{json.dumps(catalog, ensure_ascii=False)}\n"
            f"用户原话：{user_text}"
        )
        response = self.client.complete(system_prompt, user_prompt)
        result = self.parse_response(response.answer)
        return ExtractionResult(
            intent=result.intent,
            confidence=result.confidence,
            slots=result.slots,
            usage=response.usage,
        )

    @staticmethod
    def parse_response(content: str) -> ExtractionResult:
        """解析模型返回的 JSON；支持模型偶尔附加的 Markdown 代码围栏。"""

        payload = _load_json_object(content)
        intent = payload.get("intent")
        confidence = payload.get("confidence")
        slots = payload.get("slots", {})
        if not isinstance(intent, str) or not intent:
            raise ValueError("模型返回的 intent 无效")
        if not isinstance(confidence, (int, float)) or isinstance(confidence, bool):
            raise ValueError("模型返回的 confidence 无效")
        if not isinstance(slots, dict):
            raise ValueError("模型返回的 slots 无效")
        return ExtractionResult(intent=intent, confidence=float(confidence), slots=slots)


def _load_json_object(content: str) -> dict[str, Any]:
    """从纯 JSON 或 Markdown 代码围栏中读取一个 JSON 对象。"""

    text = content.strip()
    if text.startswith("```"):
        lines = text.splitlines()
        if lines and lines[0].startswith("```"):
            lines = lines[1:]
        if lines and lines[-1].strip() == "```":
            lines = lines[:-1]
        text = "\n".join(lines).strip()
    data = json.loads(text)
    if not isinstance(data, dict):
        raise ValueError("模型必须返回 JSON 对象")
    return data
