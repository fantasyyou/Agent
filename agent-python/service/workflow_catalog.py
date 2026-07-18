from __future__ import annotations

from model.requirement_analysis_models import (
    SLOT_VALUE_TYPE_INTEGER,
    SLOT_VALUE_TYPE_NUMBER,
    SLOT_VALUE_TYPE_STRING,
    SlotDefinition,
    WorkflowDefinition,
)


def default_workflows() -> list[WorkflowDefinition]:
    """返回第一阶段允许识别的最小金融客服工作流目录。"""

    return [
        WorkflowDefinition(
            intent="product_recommendation",
            description="用户希望根据金额、期限和风险承受能力筛选理财产品。",
            slots=[
                SlotDefinition(
                    name="risk_tolerance",
                    description="用户风险承受能力，取值为 conservative、balanced、aggressive。",
                    required=True,
                    question="请问您的风险承受能力属于稳健型、平衡型还是进取型？",
                    options=["稳健型", "平衡型", "进取型", "暂不确定"],
                    allowed_values=["conservative", "balanced", "aggressive", "unknown"],
                    priority=100,
                ),
                SlotDefinition(
                    name="investment_amount",
                    description="用户计划投入的金额，单位为人民币元。",
                    required=True,
                    question="您计划投入的大致金额是多少？",
                    value_type=SLOT_VALUE_TYPE_NUMBER,
                    priority=70,
                ),
                SlotDefinition(
                    name="investment_period_months",
                    description="资金预计可以持续投资的月数。",
                    required=True,
                    question="这笔资金预计多长时间内不会使用？",
                    options=["3个月以内", "3至6个月", "6至12个月", "1年以上"],
                    value_type=SLOT_VALUE_TYPE_INTEGER,
                    priority=80,
                ),
                SlotDefinition(
                    name="liquidity_requirement",
                    description="用户对资金取用和赎回速度的要求，取值为 high、medium、low。",
                    required=False,
                    question="您是否需要随时取用这笔资金，还是可以接受一定的赎回等待时间？",
                    options=["需要随时取用", "可接受1至7天到账", "没有特别要求"],
                    value_type=SLOT_VALUE_TYPE_STRING,
                    allowed_values=["high", "medium", "low"],
                    priority=60,
                ),
            ],
        ),
        WorkflowDefinition(
            intent="fee_query",
            description="用户希望查询某项金融业务的收费标准。",
            slots=[
                SlotDefinition(
                    name="business_type",
                    description="需要查询费用的业务名称，例如跨行转账、银行卡挂失。",
                    required=True,
                    question="请问您想查询哪项业务的收费标准？",
                    priority=100,
                )
            ],
        ),
        WorkflowDefinition(
            intent="human_service",
            description="用户明确要求联系或转接人工客服。",
        ),
        WorkflowDefinition(
            intent="financial_consultation",
            description="用户咨询金融概念、市场主体或一般金融知识，不要求办理具体业务。",
        ),
    ]
