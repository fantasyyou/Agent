from __future__ import annotations

from model.requirement_analysis_models import SlotDefinition, WorkflowDefinition


class QuestionPolicy:
    """根据当前表达和会话状态，从缺失参数中确定最有价值的下一问。"""

    _KEYWORDS = {
        "risk_tolerance": ("风险", "稳健", "稳定", "保本", "亏损", "波动"),
        "investment_amount": ("金额", "预算", "投入", "资金", "元", "万"),
        "investment_period_months": ("期限", "多久", "长期", "短期", "半年", "一年", "不用"),
        "liquidity_requirement": ("急用", "随时", "赎回", "取用", "流动", "到账"),
    }

    def select(
        self,
        workflow: WorkflowDefinition,
        missing_slot_names: list[str],
        user_text: str,
        last_question: str = "",
        active_slot: str = "",
    ) -> SlotDefinition:
        """返回得分最高的缺失参数；同分时保持工作流定义顺序。"""

        definitions = {slot.name: slot for slot in workflow.slots}
        candidates = [definitions[name] for name in missing_slot_names if name in definitions]
        if not candidates:
            raise ValueError("没有可提问的缺失参数")

        # 上一问对应的参数仍然缺失，说明用户回答未被有效识别，应继续澄清该参数。
        # 只有上一问已经取得有效值、从缺失列表移除后，才选择下一个问题。
        for candidate in candidates:
            if active_slot and active_slot == candidate.name:
                return candidate
        for candidate in candidates:
            if last_question and last_question == candidate.question:
                return candidate

        def score(slot: SlotDefinition) -> int:
            value = slot.priority
            if any(keyword in user_text for keyword in self._KEYWORDS.get(slot.name, ())):
                value += 25
            return value

        return max(candidates, key=score)
